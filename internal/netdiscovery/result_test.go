package netdiscovery

import (
	"slices"
	"testing"
)

func sampleResult() Result {
	return Result{
		Input:   "1.1.1.1",
		ASN:     "AS13335",
		CIDR:    "1.1.1.0/24",
		Org:     "Cloudflare, Inc.",
		Country: "US",
		Source:  sourceLocal,
	}
}

func TestResultTextIsBareCIDRByDefault(t *testing.T) {
	got := sampleResult().Text()
	if want := []string{"1.1.1.0/24"}; !slices.Equal(got, want) {
		t.Errorf("Text() = %v, want %v", got, want)
	}
}

func TestResultTextWidensUnderDetail(t *testing.T) {
	r := sampleResult()
	r.Detail = true

	got := r.Text()
	want := []string{"AS13335 | 1.1.1.0/24 | Cloudflare, Inc. | US"}
	if !slices.Equal(got, want) {
		t.Errorf("Text() = %v, want %v", got, want)
	}
}

func TestResultDetailSkipsEmptyColumns(t *testing.T) {
	r := Result{Input: "x", CIDR: "8.8.8.0/24", Detail: true}

	got := r.Text()
	if want := []string{"8.8.8.0/24"}; !slices.Equal(got, want) {
		t.Errorf("Text() = %v, want %v", got, want)
	}
}

func TestResultFallsBackToASNWhenNoCIDR(t *testing.T) {
	r := Result{Input: "tesla", ASN: "AS394161"}

	if got := r.Value(); got != "AS394161" {
		t.Errorf("Value() = %q, want the ASN", got)
	}
	if got := r.Flat(); !slices.Equal(got, []string{"AS394161"}) {
		t.Errorf("Flat() = %v", got)
	}
}

func TestResultCSVIgnoresDetail(t *testing.T) {
	// CSV is for machines, so it always carries every column.
	r := sampleResult()
	header, rows := r.CSV()

	if !slices.Equal(header, []string{"input", "asn", "cidr", "org", "country", "source"}) {
		t.Errorf("CSV header = %v", header)
	}
	want := []string{"1.1.1.1", "AS13335", "1.1.1.0/24", "Cloudflare, Inc.", "US", "local-db"}
	if len(rows) != 1 || !slices.Equal(rows[0], want) {
		t.Errorf("CSV rows = %v, want [%v]", rows, want)
	}
}

func TestDetectType(t *testing.T) {
	tests := []struct {
		input string
		want  InputType
	}{
		{"AS13335", TypeASN},
		{"as13335", TypeASN},
		{"13335", TypeASN},
		{"1.1.1.1", TypeIP},
		{"1.1.1.0/24", TypeCIDR},
		{"2606:4700::1", TypeIP},
		{"example.com", TypeDomain},
		{"sub.example.co.uk", TypeDomain},
		{"Cloudflare", TypeOrg},
		{"Tesla Motors", TypeOrg},
	}

	for _, tt := range tests {
		if got := DetectType(tt.input); got != tt.want {
			t.Errorf("DetectType(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
