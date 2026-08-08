package urlsource

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/j3ssie/metabigor/internal/output"
)

const (
	ghostArchiveBase   = "https://ghostarchive.org"
	ghostArchiveSearch = ghostArchiveBase + "/search?term=%s&page=%d"
	// ghostArchiveMaxPages bounds the search crawl. The site restarts at page 1
	// once the page number exceeds what it has, so an unbounded loop would
	// never terminate on its own.
	ghostArchiveMaxPages = 200
)

// archiveLinkPattern matches a result row, capturing the archive path and the
// URL that was archived: <a href="/archive/AbCdE">https://example.com/</a>
var archiveLinkPattern = regexp.MustCompile(`<a href="(/archive/[^"]+)">([^<]+)</a>`)

// warcTargetPattern matches the target URI header of a WARC record.
var warcTargetPattern = regexp.MustCompile(`(?i)^WARC-Target-URI:\s*(\S+)`)

// Byte literals for the WARC header scan, so the hot loop compares bytes
// without allocating a string per line.
var (
	warcPrefix        = []byte("WARC")
	warcVersionPrefix = []byte("WARC/")
	warcResponseType  = []byte("WARC-Type: response")
)

// fetchGhostArchive scrapes ghostarchive.org and then mines its WARC files.
//
// The site has no API, so the search pages are parsed directly. The valuable
// part is the second step: every archived page also has a raw WARC file, and a
// WARC records *every sub-request the browser made while capturing the page* -
// XHR calls, JSON endpoints, dynamically built script URLs.
//
// None of that appears in a CDX index, because CDX only indexes captures of the
// page URL itself. This is the one source here that discovers URLs rather than
// aggregating an existing index, so it runs whenever ghostarchive is selected.
// It costs one request per archived page, which --limit-requests bounds.
func (r *run) fetchGhostArchive() error {
	b := r.newBudget(SourceGhostArchive)
	// The theme cookie keeps the markup in the plain form the parser expects.
	headers := map[string]string{"Cookie": "theme=original"}

	archiveIDs, err := r.ghostArchiveSearch(headers, b)
	if err != nil {
		return err
	}
	if len(archiveIDs) == 0 {
		return nil
	}

	output.Verbose("ghostarchive: mining %d WARC file(s) for %s", len(archiveIDs), r.target.Host)
	tasks := make([]func(), 0, len(archiveIDs))
	for _, path := range archiveIDs {
		tasks = append(tasks, func() {
			if err := r.mineWARC(path, headers, b); err != nil {
				output.Verbose("ghostarchive: %s: %v", path, err)
			}
		})
	}
	r.fan(tasks)
	return nil
}

// ghostArchiveSearch walks the paginated search results, recording the archived
// URLs and collecting the archive paths whose WARC files are worth mining.
func (r *run) ghostArchiveSearch(headers map[string]string, b *budget) ([]string, error) {
	var (
		paths []string
		seen  = make(map[string]bool)
	)

	for page := range ghostArchiveMaxPages {
		pageURL := fmt.Sprintf(ghostArchiveSearch, url.QueryEscape(r.target.GhostArchiveTerm()), page)
		resp, err := r.claimGet(b, pageURL, headers)
		if err != nil {
			// Keep whatever earlier pages produced.
			if page == 0 {
				return nil, err
			}
			output.Verbose("ghostarchive: page %d failed: %v", page, err)
			return paths, nil
		}
		if resp == nil {
			return paths, nil
		}

		body := resp.Text()
		lower := strings.ToLower(body)
		if resp.StatusCode == 503 || strings.Contains(lower, "under maintenance") {
			return paths, fmt.Errorf("ghostarchive.org is under maintenance")
		}
		if strings.Contains(lower, "no archives for that site") {
			return paths, nil
		}
		if !resp.OK() {
			if page == 0 {
				return nil, fmt.Errorf("ghostarchive.org returned HTTP %d", resp.StatusCode)
			}
			return paths, nil
		}

		matches := archiveLinkPattern.FindAllStringSubmatch(body, -1)
		if len(matches) == 0 {
			return paths, nil
		}

		for _, match := range matches {
			archivePath, archivedURL := match[1], strings.TrimSpace(match[2])
			r.add(SourceGhostArchive, archivedURL, "", "", "")
			if !seen[archivePath] {
				seen[archivePath] = true
				paths = append(paths, archivePath)
			}
		}

		// The site loops back to page 1 past the end, so the absence of a next
		// link is the only reliable stop condition.
		if !strings.Contains(body, "Next Page") && !strings.Contains(body, ">&raquo;</a>") &&
			!strings.Contains(body, ">»</a>") {
			return paths, nil
		}
	}

	output.Verbose("ghostarchive: stopped at the %d page crawl limit", ghostArchiveMaxPages)
	return paths, nil
}

// mineWARC downloads one archived page's WARC file and extracts the target URI
// of every response record inside it.
//
// The WARC lives at the archive path with /archive swapped for /chimurai4 and a
// .warc suffix. Only response records count: a request record's target URI is
// the same URL, and metadata records carry the archive's own bookkeeping.
func (r *run) mineWARC(archivePath string, headers map[string]string, b *budget) error {
	warcURL := ghostArchiveBase + strings.Replace(archivePath, "/archive", "/chimurai4", 1) + ".warc"

	resp, err := r.claimGet(b, warcURL, headers)
	if err != nil {
		return err
	}
	if resp == nil {
		return nil
	}
	if !resp.OK() {
		return fmt.Errorf("WARC returned HTTP %d", resp.StatusCode)
	}

	for _, targetURI := range parseWARCTargets(resp.Body) {
		r.add(SourceGhostArchive, targetURI, "", "", "")
	}
	return nil
}

// parseWARCTargets extracts the target URI of every response record in a WARC.
//
// WARC record headers are ASCII lines even when the payload that follows is
// binary, so a line scan is enough. Record header order is not guaranteed, so
// the record's type and target URI are tracked independently and the URI is
// taken once both are known. A `WARC/` version line starts a new record.
//
// Only response records count: a request record names the same URL, and
// metadata records carry the archive's own bookkeeping.
func parseWARCTargets(body []byte) []string {
	var (
		targets    []string
		seen       = make(map[string]bool)
		isResponse bool
		target     string
		recorded   bool
	)

	// take records the current record's target once it is known to be a
	// response, and only once, so a payload line that happens to look like a
	// WARC header cannot add a second URL.
	take := func() {
		if !isResponse || target == "" || recorded {
			return
		}
		recorded = true
		if seen[target] {
			return
		}
		seen[target] = true
		targets = append(targets, target)
	}

	sc := bufio.NewScanner(bytes.NewReader(body))
	// WARC payloads can be large; give the scanner room so one long line does
	// not abort the scan mid-file.
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for sc.Scan() {
		// Work on the scanner's bytes rather than a string per line. Most lines
		// in a WARC are payload, not headers, and a WARC has one line per line
		// of every archived response body.
		line := bytes.TrimRight(sc.Bytes(), "\r")

		// Only header lines start with "WARC", so that prefix gates the more
		// expensive comparisons away from the payload.
		if !bytes.HasPrefix(line, warcPrefix) {
			continue
		}

		switch {
		case bytes.HasPrefix(line, warcVersionPrefix):
			isResponse, target, recorded = false, "", false
		case bytes.EqualFold(bytes.TrimSpace(line), warcResponseType):
			isResponse = true
			take()
		default:
			if match := warcTargetPattern.FindSubmatch(line); match != nil {
				target = strings.Trim(string(match[1]), `<>"`)
				take()
			}
		}
	}
	return targets
}
