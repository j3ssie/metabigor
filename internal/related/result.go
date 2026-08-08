package related

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/output"
)

// Source identifies a related-domain data source.
type Source string

const (
	// SourceCRT searches Certificate Transparency logs via crt.sh.
	SourceCRT Source = "crt"
	// SourceWhois runs a reverse WHOIS lookup via viewdns.info.
	SourceWhois Source = "whois"
	// SourceAnalytics correlates Google Analytics and Tag Manager IDs.
	SourceAnalytics Source = "analytics"
)

// Sources lists every source, in the order --sources all queries them.
var Sources = []Source{SourceCRT, SourceWhois, SourceAnalytics}

// SourceNames returns the source names for help text and errors.
func SourceNames() []string {
	names := make([]string, len(Sources))
	for i, s := range Sources {
		names[i] = string(s)
	}
	return names
}

// ParseSources validates user-supplied source names. The special value "all"
// expands to every source, and duplicates are collapsed.
func ParseSources(values []string) ([]Source, error) {
	if len(values) == 0 {
		return Sources, nil
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
			return Sources, nil
		}

		src := Source(name)
		if !slices.Contains(Sources, src) {
			return nil, fmt.Errorf("unknown source %q (pick from: %s, or all)", raw, strings.Join(SourceNames(), ", "))
		}
		if !seen[src] {
			seen[src] = true
			parsed = append(parsed, src)
		}
	}

	if len(parsed) == 0 {
		return Sources, nil
	}
	return parsed, nil
}

// Result is one domain related to the input, tagged with where it came from.
type Result struct {
	Input  string `json:"input"`
	Domain string `json:"domain"`
	Source Source `json:"source"`
}

// Text renders the domain alongside the source that surfaced it.
func (r Result) Text() []string {
	return []string{fmt.Sprintf("%s | %s", r.Domain, r.Source)}
}

// Flat renders the bare domain for piping into other tools.
func (r Result) Flat() []string {
	return []string{r.Domain}
}

// CSV renders the input, domain, and source.
func (r Result) CSV() ([]string, [][]string) {
	return []string{"input", "domain", "source"},
		[][]string{{r.Input, r.Domain, string(r.Source)}}
}

// Find queries each source for domains related to the input, deduplicating
// across sources while keeping the first source that reported each domain.
func Find(client *retryablehttp.Client, input string, sources []Source) []Result {
	var (
		results []Result
		seen    = make(map[string]bool)
	)

	for _, src := range sources {
		var domains []string
		switch src {
		case SourceCRT:
			output.Verbose("Querying crt.sh for %s", input)
			domains = CRTRelated(client, input)
		case SourceWhois:
			output.Verbose("Querying viewdns.info reverse WHOIS for %s", input)
			domains = WhoisRelated(client, input)
		case SourceAnalytics:
			output.Verbose("Correlating analytics IDs for %s", input)
			domains = AnalyticsRelated(client, input)
		}

		added := 0
		for _, d := range domains {
			d = strings.ToLower(strings.TrimSpace(d))
			if d == "" || seen[d] {
				continue
			}
			seen[d] = true
			results = append(results, Result{Input: input, Domain: d, Source: src})
			added++
		}
		output.Verbose("%s: %d new domain(s) for %s", src, added, input)
	}

	return results
}
