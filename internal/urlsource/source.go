// Package urlsource discovers URLs for a target from public web archives and
// indexes: what a site exposed historically, not just what it serves today.
//
// Five sources need no credentials. VirusTotal and Intelligence X require an
// API key, which is read from the environment so the zero-config promise still
// holds - they simply sit out when their key is absent.
package urlsource

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// Source identifies a URL data source.
type Source string

const (
	// SourceWayback queries the Wayback Machine CDX index (web.archive.org).
	SourceWayback Source = "wayback"
	// SourceCommonCrawl queries the Common Crawl CDX indexes.
	SourceCommonCrawl Source = "commoncrawl"
	// SourceOTX queries the AlienVault OTX URL list.
	SourceOTX Source = "otx"
	// SourceURLScan queries urlscan.io search.
	SourceURLScan Source = "urlscan"
	// SourceGhostArchive scrapes ghostarchive.org and mines its WARC files.
	SourceGhostArchive Source = "ghostarchive"
	// SourceVirusTotal queries the VirusTotal v3 domain relationships.
	SourceVirusTotal Source = "virustotal"
	// SourceIntelX queries the Intelligence X phonebook.
	SourceIntelX Source = "intelx"
)

// descriptor is everything the package needs to know about one source.
type descriptor struct {
	// Name is the value users pass to --sources.
	Name Source
	// Fetch queries the source and hands each URL it finds to run.add.
	Fetch func(*run) error
	// KeyEnvs lists the environment variables that can supply an API key, in
	// preference order. Empty means the source needs no credentials.
	KeyEnvs []string
	// KeyRequired marks a source that cannot run at all without a key. urlscan
	// sets KeyEnvs but not this, because it works anonymously with tighter
	// limits.
	KeyRequired bool
}

// descriptors is the single registry of sources, in the order a run queries
// them. Adding a source means adding one entry here: a missing Fetch is a
// compile error, and there is no second list to keep in sync.
var descriptors = []descriptor{
	{Name: SourceWayback, Fetch: (*run).fetchWayback},
	{Name: SourceCommonCrawl, Fetch: (*run).fetchCommonCrawl},
	{Name: SourceOTX, Fetch: (*run).fetchOTX},
	{Name: SourceURLScan, Fetch: (*run).fetchURLScan, KeyEnvs: []string{"URLSCAN_API_KEY"}},
	{Name: SourceGhostArchive, Fetch: (*run).fetchGhostArchive},
	{
		Name:        SourceVirusTotal,
		Fetch:       (*run).fetchVirusTotal,
		KeyEnvs:     []string{"VT_API_KEY", "VIRUSTOTAL_API_KEY"},
		KeyRequired: true,
	},
	{
		Name:        SourceIntelX,
		Fetch:       (*run).fetchIntelX,
		KeyEnvs:     []string{"INTELX_API_KEY"},
		KeyRequired: true,
	},
}

// Sources lists every source, in the order a run queries them.
var Sources = func() []Source {
	names := make([]Source, len(descriptors))
	for i, d := range descriptors {
		names[i] = d.Name
	}
	return names
}()

// lookup returns the descriptor for a source.
func lookup(s Source) (descriptor, bool) {
	for _, d := range descriptors {
		if d.Name == s {
			return d, true
		}
	}
	return descriptor{}, false
}

// SourceNames returns every source name, for help text and error messages.
func SourceNames() []string {
	names := make([]string, len(Sources))
	for i, s := range Sources {
		names[i] = string(s)
	}
	return names
}

// APIKey returns the key configured for a source, or "" when none is set.
func APIKey(s Source) string {
	d, ok := lookup(s)
	if !ok {
		return ""
	}
	for _, env := range d.KeyEnvs {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			return v
		}
	}
	return ""
}

// KeyEnvNames returns the environment variables that can supply a source's
// key, so errors can name them.
func KeyEnvNames(s Source) []string {
	d, _ := lookup(s)
	return d.KeyEnvs
}

// Available reports whether a source can run right now: either it needs no
// key, or its key is present in the environment.
func Available(s Source) bool {
	d, ok := lookup(s)
	if !ok {
		return false
	}
	return !d.KeyRequired || APIKey(s) != ""
}

// DefaultSources returns the sources a run uses when --sources is omitted:
// every keyless source, plus any credentialed source whose key is set.
func DefaultSources() []Source {
	var out []Source
	for _, d := range descriptors {
		if Available(d.Name) {
			out = append(out, d.Name)
		}
	}
	return out
}

// ParseSources validates user-supplied source names. An empty list and the
// special value "all" both expand to DefaultSources, and duplicates collapse.
//
// Naming a credentialed source explicitly without its key is an error rather
// than a silent skip: the user asked for it by name, so staying quiet would
// hide the reason the results look thin.
func ParseSources(values []string) ([]Source, error) {
	if len(values) == 0 {
		return defaultOrError()
	}

	var (
		parsed []Source
		seen   = make(map[Source]bool)
	)
	for _, raw := range values {
		name := strings.ToLower(strings.TrimSpace(raw))
		if name == "" {
			continue
		}
		if name == "all" {
			return defaultOrError()
		}

		src := Source(name)
		d, ok := lookup(src)
		if !ok {
			return nil, fmt.Errorf("unknown source %q (pick from: %s, or all)",
				raw, strings.Join(SourceNames(), ", "))
		}
		if d.KeyRequired && APIKey(src) == "" {
			return nil, fmt.Errorf("source %q needs an API key: set %s",
				src, strings.Join(d.KeyEnvs, " or "))
		}
		if !seen[src] {
			seen[src] = true
			parsed = append(parsed, src)
		}
	}

	if len(parsed) == 0 {
		return defaultOrError()
	}
	return parsed, nil
}

// defaultOrError guards against the pathological case where every source needs
// a key, so an empty selection never silently produces zero results.
func defaultOrError() ([]Source, error) {
	sources := DefaultSources()
	if len(sources) == 0 {
		return nil, fmt.Errorf("no sources available (pick from: %s)", strings.Join(SourceNames(), ", "))
	}
	return sources, nil
}

// keyEnvVars returns every environment variable any source reads, for tests
// that need a clean environment.
func keyEnvVars() []string {
	var envs []string
	for _, d := range descriptors {
		envs = append(envs, d.KeyEnvs...)
	}
	return slices.Compact(envs)
}
