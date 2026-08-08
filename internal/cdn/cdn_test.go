package cdn

import (
	"slices"
	"testing"

	"github.com/projectdiscovery/cdncheck"
)

func TestHostExtractsFromURL(t *testing.T) {
	tests := map[string]string{
		"1.1.1.1":                  "1.1.1.1",
		"https://1.1.1.1/path?a=b": "1.1.1.1",
		"http://1.1.1.1:8080/x":    "1.1.1.1",
		"example.com":              "example.com",
		"https://example.com":      "example.com",
		"httpsomething-not-a-url":  "httpsomething-not-a-url",
	}
	for in, want := range tests {
		if got := Host(in); got != want {
			t.Errorf("Host(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClassifyRejectsNonAddresses(t *testing.T) {
	client := cdncheck.New()
	if client == nil {
		t.Skip("cdncheck client unavailable")
	}

	for _, target := range []string{"example.com", "not an ip", "", "https://example.com"} {
		if _, ok := Classify(client, target); ok {
			t.Errorf("Classify(%q) reported success for a non-address", target)
		}
	}
}

func TestClassifyDetectsKnownCDN(t *testing.T) {
	client := cdncheck.New()
	if client == nil {
		t.Skip("cdncheck client unavailable")
	}

	got, ok := Classify(client, "1.1.1.1")
	if !ok {
		t.Fatal("Classify(1.1.1.1) failed")
	}
	if !got.IsCDN {
		t.Errorf("1.1.1.1 should be flagged as CDN/WAF, got %+v", got)
	}
	if got.Vendor != "cloudflare" {
		t.Errorf("Vendor = %q, want cloudflare", got.Vendor)
	}
}

func TestClassifyFillsPlaceholdersForUnmatchedIP(t *testing.T) {
	client := cdncheck.New()
	if client == nil {
		t.Skip("cdncheck client unavailable")
	}

	// A reserved-for-documentation address no provider announces.
	got, ok := Classify(client, "192.0.2.1")
	if !ok {
		t.Fatal("Classify(192.0.2.1) failed")
	}
	if got.IsCDN {
		t.Errorf("192.0.2.1 should not be a CDN, got %+v", got)
	}
	if got.Vendor != unknownVendor || got.Type != noType {
		t.Errorf("blank fields should get placeholders, got vendor=%q type=%q", got.Vendor, got.Type)
	}
}

func TestResultRendering(t *testing.T) {
	r := Result{IP: "1.1.1.1", IsCDN: true, Vendor: "cloudflare", Type: "waf"}

	if got, want := r.Text(), []string{"1.1.1.1 | cloudflare | waf"}; !slices.Equal(got, want) {
		t.Errorf("Text() = %v, want %v", got, want)
	}
	if got, want := r.Flat(), []string{"1.1.1.1"}; !slices.Equal(got, want) {
		t.Errorf("Flat() = %v, want %v", got, want)
	}

	header, rows := r.CSV()
	if !slices.Equal(header, []string{"ip", "is_cdn", "vendor", "type"}) {
		t.Errorf("CSV header = %v", header)
	}
	if len(rows) != 1 || !slices.Equal(rows[0], []string{"1.1.1.1", "true", "cloudflare", "waf"}) {
		t.Errorf("CSV rows = %v", rows)
	}
}
