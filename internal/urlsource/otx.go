package urlsource

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

const (
	otxIndicatorsURL = "https://otx.alienvault.com/api/v1/indicators"
	// otxPageSize is the largest page OTX serves. Asking for the maximum turns
	// a walk of many small pages into a handful of large ones.
	otxPageSize = 500
	// otxMaxPages bounds the has_next walk. That path only runs when full_size
	// is unusable, and without a cap an API that always reports another page
	// would never let the run finish.
	otxMaxPages = 500
)

// otxPage is one page of the OTX url_list response.
type otxPage struct {
	HasNext  bool `json:"has_next"`
	FullSize int  `json:"full_size"`
	URLList  []struct {
		URL      string `json:"url"`
		Date     string `json:"date"`
		HTTPCode int    `json:"httpcode"`
	} `json:"url_list"`
}

// fetchOTX pulls URLs from the AlienVault OTX url_list endpoint.
//
// OTX has no wildcard, so the indicator type follows the requested scope: a
// registrable domain is queried as a domain (which includes its subdomains),
// and a subdomain input is queried as a hostname. Widening a subdomain to its
// parent would return URLs outside the scope the user asked for.
//
// OTX reports no MIME type, so --match-mime/--filter-mime cannot apply here.
func (r *run) fetchOTX() error {
	host, indicator := r.target.Host, r.target.OTXIndicator()
	base := fmt.Sprintf("%s/%s/%s/url_list?limit=%d",
		otxIndicatorsURL, indicator, url.PathEscape(host), otxPageSize)

	b := r.newBudget(SourceOTX)
	resp, err := r.claimGet(b, base+"&page=1", nil)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	// OTX answers 404 for an indicator it has never seen.
	if resp.StatusCode == 404 {
		return nil
	}
	if !resp.OK() {
		return fmt.Errorf("otx.alienvault.com returned HTTP %d", resp.StatusCode)
	}

	first, err := parseOTXPage(resp.Body)
	if err != nil {
		return err
	}
	r.addOTXEntries(first)

	// full_size is the total across every page, so the page count is known up
	// front and the rest can be fetched in parallel.
	if first.FullSize > len(first.URLList) {
		pages := b.clamp((first.FullSize + otxPageSize - 1) / otxPageSize)
		if pages > 1 {
			output.Verbose("otx: %d URL(s) across %d page(s) for %s", first.FullSize, pages, host)
			// Page 1 is already in hand, so the range starts at 2 and the
			// upper bound is exclusive.
			r.fetchPages(b, "otx", 2, pages+1,
				func(page int) string { return fmt.Sprintf("%s&page=%d", base, page) },
				func(resp *httpclient.Response) error {
					page, err := parseOTXPage(resp.Body)
					if err != nil {
						return err
					}
					r.addOTXEntries(page)
					return nil
				})
		}
		return nil
	}

	// Without a usable full_size, fall back to walking has_next in order.
	for page := 2; first.HasNext && page <= otxMaxPages; page++ {
		next := r.fetchOTXPage(fmt.Sprintf("%s&page=%d", base, page), b)
		if next == nil {
			break
		}
		first = next
	}
	return nil
}

// fetchOTXPage retrieves and records one page, returning it so a sequential
// walk can read has_next.
func (r *run) fetchOTXPage(pageURL string, b *budget) *otxPage {
	resp, err := r.claimGet(b, pageURL, nil)
	if err != nil {
		output.Verbose("otx: %v", err)
		return nil
	}
	if resp == nil || !resp.OK() {
		return nil
	}

	page, err := parseOTXPage(resp.Body)
	if err != nil {
		output.Verbose("otx: %v", err)
		return nil
	}
	r.addOTXEntries(page)
	return page
}

func (r *run) addOTXEntries(page *otxPage) {
	for _, entry := range page.URLList {
		if entry.URL == "" {
			continue
		}
		status := ""
		if entry.HTTPCode > 0 {
			status = strconv.Itoa(entry.HTTPCode)
		}
		r.add(SourceOTX, entry.URL, status, "", TimestampFromISO(entry.Date))
	}
}

func parseOTXPage(body []byte) (*otxPage, error) {
	var page otxPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &page, nil
}
