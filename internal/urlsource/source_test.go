package urlsource

import (
	"slices"
	"strings"
	"testing"
)

// clearKeys removes any credentials the developer's shell happens to export, so
// the default-source assertions do not depend on the environment.
func clearKeys(t *testing.T) {
	t.Helper()
	for _, env := range keyEnvVars() {
		t.Setenv(env, "")
	}
}

// Without keys, the five keyless sources are the default; the credentialed ones
// sit out rather than failing the run.
func TestDefaultSourcesWithoutKeys(t *testing.T) {
	clearKeys(t)

	got := DefaultSources()
	want := []Source{SourceWayback, SourceCommonCrawl, SourceOTX, SourceURLScan, SourceGhostArchive}
	if !slices.Equal(got, want) {
		t.Errorf("DefaultSources() = %v, want %v", got, want)
	}
}

// A credentialed source joins the default set as soon as its key is present, so
// there is still nothing to configure.
func TestDefaultSourcesWithKeys(t *testing.T) {
	clearKeys(t)
	t.Setenv("VT_API_KEY", "test-key")

	got := DefaultSources()
	if !slices.Contains(got, SourceVirusTotal) {
		t.Errorf("DefaultSources() = %v, want it to include virustotal once VT_API_KEY is set", got)
	}
	if slices.Contains(got, SourceIntelX) {
		t.Errorf("DefaultSources() = %v, want intelx excluded without its key", got)
	}
}

// VIRUSTOTAL_API_KEY is accepted as an alias, since that is what other tools use.
func TestAPIKeyAlias(t *testing.T) {
	clearKeys(t)
	t.Setenv("VIRUSTOTAL_API_KEY", "alias-key")

	if got := APIKey(SourceVirusTotal); got != "alias-key" {
		t.Errorf("APIKey(virustotal) = %q, want alias-key", got)
	}
}

func TestAPIKeyPrefersFirstName(t *testing.T) {
	clearKeys(t)
	t.Setenv("VT_API_KEY", "primary")
	t.Setenv("VIRUSTOTAL_API_KEY", "alias")

	if got := APIKey(SourceVirusTotal); got != "primary" {
		t.Errorf("APIKey(virustotal) = %q, want primary to win", got)
	}
}

func TestParseSources(t *testing.T) {
	clearKeys(t)

	tests := []struct {
		name  string
		input []string
		want  []Source
	}{
		{"empty means default", nil, DefaultSources()},
		{"all means default", []string{"all"}, DefaultSources()},
		{"explicit subset", []string{"wayback", "otx"}, []Source{SourceWayback, SourceOTX}},
		{"duplicates collapse", []string{"wayback", "wayback"}, []Source{SourceWayback}},
		{"case and space tolerated", []string{" WayBack "}, []Source{SourceWayback}},
		{"blank entries ignored", []string{"wayback", ""}, []Source{SourceWayback}},
		{"all wins over a subset", []string{"wayback", "all"}, DefaultSources()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSources(tc.input)
			if err != nil {
				t.Fatalf("ParseSources(%v) returned error: %v", tc.input, err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("ParseSources(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseSourcesUnknown(t *testing.T) {
	clearKeys(t)

	_, err := ParseSources([]string{"waybackurls"})
	if err == nil {
		t.Fatal("ParseSources with an unknown source = nil error, want an error")
	}
	// The error should list the valid names so the typo is easy to fix.
	if !strings.Contains(err.Error(), "wayback") {
		t.Errorf("error %q should list the valid source names", err)
	}
}

// Naming a credentialed source without its key is an error, not a silent skip:
// staying quiet would hide why the results look thin.
func TestParseSourcesKeyedWithoutKey(t *testing.T) {
	clearKeys(t)

	for _, src := range []Source{SourceVirusTotal, SourceIntelX} {
		_, err := ParseSources([]string{string(src)})
		if err == nil {
			t.Fatalf("ParseSources(%q) = nil error, want an error about the missing key", src)
		}
		// The message has to name the variable to set.
		if !strings.Contains(err.Error(), KeyEnvNames(src)[0]) {
			t.Errorf("error %q should name %s", err, KeyEnvNames(src)[0])
		}
	}
}

func TestParseSourcesKeyedWithKey(t *testing.T) {
	clearKeys(t)
	t.Setenv("INTELX_API_KEY", "test-key")

	got, err := ParseSources([]string{"intelx"})
	if err != nil {
		t.Fatalf("ParseSources returned error: %v", err)
	}
	if !slices.Equal(got, []Source{SourceIntelX}) {
		t.Errorf("ParseSources = %v, want [intelx]", got)
	}
}

// urlscan is usable anonymously, so it must never require a key even though it
// does read one when offered.
func TestURLScanNeedsNoKey(t *testing.T) {
	clearKeys(t)

	d, ok := lookup(SourceURLScan)
	if !ok {
		t.Fatal("urlscan is missing from the registry")
	}
	if d.KeyRequired {
		t.Error("urlscan KeyRequired = true, want false: it works anonymously")
	}
	if len(d.KeyEnvs) == 0 {
		t.Error("urlscan should still read a key when one is offered")
	}
	if _, err := ParseSources([]string{"urlscan"}); err != nil {
		t.Errorf("ParseSources([urlscan]) returned error: %v", err)
	}
}

func TestSourceNamesCoversEverySource(t *testing.T) {
	if got, want := len(SourceNames()), len(Sources); got != want {
		t.Errorf("SourceNames() has %d entries, want %d", got, want)
	}
}

// The registry is the single place a source is declared, so every entry must be
// complete: a name, something to run, and key metadata that agrees with itself.
func TestRegistryIsComplete(t *testing.T) {
	for _, d := range descriptors {
		if d.Name == "" {
			t.Error("a descriptor has no name")
		}
		if d.Fetch == nil {
			t.Errorf("%s has no Fetch implementation", d.Name)
		}
		if d.KeyRequired && len(d.KeyEnvs) == 0 {
			t.Errorf("%s requires a key but names no environment variable to set it", d.Name)
		}
	}
}
