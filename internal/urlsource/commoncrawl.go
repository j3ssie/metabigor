package urlsource

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

const (
	collinfoURL = "https://index.commoncrawl.org/collinfo.json"
	// collinfoTTL is how long the cached index list stays fresh. Common Crawl
	// publishes roughly one new collection per month.
	collinfoTTL   = 30 * 24 * time.Hour
	collinfoCache = "commoncrawl-collinfo.json"
	dataDir       = ".metabigor"
)

// collection is one Common Crawl index collection.
type collection struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	CDXAPI string `json:"cdx-api"`
}

// fetchCommonCrawl pulls URLs from the Common Crawl CDX indexes.
//
// Common Crawl publishes a separate index per crawl - more than a hundred of
// them, going back to 2008. Querying only the newest, as is common, covers
// about one quarter of one year. --limit-collections controls the depth, and 0
// searches all of them.
//
// The default searches several of the newest rather than just one, because a
// collection is listed in collinfo.json as soon as it is announced, which can be
// before its index is actually queryable. Searching only the newest would then
// return nothing at all.
func (r *run) fetchCommonCrawl() error {
	indexes, err := r.commonCrawlIndexes()
	if err != nil {
		return err
	}
	if len(indexes) == 0 {
		return nil
	}

	output.Verbose("commoncrawl: searching %d index collection(s) for %s", len(indexes), r.target.Host)
	b := r.newBudget(SourceCommonCrawl)

	tasks := make([]func(), 0, len(indexes))
	for _, index := range indexes {
		tasks = append(tasks, func() {
			if err := r.fetchCommonCrawlIndex(index, b); err != nil {
				output.Verbose("commoncrawl: %s: %v", index.ID, err)
			}
		})
	}
	r.fan(tasks)
	return nil
}

// fetchCommonCrawlIndex queries one collection, paging through its results.
func (r *run) fetchCommonCrawlIndex(index collection, b *budget) error {
	base := fmt.Sprintf("%s?output=json&fl=timestamp,url,mime,status&url=%s",
		index.CDXAPI, EscapeScope(r.target.CDXScope(r.cfg.IncludeSubs))) + r.cfg.Filters.CommonCrawlParams()

	pages := 0
	if resp, err := r.claimGet(b, base+"&showNumPages=true", nil); err == nil && resp != nil && resp.OK() {
		pages = parseCommonCrawlPageCount(resp.Text())
	}
	pages = b.clamp(pages)

	if pages <= 1 {
		resp, err := r.claimGet(b, base, nil)
		if err != nil {
			return err
		}
		if resp == nil {
			return nil
		}
		return r.parseCommonCrawlBody(resp.Body, resp.StatusCode)
	}

	r.fetchPages(b, "commoncrawl "+index.ID, 0, pages,
		func(page int) string { return fmt.Sprintf("%s&page=%d", base, page) },
		func(resp *httpclient.Response) error {
			return r.parseCommonCrawlBody(resp.Body, resp.StatusCode)
		})
	return nil
}

// parseCommonCrawlPageCount reads a showNumPages response, which arrives as
// JSON ({"pages": 2, "pageSize": 5, "blocks": 9}) or, on older indexes, as a
// bare integer. An unrecognized body yields 0, meaning "page count unknown".
//
// Note that unlike the Wayback CDX, Common Crawl requires the fl field list to
// be present for this to work at all: dropping it answers 502.
func parseCommonCrawlPageCount(body string) int {
	text := strings.TrimSpace(body)
	if text == "" {
		return 0
	}

	var payload struct {
		Pages int `json:"pages"`
	}
	if err := json.Unmarshal([]byte(text), &payload); err == nil {
		return payload.Pages
	}
	if n, err := strconv.Atoi(text); err == nil && n >= 0 {
		return n
	}
	return 0
}

