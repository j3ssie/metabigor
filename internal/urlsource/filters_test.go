package urlsource

import (
	"net/url"
	"slices"
	"strings"
	"testing"
)

// queryOf parses the "&k=v" suffix the param builders return.
func queryOf(t *testing.T, params string) url.Values {
	t.Helper()
	values, err := url.ParseQuery(strings.TrimPrefix(params, "&"))
	if err != nil {
		t.Fatalf("ParseQuery(%q): %v", params, err)
	}
	return values
}

func compiled(t *testing.T, f Filters) Filters {
	t.Helper()
	if err := f.Compile(); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	return f
}

// warc/revisit records only mean "unchanged since the last capture", so every
// exclusion list drops them even when the user set no MIME filter.
func TestWaybackParamsAlwaysExcludesRevisit(t *testing.T) {
	f := compiled(t, Filters{})
	got := queryOf(t, f.WaybackParams())

	filters := got["filter"]
	if !slices.Contains(filters, "!mimetype:"+revisitMIME) {
		t.Errorf("filter = %v, want it to contain !mimetype:%s", filters, revisitMIME)
	}
	if len(got["from"]) != 0 || len(got["to"]) != 0 {
		t.Errorf("from/to should be absent without --from/--to, got %v %v", got["from"], got["to"])
	}
}

func TestWaybackParams(t *testing.T) {
	f := compiled(t, Filters{
		FilterMIME:   []string{"text/css", "image/svg+xml"},
		FilterStatus: []string{"404", "301"},
		From:         "2023",
		To:           "202401",
	})
	got := queryOf(t, f.WaybackParams())

	// An escaped plus is rewritten to a dot: the CDX API rejects `\+`, which
	// would otherwise break every MIME type containing one.
	wantMIME := "!mimetype:warc/revisit|text/css|image/svg.xml"
	if !slices.Contains(got["filter"], wantMIME) {
		t.Errorf("filter = %v, want it to contain %q", got["filter"], wantMIME)
	}
	if !slices.Contains(got["filter"], "!statuscode:404|301") {
		t.Errorf("filter = %v, want it to contain !statuscode:404|301", got["filter"])
	}
	if got.Get("from") != "2023" || got.Get("to") != "202401" {
		t.Errorf("from/to = %q/%q, want 2023/202401", got.Get("from"), got.Get("to"))
	}
}

// Match and Filter cannot both be expressed by the archive APIs, so Match wins.
func TestWaybackParamsMatchWins(t *testing.T) {
	f := compiled(t, Filters{
		MatchMIME:    []string{"text/html"},
		FilterMIME:   []string{"text/css"},
		MatchStatus:  []string{"200"},
		FilterStatus: []string{"404"},
	})
	got := queryOf(t, f.WaybackParams())

	if !slices.Contains(got["filter"], "mimetype:text/html") {
		t.Errorf("filter = %v, want it to contain mimetype:text/html", got["filter"])
	}
	if !slices.Contains(got["filter"], "statuscode:200") {
		t.Errorf("filter = %v, want it to contain statuscode:200", got["filter"])
	}
	for _, value := range got["filter"] {
		if strings.HasPrefix(value, "!") {
			t.Errorf("filter = %v, want no negated filter once a match is set", got["filter"])
		}
	}
}

// Common Crawl uses a different dialect for the same conditions: the regex is
// parenthesised and marked with a leading tilde.
func TestCommonCrawlParams(t *testing.T) {
	f := compiled(t, Filters{
		FilterMIME:   []string{"text/css"},
		FilterStatus: []string{"404"},
		From:         "2023",
	})
	got := queryOf(t, f.CommonCrawlParams())

	if !slices.Contains(got["filter"], "!~mime:(warc/revisit|text/css)") {
		t.Errorf("filter = %v, want it to contain !~mime:(warc/revisit|text/css)", got["filter"])
	}
	if !slices.Contains(got["filter"], "!~status:(404)") {
		t.Errorf("filter = %v, want it to contain !~status:(404)", got["filter"])
	}
	// Common Crawl has no date parameters; dates are handled by skipping index
	// collections and re-checking each record locally.
	if len(got["from"]) != 0 || len(got["to"]) != 0 {
		t.Errorf("from/to = %v/%v, want neither: Common Crawl has no date params", got["from"], got["to"])
	}
}

func TestCommonCrawlParamsMatchWins(t *testing.T) {
	f := compiled(t, Filters{MatchMIME: []string{"text/html"}, MatchStatus: []string{"200"}})
	got := queryOf(t, f.CommonCrawlParams())

	if !slices.Contains(got["filter"], "~mime:(text/html)") {
		t.Errorf("filter = %v, want it to contain ~mime:(text/html)", got["filter"])
	}
	if !slices.Contains(got["filter"], "~status:(200)") {
		t.Errorf("filter = %v, want it to contain ~status:(200)", got["filter"])
	}
}

