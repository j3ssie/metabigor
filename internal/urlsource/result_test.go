package urlsource

import (
	"slices"
	"testing"
)

func TestResultText(t *testing.T) {
	result := Result{
		URL:       "https://example.com/a",
		Source:    SourceWayback,
		Status:    "200",
		MIME:      "text/html",
		Timestamp: "20230104120000",
	}

	// Without --detail the output is the bare URL, so it pipes into the next
	// tool unchanged.
	if got := result.Text(); !slices.Equal(got, []string{"https://example.com/a"}) {
		t.Errorf("Text() = %v, want the bare URL", got)
	}

	result.Detail = true
	want := []string{"https://example.com/a | wayback | 200 | text/html | 20230104120000"}
	if got := result.Text(); !slices.Equal(got, want) {
		t.Errorf("Text() = %v, want %v", got, want)
	}
}

// Fields a source did not report are left out rather than rendered as blanks.
func TestResultTextDetailOmitsMissingFields(t *testing.T) {
	result := Result{URL: "https://example.com/a", Source: SourceIntelX, Detail: true}

	want := []string{"https://example.com/a | intelx"}
	if got := result.Text(); !slices.Equal(got, want) {
		t.Errorf("Text() = %v, want %v", got, want)
	}
}

// Flat ignores --detail so pipelines stay stable regardless of other flags.
func TestResultFlat(t *testing.T) {
	result := Result{URL: "https://example.com/a", Source: SourceWayback, Detail: true}
	if got := result.Flat(); !slices.Equal(got, []string{"https://example.com/a"}) {
		t.Errorf("Flat() = %v, want the bare URL", got)
	}
}

func TestResultCSV(t *testing.T) {
	result := Result{
		Input:     "example.com",
		URL:       "https://example.com/a",
		Source:    SourceURLScan,
		Status:    "200",
		MIME:      "text/html",
		Timestamp: "20230104120000",
	}

	header, rows := result.CSV()
	wantHeader := []string{"input", "url", "source", "status", "mime", "timestamp"}
	if !slices.Equal(header, wantHeader) {
		t.Errorf("CSV header = %v, want %v", header, wantHeader)
	}
	if len(rows) != 1 || len(rows[0]) != len(wantHeader) {
		t.Fatalf("CSV rows = %v, want one row of %d columns", rows, len(wantHeader))
	}
	if rows[0][2] != "urlscan" {
		t.Errorf("source column = %q, want urlscan", rows[0][2])
	}
}

func newTestRun(t *testing.T, input string, cfg Config) *run {
	t.Helper()
	target, err := ParseTarget(input)
	if err != nil {
		t.Fatalf("ParseTarget(%q): %v", input, err)
	}
	if err := cfg.Filters.Compile(); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if cfg.Concurrency == 0 {
		cfg.Concurrency = 1
	}
	return newRun(nil, target, cfg)
}

// A URL two sources both report is emitted once, credited to whichever reported
// it first.
func TestAddDeduplicatesAcrossSources(t *testing.T) {
	r := newTestRun(t, "example.com", Config{})

	r.add(SourceWayback, "https://example.com/a", "200", "text/html", "20230104120000")
	r.add(SourceCommonCrawl, "https://example.com/a", "200", "text/html", "20230104120000")

	results := r.snapshot()
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Source != SourceWayback {
		t.Errorf("source = %q, want the first reporter (wayback)", results[0].Source)
	}

	// The per-source tally credits only what each source actually contributed,
	// which is what the verbose output reports.
	if got := r.addedFor(SourceWayback); got != 1 {
		t.Errorf("addedFor(wayback) = %d, want 1", got)
	}
	if got := r.addedFor(SourceCommonCrawl); got != 0 {
		t.Errorf("addedFor(commoncrawl) = %d, want 0 for a duplicate", got)
	}
}

func TestAddRejectsOutOfScope(t *testing.T) {
	r := newTestRun(t, "example.com", Config{})

	r.add(SourceWayback, "https://notexample.com/a", "", "", "")
	r.add(SourceGhostArchive, "https://example.com.evil.net/a", "", "", "")
	r.add(SourceWayback, "https://blog.example.com/ok", "", "", "")

	results := r.snapshot()
	if len(results) != 1 {
		t.Fatalf("got %d results (%v), want only the in-scope subdomain", len(results), results)
	}
	if results[0].URL != "https://blog.example.com/ok" {
		t.Errorf("URL = %q, want the in-scope subdomain", results[0].URL)
	}
}

