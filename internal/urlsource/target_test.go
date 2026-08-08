package urlsource

import "testing"

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantHost    string
		wantPath    string
		registrable string
	}{
		{"bare domain", "example.com", "example.com", "", "example.com"},
		{"subdomain", "blog.example.com", "blog.example.com", "", "example.com"},
		{"scheme dropped", "https://example.com", "example.com", "", "example.com"},
		{"lone trailing slash is not a path", "example.com/", "example.com", "", "example.com"},
		{"path kept", "example.com/admin", "example.com", "/admin", "example.com"},
		{"deep path kept", "https://example.com/a/b/c", "example.com", "/a/b/c", "example.com"},
		{"port stripped", "example.com:8443/x", "example.com", "/x", "example.com"},
		{"query dropped", "example.com/a?b=c", "example.com", "/a", "example.com"},
		{"fragment dropped", "example.com/a#frag", "example.com", "/a", "example.com"},
		{"leading dots trimmed", ".example.com", "example.com", "", "example.com"},
		{"multi-part suffix", "a.b.example.co.uk", "a.b.example.co.uk", "", "example.co.uk"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTarget(tc.input)
			if err != nil {
				t.Fatalf("ParseTarget(%q) returned error: %v", tc.input, err)
			}
			if got.Host != tc.wantHost {
				t.Errorf("Host = %q, want %q", got.Host, tc.wantHost)
			}
			if got.Path != tc.wantPath {
				t.Errorf("Path = %q, want %q", got.Path, tc.wantPath)
			}
			if got.Registrable != tc.registrable {
				t.Errorf("Registrable = %q, want %q", got.Registrable, tc.registrable)
			}
		})
	}
}

// The hostname is case-insensitive but the path is not, so only the host may be
// lowercased. Lowercasing the path silently breaks case-sensitive routes.
func TestParseTargetPreservesPathCase(t *testing.T) {
	got, err := ParseTarget("EXAMPLE.com/Admin/File.JS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Host != "example.com" {
		t.Errorf("Host = %q, want lowercased example.com", got.Host)
	}
	if got.Path != "/Admin/File.JS" {
		t.Errorf("Path = %q, want the original case preserved", got.Path)
	}
}

// Archives store unicode domains in punycode, so a unicode input has to be
// converted or it matches nothing at all.
func TestParseTargetPunycode(t *testing.T) {
	got, err := ParseTarget("bücher.example")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Host != "xn--bcher-kva.example" {
		t.Errorf("Host = %q, want punycode xn--bcher-kva.example", got.Host)
	}
}

func TestParseTargetErrors(t *testing.T) {
	for _, input := range []string{"", "   ", "localhost", "https://", "exa mple.com"} {
		if got, err := ParseTarget(input); err == nil {
			t.Errorf("ParseTarget(%q) = %+v, want an error", input, got)
		}
	}
}

func TestCDXScope(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		includeSubs bool
		want        string
	}{
		{"subdomains on", "example.com", true, "*.example.com/*"},
		{"subdomains off", "example.com", false, "example.com/*"},
		{"subdomain input keeps its subtree", "blog.example.com", true, "*.blog.example.com/*"},
		// A path pins the host, so the subdomain wildcard would widen the scope
		// past what was asked for and is not applied.
		{"path scope ignores subs", "example.com/admin", true, "example.com/admin*"},
		{"path scope without subs", "example.com/admin", false, "example.com/admin*"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			target, err := ParseTarget(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := target.CDXScope(tc.includeSubs); got != tc.want {
				t.Errorf("CDXScope(%v) = %q, want %q", tc.includeSubs, got, tc.want)
			}
		})
	}
}

// The wildcard and path separator carry meaning for the CDX APIs, which infer
// their match type from them, so they must survive escaping.
func TestEscapeScope(t *testing.T) {
	if got, want := EscapeScope("*.example.com/*"), "*.example.com/*"; got != want {
		t.Errorf("EscapeScope = %q, want %q", got, want)
	}
	if got, want := EscapeScope("example.com/a b"), "example.com/a+b"; got != want {
		t.Errorf("EscapeScope = %q, want %q", got, want)
	}
}

func TestOTXIndicator(t *testing.T) {
	tests := []struct {
		input         string
		wantHost      string
		wantIndicator string
	}{
		// A registrable domain is queried as a domain, which includes its
		// subdomains.
		{"example.com", "example.com", "domain"},
		// A subdomain is queried as a hostname; widening it to the parent would
		// return URLs outside the requested scope.
		{"blog.example.com", "blog.example.com", "hostname"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			target, err := ParseTarget(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := target.OTXIndicator(); got != tc.wantIndicator {
				t.Errorf("OTXIndicator() = %q, want %q", got, tc.wantIndicator)
			}
			if target.Host != tc.wantHost {
				t.Errorf("Host = %q, want %q", target.Host, tc.wantHost)
			}
		})
	}
}

// A registrable domain is dot-prefixed so the search anchors on a label
// boundary rather than also matching notexample.com.
func TestGhostArchiveTerm(t *testing.T) {
	tests := map[string]string{
		"example.com":       ".example.com",
		"example.co.uk":     ".example.co.uk",
		"blog.example.com":  "blog.example.com",
		"a.b.example.co.uk": "a.b.example.co.uk",
	}

	for input, want := range tests {
		target, err := ParseTarget(input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", input, err)
		}
		if got := target.GhostArchiveTerm(); got != want {
			t.Errorf("GhostArchiveTerm(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestInScope(t *testing.T) {
	bare, err := ParseTarget("example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	scoped, err := ParseTarget("example.com/Admin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name      string
		target    Target
		candidate string
		want      bool
	}{
		{"exact host", bare, "https://example.com/a", true},
		{"subdomain", bare, "https://blog.example.com/a", true},
		{"deep subdomain", bare, "http://a.b.example.com/", true},
		// A substring check would wrongly accept these two.
		{"lookalike suffix rejected", bare, "https://notexample.com/a", false},
		{"different tld rejected", bare, "https://example.com.evil.net/a", false},
		{"unrelated host rejected", bare, "https://other.org/example.com", false},
		{"bare hostname accepted", bare, "http://blog.example.com", true},
		{"uppercase host", bare, "https://EXAMPLE.com/a", true},
		{"port ignored", bare, "https://example.com:8080/a", true},
		{"path prefix matches", scoped, "https://example.com/Admin/users", true},
		{"path prefix case-insensitive", scoped, "https://example.com/admin/users", true},
		{"path prefix encoded", scoped, "https://example.com/Admin%2Fusers", true},
		{"wrong path rejected", scoped, "https://example.com/public", false},
		{"empty candidate rejected", bare, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := InScope(tc.candidate, tc.target); got != tc.want {
				t.Errorf("InScope(%q) = %v, want %v", tc.candidate, got, tc.want)
			}
		})
	}
}