func TestKeywordPushdown(t *testing.T) {
	wayback := compiled(t, Filters{Keywords: `\.js`})
	if got := queryOf(t, wayback.WaybackParams())["filter"]; !slices.Contains(got, `original:.*(\.js).*`) {
		t.Errorf("filter = %v, want it to contain original:.*(\\.js).*", got)
	}

	cc := compiled(t, Filters{Keywords: `\.js`})
	if got := queryOf(t, cc.CommonCrawlParams())["filter"]; !slices.Contains(got, `~url:.*(\.js).*`) {
		t.Errorf("filter = %v, want it to contain ~url:.*(\\.js).*", got)
	}
}

// The CDX filter matches with re.match semantics: anchored at the start,
// unanchored at the end. A leading .* is always needed; a trailing .* has to be
// left off when the pattern already anchors the end, or nothing can match.
func TestCDXKeywordPattern(t *testing.T) {
	tests := map[string]string{
		`\.js`:  `.*(\.js).*`,
		`\.js$`: `.*(\.js$)`,
		`admin`: `.*(admin).*`,
		// The $ inside the alternation still anchors the end, so no suffix.
		`\.js(\?.*|$)`: `.*(\.js(\?.*|$))`,
		// An escaped dollar is a literal, not an anchor, so the suffix stays.
		`price\$`: `.*(price\$).*`,
	}

	for pattern, want := range tests {
		if got := cdxKeywordPattern(pattern); got != want {
			t.Errorf("cdxKeywordPattern(%q) = %q, want %q", pattern, got, want)
		}
	}
}

func TestHasUnescapedDollar(t *testing.T) {
	tests := map[string]bool{
		`abc$`:    true,
		`abc`:     false,
		`abc\$`:   false,
		`abc\\$`:  true, // escaped backslash, then a real anchor
		`a$b`:     true,
		`abc\\\$`: false,
	}

	for pattern, want := range tests {
		if got := hasUnescapedDollar(pattern); got != want {
			t.Errorf("hasUnescapedDollar(%q) = %v, want %v", pattern, got, want)
		}
	}
}

func TestCompileRejectsBadInput(t *testing.T) {
	tests := []struct {
		name    string
		filters Filters
	}{
		// The usual cause is bash expanding $) inside double quotes.
		{"broken regex", Filters{Keywords: `\.js(\?|`}},
		{"non-numeric from", Filters{From: "last-year"}},
		{"overlong to", Filters{To: "202401011200000"}},
		{"inverted range", Filters{From: "2024", To: "2023"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.filters
			if err := f.Compile(); err == nil {
				t.Error("Compile() = nil, want an error")
			}
		})
	}
}

// A from-date pads to the earliest instant in the period and a to-date to the
// latest, so --from 2023 --to 2023 covers all of 2023.
func TestAllowTimestamp(t *testing.T) {
	f := compiled(t, Filters{From: "2023", To: "2023"})

	tests := map[string]bool{
		"20230101000000": true,
		"20231231235959": true,
		"20221231235959": false,
		"20240101000000": false,
		// No timestamp is not evidence the URL is out of range.
		"": true,
	}

	for ts, want := range tests {
		if got := f.AllowTimestamp(ts); got != want {
			t.Errorf("AllowTimestamp(%q) = %v, want %v", ts, got, want)
		}
	}
}

func TestAllowTimestampUnbounded(t *testing.T) {
	f := compiled(t, Filters{})
	if !f.AllowTimestamp("19990101000000") {
		t.Error("AllowTimestamp should accept anything when no range is set")
	}
	if f.HasDateRange() {
		t.Error("HasDateRange() = true, want false without --from/--to")
	}
}

func TestFilterYears(t *testing.T) {
	f := compiled(t, Filters{From: "2016", To: "202312"})
	if got := f.FromYear(); got != 2016 {
		t.Errorf("FromYear() = %d, want 2016", got)
	}
	if got := f.ToYear(); got != 2023 {
		t.Errorf("ToYear() = %d, want 2023", got)
	}

	empty := compiled(t, Filters{})
	if got := empty.FromYear(); got != 0 {
		t.Errorf("FromYear() = %d, want 0 when unset", got)
	}
}

