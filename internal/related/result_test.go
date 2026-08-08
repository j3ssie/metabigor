package related

import (
	"slices"
	"strings"
	"testing"
)

func TestParseSourcesAcceptsCommaSeparatedList(t *testing.T) {
	// Cobra splits `-s crt,whois` into two values before it reaches us.
	got, err := ParseSources([]string{"crt", "whois"})
	if err != nil {
		t.Fatalf("ParseSources: %v", err)
	}

	want := []Source{SourceCRT, SourceWhois}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseSourcesDefaultsToAll(t *testing.T) {
	for _, in := range [][]string{nil, {}, {"all"}, {"crt", "all"}, {"  "}} {
		got, err := ParseSources(in)
		if err != nil {
			t.Fatalf("ParseSources(%v): %v", in, err)
		}
		if !slices.Equal(got, Sources) {
			t.Errorf("ParseSources(%v) = %v, want every source", in, got)
		}
	}
}

func TestParseSourcesDeduplicatesAndNormalizes(t *testing.T) {
	got, err := ParseSources([]string{"CRT", " crt ", "crt"})
	if err != nil {
		t.Fatalf("ParseSources: %v", err)
	}
	if !slices.Equal(got, []Source{SourceCRT}) {
		t.Errorf("got %v, want [crt]", got)
	}
}

func TestParseSourcesRejectsUnknownSource(t *testing.T) {
	_, err := ParseSources([]string{"crt", "bogus"})
	if err == nil {
		t.Fatal("an unknown source should be an error")
	}
	// The message has to name the offender and list the valid options.
	for _, want := range []string{"bogus", "crt", "whois", "analytics", "all"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q", err, want)
		}
	}
}

func TestResultRendering(t *testing.T) {
	r := Result{Input: "tesla.com", Domain: "teslamotors.com", Source: SourceWhois}

	if got, want := r.Text(), []string{"teslamotors.com | whois"}; !slices.Equal(got, want) {
		t.Errorf("Text() = %v, want %v", got, want)
	}
	if got, want := r.Flat(), []string{"teslamotors.com"}; !slices.Equal(got, want) {
		t.Errorf("Flat() = %v, want %v", got, want)
	}

	header, rows := r.CSV()
	if !slices.Equal(header, []string{"input", "domain", "source"}) {
		t.Errorf("CSV header = %v", header)
	}
	if len(rows) != 1 || !slices.Equal(rows[0], []string{"tesla.com", "teslamotors.com", "whois"}) {
		t.Errorf("CSV rows = %v", rows)
	}
}
