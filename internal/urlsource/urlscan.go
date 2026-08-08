package urlsource

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/j3ssie/metabigor/internal/output"
)

const urlscanSearchURL = "https://urlscan.io/api/v1/search/"

// urlscanPageSize is how many results to ask for per page. Anonymous callers
// are held to 100; a key raises the ceiling considerably.
const (
	urlscanPageSize       = 100
	urlscanKeyedPageSize  = 1000
	urlscanEarliestScanYr = "2016-01-01" // urlscan.io has nothing older
)

// urlscanResponse is one page of urlscan.io search results.
type urlscanResponse struct {
	Total   int  `json:"total"`
	HasMore bool `json:"has_more"`
	Results []struct {
		// Sort is kept as raw JSON. Decoding it into `any` would turn the
		// millisecond timestamp into a float64, which renders in scientific
		// notation and makes the search_after cursor a 400.
		Sort []json.RawMessage `json:"sort"`
		Page struct {
			URL    string `json:"url"`
			Domain string `json:"domain"`
			// PTR is the reverse-DNS name urlscan resolved for the host, which
			// is frequently a hostname that appears nowhere else.
			PTR      string `json:"ptr"`
			Status   any    `json:"status"`
			MimeType string `json:"mimeType"`
		} `json:"page"`
		Task struct {
			// URL is what was submitted for scanning, which can differ from the
			// URL that was finally loaded after redirects.
			URL  string `json:"url"`
			Time string `json:"time"`
		} `json:"task"`
	} `json:"results"`
}

// fetchURLScan pulls URLs from urlscan.io search.
//
// Three URLs are harvested per result, not one: the page that loaded, the task
// URL that was submitted, and the host's PTR record. Redirect chains and
// reverse-DNS names routinely surface hostnames no archive index carries.
//
// A key is used when URLSCAN_API_KEY is set. If a keyed request fails - an
// expired key, or a quota that is spent - the same query is retried
// anonymously, since the anonymous tier still answers.
func (r *run) fetchURLScan() error {
	apiKey := r.key(SourceURLScan)

	query := "domain:" + r.target.Host
	if dateRange := r.urlscanDateRange(); dateRange != "" {
		query += " " + dateRange
	}

	// The page size follows the key rather than being tracked alongside it, so
	// the anonymous fallbacks below only have to clear the key.
	baseFor := func(key string) string {
		size := urlscanPageSize
		if key != "" {
			size = urlscanKeyedPageSize
		}
		return fmt.Sprintf("%s?q=%s&size=%d", urlscanSearchURL, url.QueryEscape(query), size)
	}

	b := r.newBudget(SourceURLScan)
	base := baseFor(apiKey)
	searchAfter := ""

	for {
		pageURL := base
		if searchAfter != "" {
			pageURL += "&search_after=" + url.QueryEscape(searchAfter)
		}

		resp, err := r.claimGet(b, pageURL, urlscanHeaders(apiKey))
		if err != nil {
			// The retry layer has already backed off and given up, which for
			// urlscan almost always means the rate limit. Falling back to the
			// anonymous tier recovers the run instead of abandoning it.
			if apiKey != "" {
				output.Verbose("urlscan: keyed request failed (%v), retrying anonymously", err)
				apiKey, base = "", baseFor("")
				continue
			}
			return err
		}
		if resp == nil {
			return nil
		}

		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			if apiKey != "" {
				output.Warn("urlscan: API key rejected (HTTP %d), continuing anonymously", resp.StatusCode)
				apiKey, base = "", baseFor("")
				continue
			}
			return fmt.Errorf("urlscan.io returned HTTP %d", resp.StatusCode)
		}
		if resp.StatusCode == 429 {
			return fmt.Errorf("urlscan.io rate limit reached; results are partial")
		}
		if !resp.OK() {
			return fmt.Errorf("urlscan.io returned HTTP %d", resp.StatusCode)
		}

		var page urlscanResponse
		if err := json.Unmarshal(resp.Body, &page); err != nil {
			return fmt.Errorf("parse response: %w", err)
		}
		if len(page.Results) == 0 {
			return nil
		}

		for _, item := range page.Results {
			// Without --subs, keep only exact-host matches: urlscan's `domain:`
			// operator always includes subdomains.
			if !r.cfg.IncludeSubs && item.Page.Domain != "" &&
				!strings.EqualFold(item.Page.Domain, r.target.Host) {
				continue
			}

			status := stringifyStatus(item.Page.Status)
			timestamp := TimestampFromISO(item.Task.Time)

			r.add(SourceURLScan, item.Page.URL, status, item.Page.MimeType, timestamp)
			r.add(SourceURLScan, item.Task.URL, status, item.Page.MimeType, timestamp)
			// A PTR is a bare hostname, so it needs a scheme to be a URL.
			if item.Page.PTR != "" {
				r.add(SourceURLScan, withScheme(item.Page.PTR), "", "", timestamp)
			}
		}

		if !page.HasMore {
			return nil
		}
		// Pagination is cursor-based: the last result's sort values become the
		// search_after of the next request.
		next := sortCursor(page.Results[len(page.Results)-1].Sort)
		if next == "" || next == searchAfter {
			return nil
		}
		searchAfter = next
	}
}

func urlscanHeaders(apiKey string) map[string]string {
	if apiKey == "" {
		return nil
	}
	return map[string]string{"API-Key": apiKey}
}

// urlscanDateRange renders --from/--to as an urlscan query clause.
func (r *run) urlscanDateRange() string {
	if !r.cfg.Filters.HasDateRange() {
		return ""
	}
	from := isoDate(r.cfg.Filters.fromTS)
	if from == "" {
		from = urlscanEarliestScanYr
	}
	to := isoDate(r.cfg.Filters.toTS)
	if to == "" {
		to = "now"
	}
	return fmt.Sprintf("date:[%s TO %s]", from, to)
}

// isoDate renders the first eight digits of a 14-digit timestamp as YYYY-MM-DD.
func isoDate(ts string) string {
	if len(ts) < 8 {
		return ""
	}
	return ts[0:4] + "-" + ts[4:6] + "-" + ts[6:8]
}

// sortCursor joins urlscan's sort values into a search_after cursor, preserving
// each value exactly as the server sent it.
func sortCursor(sort []json.RawMessage) string {
	if len(sort) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sort))
	for _, raw := range sort {
		value := strings.TrimSpace(string(raw))
		// A string value arrives quoted; the cursor takes the bare text.
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		parts = append(parts, value)
	}
	return strings.Join(parts, ",")
}

// stringifyStatus renders a status that arrives as either a number or a string.
func stringifyStatus(v any) string {
	switch typed := v.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.Itoa(int(typed))
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

// withScheme prefixes a bare hostname so it renders as a URL.
func withScheme(host string) string {
	if strings.Contains(host, "://") {
		return host
	}
	return "http://" + host
}
