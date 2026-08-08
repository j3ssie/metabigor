package urlsource

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

// The plain-text CDX form has no header row, unlike output=json, so every line
// is data.
func TestParseCDXLine(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		timestamp string
		url       string
		mime      string
		status    string
	}{
		{
			name:      "full row",
			line:      "20230104120000 https://example.com/a text/html 200",
			timestamp: "20230104120000",
			url:       "https://example.com/a",
			mime:      "text/html",
			status:    "200",
		},
		{
			name:      "missing trailing fields",
			line:      "20230104120000 https://example.com/a",
			timestamp: "20230104120000",
			url:       "https://example.com/a",
		},
		{
			name:      "extra whitespace",
			line:      "  20230104120000   https://example.com/a   text/html   200  ",
			timestamp: "20230104120000",
			url:       "https://example.com/a",
			mime:      "text/html",
			status:    "200",
		},
		{name: "blank line", line: ""},
		{name: "single field", line: "20230104120000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			timestamp, rawURL, mime, status := parseCDXLine(tc.line)
			if timestamp != tc.timestamp || rawURL != tc.url || mime != tc.mime || status != tc.status {
				t.Errorf("parseCDXLine(%q) = (%q, %q, %q, %q), want (%q, %q, %q, %q)",
					tc.line, timestamp, rawURL, mime, status, tc.timestamp, tc.url, tc.mime, tc.status)
			}
		})
	}
}

// A WARC holds every sub-request made while the page was archived. Only
// response records are taken: a request record names the same URL, and metadata
// records are the archive's own bookkeeping.
func TestParseWARCTargets(t *testing.T) {
	warc := strings.Join([]string{
		"WARC/1.0",
		"WARC-Type: warcinfo",
		"WARC-Date: 2024-01-01T00:00:00Z",
		"Content-Length: 0",
		"",
		"WARC/1.0",
		"WARC-Type: request",
		"WARC-Target-URI: https://example.com/never-from-a-request",
		"Content-Length: 0",
		"",
		"WARC/1.0",
		"WARC-Type: response",
		"WARC-Target-URI: https://example.com/",
		"Content-Type: application/http; msgtype=response",
		"",
		"HTTP/1.1 200 OK",
		"Content-Type: text/html",
		"",
		"<html><body>hello</body></html>",
		"WARC/1.0",
		"WARC-Type: response",
		"WARC-Target-URI: https://api.example.com/v1/users?id=1",
		"Content-Type: application/http; msgtype=response",
		"",
		"HTTP/1.1 200 OK",
		"Content-Type: application/json",
		"",
		`{"ok":true}`,
		"WARC/1.0",
		"WARC-Type: metadata",
		"WARC-Target-URI: https://example.com/metadata-only",
		"Content-Length: 0",
		"",
	}, "\n")

	got := parseWARCTargets([]byte(warc))
	want := []string{"https://example.com/", "https://api.example.com/v1/users?id=1"}
	if !slices.Equal(got, want) {
		t.Errorf("parseWARCTargets() = %v, want %v", got, want)
	}
}

// WARC header order is not guaranteed, so the target URI may precede the type.
func TestParseWARCTargetsHeaderOrder(t *testing.T) {
	warc := strings.Join([]string{
		"WARC/1.0",
		"WARC-Target-URI: https://example.com/a.js",
		"WARC-Type: response",
		"Content-Length: 0",
		"",
	}, "\n")

	got := parseWARCTargets([]byte(warc))
	if want := []string{"https://example.com/a.js"}; !slices.Equal(got, want) {
		t.Errorf("parseWARCTargets() = %v, want %v", got, want)
	}
}

func TestParseWARCTargetsDeduplicates(t *testing.T) {
	record := "WARC/1.0\nWARC-Type: response\nWARC-Target-URI: https://example.com/a\n\n"
	got := parseWARCTargets([]byte(record + record + record))

	if want := []string{"https://example.com/a"}; !slices.Equal(got, want) {
		t.Errorf("parseWARCTargets() = %v, want %v", got, want)
	}
}

