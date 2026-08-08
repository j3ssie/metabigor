package cert

import (
	"slices"
	"strings"
	"testing"
)

func sampleGroup() DomainGroup {
	return DomainGroup{
		Domain:      "api.example.com",
		CertIDs:     []string{"1", "2"},
		Count:       2,
		FirstSeen:   "2023-01-01",
		LastExpires: "2025-01-01",
		Issuers:     []string{"Let's Encrypt"},
		CommonNames: []string{"api.example.com"},
	}
}

func TestGroupTextIsBareDomainByDefault(t *testing.T) {
	// The default has to stay pipeable into dnsx, httpx, and friends.
	got := sampleGroup().Text()
	if want := []string{"api.example.com"}; !slices.Equal(got, want) {
		t.Errorf("Text() = %v, want %v", got, want)
	}
}

func TestGroupTextExpandsUnderDetail(t *testing.T) {
	g := sampleGroup()
	g.Detail = true

	got := strings.Join(g.Text(), "\n")
	for _, want := range []string{"Domain: api.example.com", "Certificates (2): 1, 2", "2023-01-01", "2025-01-01", "Let's Encrypt"} {
		if !strings.Contains(got, want) {
			t.Errorf("detail output should contain %q:\n%s", want, got)
		}
	}
	// A trailing blank line separates consecutive blocks.
	if lines := g.Text(); lines[len(lines)-1] != "" {
		t.Error("detail output should end with a blank line")
	}
}

func TestGroupFlatIgnoresDetail(t *testing.T) {
	g := sampleGroup()
	g.Detail = true

	if got := g.Flat(); !slices.Equal(got, []string{"api.example.com"}) {
		t.Errorf("Flat() = %v, want the bare domain regardless of --detail", got)
	}
}

func TestGroupByDomainAggregatesCertificates(t *testing.T) {
	entries := []CertEntry{
		{CertID: "1", NotBefore: "2023-06-01", NotAfter: "2024-06-01", IssuerName: "CA One", MatchingIdentities: []string{"a.example.com"}},
		{CertID: "2", NotBefore: "2022-01-01", NotAfter: "2025-01-01", IssuerName: "CA Two", MatchingIdentities: []string{"a.example.com", "b.example.com"}},
	}

	groups := GroupByDomain(entries)
	if len(groups) != 2 {
		t.Fatalf("got %d group(s), want 2", len(groups))
	}

	// Sorted by domain, so a.example.com comes first.
	a := groups[0]
	if a.Domain != "a.example.com" {
		t.Fatalf("first group is %q, want a.example.com", a.Domain)
	}
	if a.Count != 2 {
		t.Errorf("Count = %d, want 2", a.Count)
	}
	if a.FirstSeen != "2022-01-01" {
		t.Errorf("FirstSeen = %q, want the earliest NotBefore", a.FirstSeen)
	}
	if a.LastExpires != "2025-01-01" {
		t.Errorf("LastExpires = %q, want the latest NotAfter", a.LastExpires)
	}
	if len(a.Issuers) != 2 {
		t.Errorf("Issuers = %v, want both CAs", a.Issuers)
	}
}

func TestFilterEntriesCleanAndWildcard(t *testing.T) {
	entries := []CertEntry{
		{CertID: "1", MatchingIdentities: []string{"*.example.com", "example.com"}},
	}

	clean := filterEntries(entries, true, false)
	if got := clean[0].MatchingIdentities; !slices.Equal(got, []string{"example.com", "example.com"}) {
		t.Errorf("--clean should strip the *. prefix, got %v", got)
	}

	wild := filterEntries(entries, false, true)
	if got := wild[0].MatchingIdentities; !slices.Equal(got, []string{"*.example.com"}) {
		t.Errorf("--wildcard should keep only wildcards, got %v", got)
	}

	both := filterEntries(entries, true, true)
	if got := both[0].MatchingIdentities; !slices.Equal(got, []string{"example.com"}) {
		t.Errorf("--clean --wildcard should yield stripped wildcards, got %v", got)
	}
}
