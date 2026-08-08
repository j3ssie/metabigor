package urlsource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/j3ssie/metabigor/internal/output"
)

// intelxBases are probed in order: the paid endpoint first, then the free one.
// An academia or paid key works on the first; other keys fall through.
var intelxBases = []string{"https://2.intelx.io", "https://free.intelx.io"}

const (
	// intelxTargetDomains searches the phonebook for hostnames.
	intelxTargetDomains = 1
	// intelxTargetURLs searches it for URLs.
	intelxTargetURLs = 3

	intelxMaxResults = 10000
	// intelxSearchTimeout is the server-side search budget, in seconds.
	intelxSearchTimeout = 20
	// intelxStatusNoMore is the status that means the result set is complete.
	intelxStatusNoMore = 1
	// intelxMaxPolls caps how many times one search is polled, so a search that
	// never reports completion cannot spin forever.
	intelxMaxPolls = 50
)

// fetchIntelX pulls URLs from the Intelligence X phonebook.
//
// The phonebook is an asynchronous search: a POST starts a job and returns an
// id, then results are polled until the service reports the set is complete.
// Both targets are searched - URLs, and hostnames which are emitted with a
// scheme - because they are indexed separately and overlap only partially.
func (r *run) fetchIntelX() error {
	apiKey := r.key(SourceIntelX)
	if apiKey == "" {
		return fmt.Errorf("no API key configured")
	}

	b := r.newBudget(SourceIntelX)
	base, err := r.intelxBase(apiKey, b)
	if err != nil {
		return err
	}
	output.Verbose("intelx: using %s", base)

	headers := map[string]string{"x-key": apiKey, "Content-Type": "application/json"}
	for _, target := range []int{intelxTargetURLs, intelxTargetDomains} {
		if err := r.fetchIntelXTarget(base, headers, target, b); err != nil {
			// One target failing (often a spent quota) should not discard the
			// other's results.
			output.Verbose("intelx: target %d: %v", target, err)
		}
	}
	return nil
}

// intelxBase returns the first endpoint that accepts the key.
func (r *run) intelxBase(apiKey string, b *budget) (string, error) {
	headers := map[string]string{"x-key": apiKey}
	var lastStatus int

	for _, base := range intelxBases {
		resp, err := r.claimGet(b, base+"/authenticate/info", headers)
		if err != nil {
			output.Verbose("intelx: %s unreachable: %v", base, err)
			continue
		}
		if resp == nil {
			return "", fmt.Errorf("request budget exhausted before authenticating")
		}
		if resp.OK() {
			return base, nil
		}
		lastStatus = resp.StatusCode
		output.Verbose("intelx: %s rejected the key (HTTP %d)", base, resp.StatusCode)
	}

	if lastStatus != 0 {
		return "", fmt.Errorf("API key rejected by every endpoint (last status HTTP %d)", lastStatus)
	}
	return "", fmt.Errorf("no Intelligence X endpoint reachable")
}

// intelxResultPage is one page of phonebook results.
type intelxResultPage struct {
	Status    int `json:"status"`
	Selectors []struct {
		Value  string `json:"selectorvalue"`
		ValueH string `json:"selectorvalueh"`
	} `json:"selectors"`
}

// fetchIntelXTarget runs one phonebook search to completion.
func (r *run) fetchIntelXTarget(base string, headers map[string]string, target int, b *budget) error {
	searchID, err := r.startIntelXSearch(base, headers, target, b)
	if err != nil || searchID == "" {
		return err
	}
	return r.pollIntelXResults(base, headers, searchID, target, b)
}

// startIntelXSearch submits the search and returns its id. An empty id with no
// error means the request budget ran out before the search began.
func (r *run) startIntelXSearch(base string, headers map[string]string, target int, b *budget) (string, error) {
	body, err := json.Marshal(map[string]any{
		"term":       r.target.Host,
		"target":     target,
		"maxresults": intelxMaxResults,
		"media":      0,
		"timeout":    intelxSearchTimeout,
	})
	if err != nil {
		return "", fmt.Errorf("build search request: %w", err)
	}

	if !b.claim() {
		return "", nil
	}
	resp, err := r.post(base+"/phonebook/search", headers, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	switch resp.StatusCode {
	case 401, 403:
		return "", fmt.Errorf("API key rejected (HTTP %d)", resp.StatusCode)
	case 402:
		return "", fmt.Errorf("out of daily credits (HTTP 402)")
	case 429:
		return "", fmt.Errorf("rate limit reached (HTTP 429)")
	}
	if !resp.OK() {
		return "", fmt.Errorf("search returned HTTP %d", resp.StatusCode)
	}

	var started struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp.Body, &started); err != nil {
		return "", fmt.Errorf("parse search response: %w", err)
	}
	if started.ID == "" {
		return "", fmt.Errorf("search returned no id")
	}
	return started.ID, nil
}

// pollIntelXResults reads result pages until the service reports the set is
// complete, recording every selector it returns.
func (r *run) pollIntelXResults(base string, headers map[string]string, searchID string, target int, b *budget) error {
	resultURL := fmt.Sprintf("%s/phonebook/search/result?id=%s&limit=%d",
		base, url.QueryEscape(searchID), intelxMaxResults)

	for range intelxMaxPolls {
		resp, err := r.claimGet(b, resultURL, headers)
		if err != nil {
			return err
		}
		if resp == nil {
			return nil
		}
		if !resp.OK() {
			return fmt.Errorf("result read returned HTTP %d", resp.StatusCode)
		}

		var page intelxResultPage
		if err := json.Unmarshal(resp.Body, &page); err != nil {
			return fmt.Errorf("parse result page: %w", err)
		}
		r.addIntelXSelectors(page)

		// Status 1 means the set is complete. An empty page means there is
		// nothing more to wait for.
		if page.Status == intelxStatusNoMore || len(page.Selectors) == 0 {
			return nil
		}
	}

	output.Verbose("intelx: target %d still reporting more results after %d polls; stopping", target, intelxMaxPolls)
	return nil
}

func (r *run) addIntelXSelectors(page intelxResultPage) {
	for _, selector := range page.Selectors {
		for _, value := range []string{selector.Value, selector.ValueH} {
			if value == "" {
				continue
			}
			// A domains search returns bare hostnames; a URL search returns
			// URLs that may still lack a scheme.
			r.add(SourceIntelX, withScheme(value), "", "", "")
		}
	}
}
