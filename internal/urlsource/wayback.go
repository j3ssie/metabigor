package urlsource

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

const waybackCDXURL = "https://web.archive.org/cdx/search/cdx"

// fetchWayback pulls URLs from the Wayback Machine CDX index.
//
// Two deliberate choices make this return more than a conventional CDX query:
//
//   - No output=json. The default plain-text form is smaller on the wire and
//     parses with one split per line. It also has no header row to skip, which
//     the JSON form does.
//   - No collapse=urlkey. Collapsing merges rows whose *canonicalized* key
//     matches - lowercased host, reordered query string - so genuinely distinct
//     URLs disappear. It also interacts badly with paging once filters are
//     applied. Deduplication happens locally instead, on the exact URL.
func (r *run) fetchWayback() error {
	scope := EscapeScope(r.target.CDXScope(r.cfg.IncludeSubs))
	filters := r.cfg.Filters.WaybackParams()

	// The page count has to be requested *without* fl. Asking for a field list
	// makes the API answer with an empty row ("- - - -") instead of the count,
	// which is easy to mistake for "paging is unavailable" and leads to pulling
	// a whole domain in one unpaged request that then times out.
	countURL := fmt.Sprintf("%s?url=%s%s&showNumPages=true", waybackCDXURL, scope, filters)
	base := fmt.Sprintf("%s?url=%s&fl=timestamp,original,mimetype,statuscode%s", waybackCDXURL, scope, filters)

	b := r.newBudget(SourceWayback)

	if pages := r.waybackPageCount(countURL, b); pages > 0 {
		output.Verbose("wayback: fetching %d page(s) for %s", pages, r.target.Host)
		r.fetchPages(b, "wayback", 0, pages,
			func(page int) string { return fmt.Sprintf("%s&page=%d", base, page) },
			r.parseWaybackBody)
		return nil
	}

	// Without a count, walk pages in order until one comes back empty. This
	// beats a single unpaged request, which for a busy domain is tens of
	// megabytes and rarely finishes inside a sane timeout.
	output.Verbose("wayback: page count unavailable, walking pages for %s", r.target.Host)
	return r.walkWaybackPages(base, b)
}

// walkWaybackPages fetches consecutive pages until one yields no rows.
func (r *run) walkWaybackPages(base string, b *budget) error {
	for page := 0; ; page++ {
		resp, err := r.claimGet(b, fmt.Sprintf("%s&page=%d", base, page), nil)
		if err != nil {
			if page == 0 {
				return err
			}
			output.Verbose("wayback: stopped at page %d: %v", page, err)
			return nil
		}
		if resp == nil {
			return nil
		}
		if len(bytes.TrimSpace(resp.Body)) == 0 {
			return nil
		}
		if err := r.parseWaybackBody(resp); err != nil {
			if page == 0 {
				return err
			}
			return nil
		}
	}
}

// waybackPageCount asks how many pages the query spans, returning 0 when the
// API will not say.
func (r *run) waybackPageCount(countURL string, b *budget) int {
	resp, err := r.claimGet(b, countURL, nil)
	if err != nil {
		output.Verbose("wayback: page count request failed: %v", err)
		return 0
	}
	if resp == nil || !resp.OK() {
		return 0
	}

	pages, err := strconv.Atoi(strings.TrimSpace(resp.Text()))
	if err != nil || pages < 0 {
		return 0
	}
	return b.clamp(pages)
}

// parseWaybackBody reads one plain-text CDX page.
func (r *run) parseWaybackBody(resp *httpclient.Response) error {
	// The CDX API answers 400 for a scope it has nothing indexed for, which is
	// an empty result rather than a failure.
	if resp.StatusCode == 400 {
		return nil
	}
	if !resp.OK() {
		return fmt.Errorf("archive.org returned HTTP %d", resp.StatusCode)
	}
	// The exclusion notice is a short error page, so only the head is checked.
	// Lowercasing a multi-megabyte CDX page to find it would copy every page
	// twice for a string that can never appear in one.
	if containsFoldPrefix(resp.Body, "blocked site error") {
		return fmt.Errorf("archive.org excludes captures for this site")
	}

	sc := bufio.NewScanner(bytes.NewReader(resp.Body))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		timestamp, rawURL, mime, status := parseCDXLine(sc.Text())
		if rawURL == "" {
			continue
		}
		r.add(SourceWayback, rawURL, status, mime, timestamp)
	}
	return sc.Err()
}

// parseCDXLine splits a plain-text CDX row of
// `timestamp original mimetype statuscode`. Trailing fields may be absent.
//
// Splitting on whitespace is safe because the CDX index stores `original`
// percent-encoded, so a captured URL never contains a literal space.
func parseCDXLine(line string) (timestamp, rawURL, mime, status string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", "", ""
	}
	timestamp, rawURL = fields[0], fields[1]
	if len(fields) > 2 {
		mime = fields[2]
	}
	if len(fields) > 3 {
		status = fields[3]
	}
	return timestamp, rawURL, mime, status
}