func TestAllow(t *testing.T) {
	tests := []struct {
		name    string
		filters Filters
		url     string
		status  string
		mime    string
		want    bool
	}{
		{"no filters keeps everything", Filters{}, "https://example.com/a", "200", "text/html", true},
		{"status excluded", Filters{FilterStatus: []string{"404"}}, "https://example.com/a", "404", "", false},
		{"status kept", Filters{FilterStatus: []string{"404"}}, "https://example.com/a", "200", "", true},
		{"status match", Filters{MatchStatus: []string{"200"}}, "https://example.com/a", "301", "", false},
		{"mime excluded", Filters{FilterMIME: []string{"text/css"}}, "https://example.com/a.css", "", "text/css", false},
		{"mime match", Filters{MatchMIME: []string{"text/html"}}, "https://example.com/a", "", "application/json", false},
		{"mime case-insensitive", Filters{FilterMIME: []string{"text/css"}}, "https://example.com/a", "", "TEXT/CSS", false},
		// A field the source did not report is not evidence against the URL.
		{"unknown status kept", Filters{FilterStatus: []string{"404"}}, "https://example.com/a", "", "", true},
		{"unknown mime kept", Filters{FilterMIME: []string{"text/css"}}, "https://example.com/a", "", "", true},
		// warc/revisit is always dropped.
		{"revisit dropped", Filters{}, "https://example.com/a", "", revisitMIME, false},
		{"extension blacklisted", Filters{Blacklist: []string{"png"}}, "https://example.com/a.png", "", "", false},
		{"extension with dot", Filters{Blacklist: []string{".png"}}, "https://example.com/a.png", "", "", false},
		{"extension with query", Filters{Blacklist: []string{"png"}}, "https://example.com/a.png?v=2", "", "", false},
		{"extension not blacklisted", Filters{Blacklist: []string{"png"}}, "https://example.com/a.js", "", "", true},
		{"default blacklist expands", Filters{Blacklist: []string{"default"}}, "https://example.com/a.woff2", "", "", false},
		{"keyword matches", Filters{Keywords: `\.js$`}, "https://example.com/a.js", "", "", true},
		{"keyword misses", Filters{Keywords: `\.js$`}, "https://example.com/a.html", "", "", false},
		{"keyword case-insensitive", Filters{Keywords: "admin"}, "https://example.com/ADMIN", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := compiled(t, tc.filters)
			if got := f.Allow(tc.url, tc.status, tc.mime); got != tc.want {
				t.Errorf("Allow(%q, %q, %q) = %v, want %v", tc.url, tc.status, tc.mime, got, tc.want)
			}
		})
	}
}

// A single flag value may carry a comma-separated list, since pflag splits on
// commas only for some shells' quoting.
func TestAllowSplitsCommaLists(t *testing.T) {
	f := compiled(t, Filters{FilterStatus: []string{"404,301"}})
	if f.Allow("https://example.com/a", "301", "") {
		t.Error("Allow should reject 301 from a comma-separated --filter-status")
	}
}

func TestNormalize(t *testing.T) {
	tests := map[string]string{
		"https://example.com/a": "https://example.com/a",
		// archive.org appends an encoded newline plus junk; the URL only
		// resolves once that tail is cut.
		"https://example.com/a%0Ajunk": "https://example.com/a",
		"https://example.com/a%0ajunk": "https://example.com/a",
		// A default port splits one endpoint into two results.
		"http://example.com:80/a":  "http://example.com/a",
		"https://example.com:443/": "https://example.com/",
		// A non-default port is meaningful and stays.
		"https://example.com:8080/a": "https://example.com:8080/a",
		"  https://example.com/a  ":  "https://example.com/a",
		"":                           "",
	}

	for input, want := range tests {
		if got := Normalize(input); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

// --no-params collapses URLs that differ only in their query string.
func TestEndpointKey(t *testing.T) {
	a := EndpointKey("https://example.com/search?q=1")
	b := EndpointKey("https://example.com/search?q=2&page=3")
	if a != b {
		t.Errorf("EndpointKey should match for query-only differences: %q vs %q", a, b)
	}

	c := EndpointKey("https://example.com/other?q=1")
	if a == c {
		t.Errorf("EndpointKey should differ for different paths: %q vs %q", a, c)
	}
}

func TestTimestampFromISO(t *testing.T) {
	tests := map[string]string{
		"2021-03-04T05:06:07":      "20210304050607",
		"2021-03-04 05:06:07":      "20210304050607",
		"2021-03-04T05:06:07.123Z": "20210304050607",
		"2021-03-04":               "20210304000000",
		"":                         "",
		"not-a-date":               "",
	}

	for input, want := range tests {
		if got := TimestampFromISO(input); got != want {
			t.Errorf("TimestampFromISO(%q) = %q, want %q", input, got, want)
		}
	}
}
