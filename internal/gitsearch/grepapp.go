// Package gitsearch provides GitHub code search functionality via grep.app API.
package gitsearch

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

// GrepAppHit represents a single grep.app search hit.
type GrepAppHit struct {
	OwnerID      string `json:"owner_id"`
	Repo         string `json:"repo"`
	Branch       string `json:"branch"`
	Path         string `json:"path"`
	Content      HitContent `json:"content"`
	TotalMatches string     `json:"total_matches"`
}

// HitContent contains the code snippet from a search hit.
type HitContent struct {
	Snippet string `json:"snippet"`
}

// CleanSnippet returns the snippet with HTML tags and <mark> wrappers stripped.
func (h *GrepAppHit) CleanSnippet() string {
	s := h.Content.Snippet
	s = strings.ReplaceAll(s, "<mark>", "")
	s = strings.ReplaceAll(s, "</mark>", "")
	s = stripHTMLTags(s)
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	// Collapse multi-line into trimmed lines
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// ParseSnippet parses the HTML table snippet and returns formatted code with line numbers.
// Expected format: <table class="highlight-table"><tr data-line="3"><td><div class="lineno">3</div></td><td>code here</td></tr>...
func (h *GrepAppHit) ParseSnippet() string {
	s := h.Content.Snippet
	if s == "" {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(s))
	if err != nil {
		// Fallback to cleaned snippet if parsing fails
		return h.CleanSnippet()
	}

	var lines []string
	doc.Find("table.highlight-table tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() < 2 {
			return
		}

		// First td contains line number
		lineNum := strings.TrimSpace(tds.Eq(0).Text())
		// Second td contains code
		code := tds.Eq(1).Text()

		// Decode HTML entities
		code = strings.ReplaceAll(code, "&gt;", ">")
		code = strings.ReplaceAll(code, "&lt;", "<")
		code = strings.ReplaceAll(code, "&amp;", "&")
		code = strings.ReplaceAll(code, "&quot;", "\"")
		code = strings.ReplaceAll(code, "&#39;", "'")

		if lineNum != "" && code != "" {
			lines = append(lines, fmt.Sprintf("%s: %s", lineNum, code))
		}
	})

	if len(lines) == 0 {
		// Fallback if no table found
		return h.CleanSnippet()
	}

	return strings.Join(lines, "\n")
}

type grepAppResponse struct {
	Hits struct {
		Total int          `json:"total"`
		Hits  []GrepAppHit `json:"hits"`
	} `json:"hits"`
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTMLTags(s string) string {
	return htmlTagRe.ReplaceAllString(s, "")
}

// SearchAll queries grep.app API with auto-pagination until no more hits.
// Returns all hits across all pages. Adds a delay between page requests.
// If maxPages > 0, stops after fetching that many pages.
func SearchAll(client *retryablehttp.Client, query string, delay time.Duration, maxPages int) []GrepAppHit {
	var allHits []GrepAppHit

	for page := 1; ; page++ {
		// Check max pages limit
		if maxPages > 0 && page > maxPages {
			output.Verbose("grep.app: reached max page limit (%d), stopping", maxPages)
			break
		}

		u := fmt.Sprintf("https://grep.app/api/search?regexp=true&q=%s&page=%d&format=e", url.QueryEscape(query), page)
		output.Verbose("grep.app page %d: %s", page, u)

		body, err := httpclient.Get(client, u)
		if err != nil {
			output.Error("grep.app page %d failed: %v", page, err)
			break
		}

		var resp grepAppResponse
		if err := json.Unmarshal([]byte(body), &resp); err != nil {
			output.Debug("grep.app page %d parse error: %v", page, err)
			break
		}

		if len(resp.Hits.Hits) == 0 {
			output.Verbose("grep.app page %d: no more hits, stopping", page)
			break
		}

		allHits = append(allHits, resp.Hits.Hits...)
		output.Verbose("grep.app page %d: %d hits (total so far: %d / %d)", page, len(resp.Hits.Hits), len(allHits), resp.Hits.Total)

		if len(allHits) >= resp.Hits.Total {
			break
		}

		// Delay before next page
		if delay > 0 {
			time.Sleep(delay)
		}
	}

	output.Good("grep.app: %d total hits for %q", len(allHits), query)
	return allHits
}

// subdomainRe matches subdomains: one or more label.domain pattern
var subdomainRe = regexp.MustCompile(`[a-zA-Z0-9]([a-zA-Z0-9_-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9_-]{0,61}[a-zA-Z0-9])?)+`)

// ExtractSubdomains pulls subdomains matching the input domain from the cleaned snippet.
// e.g. input="sc-corp.net" will find "jira.sc-corp.net"
func ExtractSubdomains(hits []GrepAppHit, domain string) []string {
	domain = strings.ToLower(domain)
	seen := make(map[string]bool)
	var results []string

	for _, h := range hits {
		clean := h.CleanSnippet()
		clean = strings.ReplaceAll(clean, "<mark>", "")
		clean = strings.ReplaceAll(clean, "</mark>", "")
		for _, match := range subdomainRe.FindAllString(clean, -1) {
			m := strings.ToLower(match)
			if strings.HasSuffix(m, "."+domain) || m == domain {
				if !seen[m] {
					seen[m] = true
					results = append(results, m)
				}
			}
		}
	}

	return results
}