// parseCommonCrawlBody reads one NDJSON page of CDX records.
func (r *run) parseCommonCrawlBody(body []byte, statusCode int) error {
	// A collection with nothing for the scope answers 404 with a short
	// plain-text notice. That is an empty result, not a failure. Only the head
	// is checked: the notice is never inside a real NDJSON page, and lowercasing
	// every page in full to look for it would copy megabytes per request.
	if statusCode == 404 || containsFoldPrefix(body, "no captures found") {
		return nil
	}
	if statusCode < 200 || statusCode >= 300 {
		return fmt.Errorf("HTTP %d", statusCode)
	}

	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}

		var record struct {
			URL       string `json:"url"`
			MIME      string `json:"mime"`
			Status    string `json:"status"`
			Timestamp string `json:"timestamp"`
			Error     string `json:"error"`
		}
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}
		if record.Error != "" || record.URL == "" {
			continue
		}
		r.add(SourceCommonCrawl, record.URL, record.Status, record.MIME, record.Timestamp)
	}
	return sc.Err()
}

// commonCrawlIndexes returns the collections to search, newest first, after
// applying the date range and the --limit-collections cap.
func (r *run) commonCrawlIndexes() ([]collection, error) {
	raw, err := r.collinfo()
	if err != nil {
		return nil, err
	}

	var all []collection
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("parse collinfo.json: %w", err)
	}

	var selected []collection
	for _, c := range all {
		if c.CDXAPI == "" {
			continue
		}
		if !r.collectionInDateRange(c) {
			continue
		}
		selected = append(selected, c)
		if r.cfg.LimitCollections > 0 && len(selected) >= r.cfg.LimitCollections {
			break
		}
	}

	if r.cfg.LimitCollections > 0 && len(all) > len(selected) {
		output.Verbose("commoncrawl: using the newest %d of %d collections (--limit-collections 0 searches all)",
			len(selected), len(all))
	}
	return selected, nil
}

// collectionInDateRange reports whether a collection's crawl year overlaps
// --from/--to, so out-of-range collections are skipped without a request.
func (r *run) collectionInDateRange(c collection) bool {
	if !r.cfg.Filters.HasDateRange() {
		return true
	}

	year := collectionYear(c)
	if year == 0 {
		// An unparseable name is searched rather than silently skipped.
		return true
	}
	// The oldest collections are named for two years (CC-MAIN-2009-2010), so the
	// lower bound is relaxed by one to keep them in range.
	if from := r.cfg.Filters.FromYear(); from > 0 && year < from-1 {
		return false
	}
	if to := r.cfg.Filters.ToYear(); to > 0 && year > to {
		return false
	}
	return true
}

// collectionYear extracts the crawl year from an ID like "CC-MAIN-2024-33".
func collectionYear(c collection) int {
	name := c.ID
	if name == "" {
		name = c.CDXAPI
	}
	idx := strings.Index(name, "CC-MAIN-")
	if idx < 0 {
		return 0
	}
	rest := name[idx+len("CC-MAIN-"):]
	if len(rest) < 4 {
		return 0
	}
	year, err := strconv.Atoi(rest[:4])
	if err != nil {
		return 0
	}
	return year
}

// collinfo returns the raw index list, from the local cache when it is fresh.
//
// The list changes about once a month but is needed on every run, so caching it
// removes one request per invocation. A stale or unreadable cache falls back to
// the network, and a cache that cannot be written is not fatal.
func (r *run) collinfo() ([]byte, error) {
	path := collinfoCachePath()

	if path != "" {
		if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < collinfoTTL {
			if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
				output.Debug("commoncrawl: using cached index list %s", path)
				return data, nil
			}
		}
	}

	resp, err := r.get(collinfoURL, nil)
	if err != nil {
		// A stale cache beats no results at all.
		if data, readErr := os.ReadFile(path); readErr == nil && len(data) > 0 {
			output.Verbose("commoncrawl: index list unavailable (%v), using stale cache", err)
			return data, nil
		}
		return nil, fmt.Errorf("fetch index list: %w", err)
	}
	if !resp.OK() {
		return nil, fmt.Errorf("index list returned HTTP %d", resp.StatusCode)
	}

	if path != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err == nil {
			if err := os.WriteFile(path, resp.Body, 0o644); err != nil {
				output.Debug("commoncrawl: could not cache index list: %v", err)
			}
		}
	}
	return resp.Body, nil
}

func collinfoCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, dataDir, collinfoCache)
}