// A response body that happens to contain a WARC-looking line must not add a
// second URL to the record.
func TestParseWARCTargetsIgnoresPayloadHeaders(t *testing.T) {
	warc := strings.Join([]string{
		"WARC/1.0",
		"WARC-Type: response",
		"WARC-Target-URI: https://example.com/real",
		"",
		"HTTP/1.1 200 OK",
		"",
		"WARC-Target-URI: https://evil.example.net/injected",
		"",
	}, "\n")

	got := parseWARCTargets([]byte(warc))
	if want := []string{"https://example.com/real"}; !slices.Equal(got, want) {
		t.Errorf("parseWARCTargets() = %v, want %v", got, want)
	}
}

func TestParseWARCTargetsEmpty(t *testing.T) {
	if got := parseWARCTargets(nil); len(got) != 0 {
		t.Errorf("parseWARCTargets(nil) = %v, want empty", got)
	}
	if got := parseWARCTargets([]byte("not a warc at all")); len(got) != 0 {
		t.Errorf("parseWARCTargets(garbage) = %v, want empty", got)
	}
}

// The search page rows carry both the archive path, needed to fetch the WARC,
// and the URL that was archived.
func TestArchiveLinkPattern(t *testing.T) {
	html := `<tr><td><a href="/archive/gkOOR">https://example.com/login</a></td></tr>` +
		`<tr><td><a href="/archive/xY12z">https://blog.example.com/</a></td></tr>` +
		`<a href="/about">not an archive row</a>`

	matches := archiveLinkPattern.FindAllStringSubmatch(html, -1)
	if len(matches) != 2 {
		t.Fatalf("found %d rows, want 2", len(matches))
	}
	if matches[0][1] != "/archive/gkOOR" || matches[0][2] != "https://example.com/login" {
		t.Errorf("first row = (%q, %q), want (/archive/gkOOR, https://example.com/login)", matches[0][1], matches[0][2])
	}
	if matches[1][1] != "/archive/xY12z" {
		t.Errorf("second archive path = %q, want /archive/xY12z", matches[1][1])
	}
}

// The real showNumPages response, captured from index.commoncrawl.org.
func TestParseCommonCrawlPageCount(t *testing.T) {
	tests := map[string]int{
		`{"pages": 2, "pageSize": 5, "blocks": 9}`: 2,
		`{"pages": 0, "pageSize": 5, "blocks": 0}`: 0,
		"7":             7,
		"  12  \n":      12,
		"":              0,
		"- - - -":       0,
		"<html>502":     0,
		`{"error":"x"}`: 0,
	}

	for body, want := range tests {
		if got := parseCommonCrawlPageCount(body); got != want {
			t.Errorf("parseCommonCrawlPageCount(%q) = %d, want %d", body, got, want)
		}
	}
}

// Records arrive as NDJSON, one JSON object per line.
func TestParseCommonCrawlBody(t *testing.T) {
	r := newTestRun(t, "tesla.com", Config{})

	body := strings.Join([]string{
		`{"urlkey":"com,tesla)/","timestamp":"20240815120000","url":"https://www.tesla.com/","mime":"text/html","status":"200"}`,
		`{"urlkey":"com,tesla)/model3","timestamp":"20240815130000","url":"https://www.tesla.com/model3","mime":"text/html","status":"200"}`,
		// An error record is skipped rather than aborting the page.
		`{"error":"something went wrong"}`,
		// So is a malformed line.
		`not json at all`,
		``,
	}, "\n")

	if err := r.parseCommonCrawlBody([]byte(body), 200); err != nil {
		t.Fatalf("parseCommonCrawlBody: %v", err)
	}

	results := r.snapshot()
	if len(results) != 2 {
		t.Fatalf("got %d results (%v), want 2", len(results), results)
	}
	if results[0].MIME != "text/html" || results[0].Status != "200" {
		t.Errorf("metadata not captured: %+v", results[0])
	}
	if results[0].Timestamp != "20240815120000" {
		t.Errorf("timestamp = %q, want 20240815120000", results[0].Timestamp)
	}
}

