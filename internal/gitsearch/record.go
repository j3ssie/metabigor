package gitsearch

import (
	"fmt"
	"strings"

	"github.com/j3ssie/metabigor/internal/output"
)

// Text renders one line per hit, followed by the matching snippet under
// --detail.
func (h GrepAppHit) Text() []string {
	var matches string
	if h.TotalMatches != "" {
		matches = h.TotalMatches + " matches"
	}
	line := output.Columns(h.Repo, h.Path, h.Branch, matches)
	if !h.Detail {
		return []string{line}
	}

	lines := []string{line}
	for _, l := range strings.Split(h.ParseSnippet(), "\n") {
		lines = append(lines, "  "+l)
	}
	// Trailing blank line separates consecutive snippets.
	return append(lines, "")
}

// Flat renders repo/path, the location of the match.
func (h GrepAppHit) Flat() []string {
	return []string{fmt.Sprintf("%s/%s", h.Repo, h.Path)}
}

// CSV renders the hit metadata plus a single-line form of the snippet.
func (h GrepAppHit) CSV() ([]string, [][]string) {
	snippet := strings.ReplaceAll(h.CleanSnippet(), "\n", " ")
	return []string{"repo", "branch", "path", "matches", "snippet"},
		[][]string{{h.Repo, h.Branch, h.Path, h.TotalMatches, snippet}}
}

// Subdomain is a hostname extracted from search hits. It exists so `github
// --subs` can stream results through the same formatting pipeline.
type Subdomain struct {
	Query  string `json:"query"`
	Domain string `json:"domain"`
}

// Text renders the bare subdomain.
func (s Subdomain) Text() []string { return []string{s.Domain} }

// Flat renders the bare subdomain.
func (s Subdomain) Flat() []string { return []string{s.Domain} }

// CSV renders the originating query alongside the subdomain.
func (s Subdomain) CSV() ([]string, [][]string) {
	return []string{"query", "domain"}, [][]string{{s.Query, s.Domain}}
}