func TestAddAppliesFilters(t *testing.T) {
	r := newTestRun(t, "example.com", Config{
		Filters: Filters{FilterStatus: []string{"404"}, Blacklist: []string{"png"}},
	})

	r.add(SourceWayback, "https://example.com/gone", "404", "", "")
	r.add(SourceWayback, "https://example.com/logo.png", "200", "", "")
	r.add(SourceWayback, "https://example.com/kept", "200", "", "")

	results := r.snapshot()
	if len(results) != 1 || results[0].URL != "https://example.com/kept" {
		t.Errorf("results = %v, want only https://example.com/kept", results)
	}
}

func TestAddAppliesDateRange(t *testing.T) {
	r := newTestRun(t, "example.com", Config{Filters: Filters{From: "2023", To: "2023"}})

	r.add(SourceWayback, "https://example.com/old", "", "", "20220101000000")
	r.add(SourceWayback, "https://example.com/current", "", "", "20230615120000")
	// A source that reports no date cannot be judged, so it is kept.
	r.add(SourceIntelX, "https://example.com/undated", "", "", "")

	results := r.snapshot()
	if len(results) != 2 {
		t.Fatalf("got %d results (%v), want 2", len(results), results)
	}
}

// --no-params keeps one URL per endpoint.
func TestAddCollapsesParams(t *testing.T) {
	r := newTestRun(t, "example.com", Config{Filters: Filters{NoParams: true}})

	r.add(SourceWayback, "https://example.com/search?q=1", "", "", "")
	r.add(SourceWayback, "https://example.com/search?q=2", "", "", "")
	r.add(SourceWayback, "https://example.com/other?q=1", "", "", "")

	if results := r.snapshot(); len(results) != 2 {
		t.Errorf("got %d results (%v), want 2 endpoints", len(results), results)
	}
}

// Without --no-params, query-string variants are distinct results.
func TestAddKeepsParamsByDefault(t *testing.T) {
	r := newTestRun(t, "example.com", Config{})

	r.add(SourceWayback, "https://example.com/search?q=1", "", "", "")
	r.add(SourceWayback, "https://example.com/search?q=2", "", "", "")

	if results := r.snapshot(); len(results) != 2 {
		t.Errorf("got %d results, want both query variants", len(results))
	}
}

// Normalization runs before deduplication, so :443 and the bare host collapse.
func TestAddNormalizesBeforeDedup(t *testing.T) {
	r := newTestRun(t, "example.com", Config{})

	r.add(SourceWayback, "https://example.com:443/a", "", "", "")
	r.add(SourceWayback, "https://example.com/a", "", "", "")

	results := r.snapshot()
	if len(results) != 1 {
		t.Fatalf("got %d results (%v), want 1", len(results), results)
	}
	if results[0].URL != "https://example.com/a" {
		t.Errorf("URL = %q, want the port stripped", results[0].URL)
	}
}

func TestAddPropagatesDetail(t *testing.T) {
	r := newTestRun(t, "example.com", Config{Detail: true})
	r.add(SourceWayback, "https://example.com/a", "", "", "")

	results := r.snapshot()
	if len(results) != 1 || !results[0].Detail {
		t.Errorf("results = %v, want Detail propagated from the config", results)
	}
}

// The cap is a hard stop, and hitting it is reported rather than silently
// truncating the results.
func TestBudgetClaim(t *testing.T) {
	b := &budget{src: SourceWayback, limit: 2}
	if !b.claim() || !b.claim() {
		t.Fatal("the first two claims should succeed")
	}
	if b.claim() {
		t.Error("the third claim should fail once the limit is spent")
	}
	if !b.warned {
		t.Error("exhausting the budget should log a warning once")
	}
}

func TestBudgetUnlimited(t *testing.T) {
	b := &budget{src: SourceWayback, limit: 0}
	for i := range 100 {
		if !b.claim() {
			t.Fatalf("claim %d failed, want an unlimited budget", i)
		}
	}
}