// A collection with nothing for the scope answers 404 with a plain-text notice.
// That is an empty result, not a failure.
func TestParseCommonCrawlBodyEmptyResult(t *testing.T) {
	r := newTestRun(t, "tesla.com", Config{})

	if err := r.parseCommonCrawlBody([]byte("No Captures found for: *.tesla.com/*"), 404); err != nil {
		t.Errorf("a 404 no-captures body should not be an error, got %v", err)
	}
	if err := r.parseCommonCrawlBody([]byte("No Captures found for: x"), 200); err != nil {
		t.Errorf("a no-captures body should not be an error, got %v", err)
	}
	if err := r.parseCommonCrawlBody([]byte("gateway timeout"), 502); err == nil {
		t.Error("a 502 should be reported as an error")
	}
}

func TestCollectionYear(t *testing.T) {
	tests := []struct {
		collection collection
		want       int
	}{
		{collection{ID: "CC-MAIN-2024-33"}, 2024},
		{collection{ID: "CC-MAIN-2008-2009"}, 2008},
		{collection{CDXAPI: "https://index.commoncrawl.org/CC-MAIN-2019-04-index"}, 2019},
		{collection{ID: "unexpected-name"}, 0},
		{collection{}, 0},
	}

	for _, tc := range tests {
		if got := collectionYear(tc.collection); got != tc.want {
			t.Errorf("collectionYear(%+v) = %d, want %d", tc.collection, got, tc.want)
		}
	}
}

func TestStringifyStatus(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{nil, ""},
		{"200", "200"},
		{float64(301), "301"},
		{float64(0), "0"},
	}

	for _, tc := range tests {
		if got := stringifyStatus(tc.input); got != tc.want {
			t.Errorf("stringifyStatus(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestWithScheme(t *testing.T) {
	tests := map[string]string{
		"example.com":         "http://example.com",
		"https://example.com": "https://example.com",
		"http://example.com":  "http://example.com",
	}

	for input, want := range tests {
		if got := withScheme(input); got != want {
			t.Errorf("withScheme(%q) = %q, want %q", input, got, want)
		}
	}
}

// The cursor has to preserve the millisecond timestamp digit for digit.
// Decoding it as a float64 renders "1.7e+12", which urlscan rejects with a 400.
func TestSortCursor(t *testing.T) {
	tests := []struct {
		name string
		sort []json.RawMessage
		want string
	}{
		{
			name: "number and string",
			sort: []json.RawMessage{json.RawMessage(`1763925600000`), json.RawMessage(`"abc123"`)},
			want: "1763925600000,abc123",
		},
		{
			name: "large number keeps every digit",
			sort: []json.RawMessage{json.RawMessage(`1700000000000`)},
			want: "1700000000000",
		},
		{name: "empty", sort: nil, want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := sortCursor(tc.sort); got != tc.want {
				t.Errorf("sortCursor() = %q, want %q", got, tc.want)
			}
		})
	}
}

// A full search response round-trips through the same decode path the source
// uses, so the cursor is exercised end to end.
func TestURLScanResponseCursor(t *testing.T) {
	body := `{"total":2,"has_more":true,"results":[
		{"sort":[1763925600000,"xyz789"],"page":{"url":"https://www.tesla.com/","domain":"www.tesla.com","status":200,"mimeType":"text/html"},"task":{"url":"https://tesla.com/","time":"2024-08-15T12:00:00.000Z"}}
	]}`

	var page urlscanResponse
	if err := json.Unmarshal([]byte(body), &page); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got, want := sortCursor(page.Results[0].Sort), "1763925600000,xyz789"; got != want {
		t.Errorf("cursor = %q, want %q", got, want)
	}
	if got := stringifyStatus(page.Results[0].Page.Status); got != "200" {
		t.Errorf("status = %q, want 200", got)
	}
	if got := TimestampFromISO(page.Results[0].Task.Time); got != "20240815120000" {
		t.Errorf("timestamp = %q, want 20240815120000", got)
	}
}

func TestIsoDate(t *testing.T) {
	if got, want := isoDate("20230415120000"), "2023-04-15"; got != want {
		t.Errorf("isoDate() = %q, want %q", got, want)
	}
	if got := isoDate("2023"); got != "" {
		t.Errorf("isoDate(short) = %q, want empty", got)
	}
}
