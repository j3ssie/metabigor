package urlsource

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// revisitMIME marks a capture that only records "unchanged since last time".
// It is never a distinct URL, so every exclusion list drops it.
const revisitMIME = "warc/revisit"

// JunkExtensions is the extension list `--blacklist default` expands to: static
// assets that inflate a URL list without adding attack surface.
var JunkExtensions = []string{
	"css", "jpg", "jpeg", "png", "gif", "svg", "webp", "bmp", "ico", "tif", "tiff", "avif",
	"woff", "woff2", "ttf", "otf", "eot",
	"mp3", "mp4", "m4a", "m4p", "wav", "wma", "avi", "flv", "mov", "webm", "ogv", "wmv", "asx",
}

// Filters narrows results, both by pushing conditions into the source APIs and
// by re-checking locally. Callers must call Compile before use.
//
// Match wins over Filter when both are set for the same field, because the
// archive APIs cannot express an include-list and an exclude-list at once.
type Filters struct {
	MatchStatus  []string
	FilterStatus []string
	MatchMIME    []string
	FilterMIME   []string
	From         string
	To           string
	Keywords     string
	Blacklist    []string
	NoParams     bool

	keywordRE *regexp.Regexp
	blacklist map[string]bool
	fromTS    string
	toTS      string
}

// Compile validates the filter values and precomputes what the hot path needs.
func (f *Filters) Compile() error {
	if f.Keywords != "" {
		// Case-insensitive, matching the upstream tools. A bad pattern here is
		// almost always a shell expansion accident, so the error says so.
		re, err := regexp.Compile("(?i)" + f.Keywords)
		if err != nil {
			return fmt.Errorf("--keywords is not a valid regex: %w\n"+
				"tip: use single quotes so the shell leaves $ and () alone, e.g. --keywords '\\.js(\\?|$)'", err)
		}
		f.keywordRE = re
	}

	var err error
	if f.fromTS, err = padTimestamp(f.From, '0'); err != nil {
		return fmt.Errorf("--from: %w", err)
	}
	if f.toTS, err = padTimestamp(f.To, '9'); err != nil {
		return fmt.Errorf("--to: %w", err)
	}
	if f.fromTS != "" && f.toTS != "" && f.fromTS > f.toTS {
		return fmt.Errorf("--from %s is later than --to %s", f.From, f.To)
	}

	f.blacklist = make(map[string]bool)
	for _, raw := range f.Blacklist {
		for _, ext := range expandBlacklist(raw) {
			f.blacklist[ext] = true
		}
	}

	f.MatchStatus = cleanList(f.MatchStatus)
	f.FilterStatus = cleanList(f.FilterStatus)
	f.MatchMIME = lowerList(cleanList(f.MatchMIME))
	f.FilterMIME = lowerList(cleanList(f.FilterMIME))
	return nil
}

// expandBlacklist normalizes one --blacklist entry, expanding "default" to the
// built-in junk list.
func expandBlacklist(raw string) []string {
	ext := strings.ToLower(strings.TrimSpace(raw))
	ext = strings.TrimPrefix(ext, ".")
	switch ext {
	case "":
		return nil
	case "default":
		return JunkExtensions
	default:
		return []string{ext}
	}
}

// padTimestamp normalizes a partial date (2016, 201803, 20180321) to the 14
// digits the CDX indexes use. from-dates pad with zeros to reach the earliest
// instant in the period; to-dates pad with nines to reach the latest.
func padTimestamp(s string, pad byte) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("%q is not a date (expected digits like 2016, 201803, or 20180321)", s)
		}
	}
	if len(s) > 14 {
		return "", fmt.Errorf("%q is too long (at most 14 digits: YYYYMMDDhhmmss)", s)
	}
	return s + strings.Repeat(string(pad), 14-len(s)), nil
}

// WaybackParams renders the server-side filters for the Wayback CDX API, whose
// dialect is `filter=[!]field:regex` with alternation inside the value.
func (f *Filters) WaybackParams() string {
	q := url.Values{}
	if f.From != "" {
		q.Add("from", f.From)
	}
	if f.To != "" {
		q.Add("to", f.To)
	}

	if len(f.MatchMIME) > 0 {
		q.Add("filter", "mimetype:"+alternation(f.MatchMIME))
	} else {
		q.Add("filter", "!mimetype:"+alternation(append([]string{revisitMIME}, f.FilterMIME...)))
	}

	if len(f.MatchStatus) > 0 {
		q.Add("filter", "statuscode:"+alternation(f.MatchStatus))
	} else if len(f.FilterStatus) > 0 {
		q.Add("filter", "!statuscode:"+alternation(f.FilterStatus))
	}

	if f.Keywords != "" {
		q.Add("filter", "original:"+cdxKeywordPattern(f.Keywords))
	}
	return encodeParams(q)
}

// CommonCrawlParams renders the same conditions in Common Crawl's dialect,
// which wraps the regex in parentheses and marks it with a leading `~`.
//
// Common Crawl has no from/to parameters, so dates are handled by skipping
// whole index collections and by re-checking each record's timestamp locally.
func (f *Filters) CommonCrawlParams() string {
	q := url.Values{}
	if len(f.MatchMIME) > 0 {
		q.Add("filter", "~mime:("+alternation(f.MatchMIME)+")")
	} else {
		q.Add("filter", "!~mime:("+alternation(append([]string{revisitMIME}, f.FilterMIME...))+")")
	}

	if len(f.MatchStatus) > 0 {
		q.Add("filter", "~status:("+alternation(f.MatchStatus)+")")
	} else if len(f.FilterStatus) > 0 {
		q.Add("filter", "!~status:("+alternation(f.FilterStatus)+")")
	}

	if f.Keywords != "" {
		q.Add("filter", "~url:"+cdxKeywordPattern(f.Keywords))
	}
	return encodeParams(q)
}

// encodeParams renders query parameters as a suffix ready to append to a URL
// that already has a query string.
func encodeParams(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	return "&" + q.Encode()
}

// alternation escapes each value and joins them into a regex alternation.
//
// The escaped `\+` is rewritten to `.` because the CDX APIs reject a
// backslash-escaped plus, which would otherwise break every MIME type that
// contains one, such as image/svg+xml.
func alternation(values []string) string {
	escaped := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		escaped = append(escaped, strings.ReplaceAll(regexp.QuoteMeta(v), `\+`, "."))
	}
	return strings.Join(escaped, "|")
}

// cdxKeywordPattern wraps a user regex for the CDX filter dialects, which match
// with "anchored at the start, unanchored at the end" semantics.
//
// A `.*` prefix always goes on, so the pattern can match anywhere in the URL. A
// trailing `.*` goes on too, unless the pattern already anchors the end with an
// unescaped `$`, where it would prevent any match.
func cdxKeywordPattern(pattern string) string {
	suffix := ".*"
	if hasUnescapedDollar(pattern) {
		suffix = ""
	}
	return ".*(" + pattern + ")" + suffix
}

// hasUnescapedDollar reports whether the pattern contains a `$` acting as an
// end anchor rather than a literal. Go's regexp has no lookbehind, so this
// counts the preceding backslashes: an even count leaves the `$` unescaped.
func hasUnescapedDollar(pattern string) bool {
	for i, r := range pattern {
		if r != '$' {
			continue
		}
		backslashes := 0
		for j := i - 1; j >= 0 && pattern[j] == '\\'; j-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return true
		}
	}
	return false
}

// Normalize cleans a URL as an archive reported it, returning "" for input
// that cannot be salvaged.
func Normalize(raw string) string {
	u := strings.TrimSpace(raw)
	if u == "" {
		return ""
	}

	// archive.org occasionally appends %0A (an encoded newline) and trailing
	// junk. The URL only resolves once that tail is cut. Both realistic casings
	// are checked directly rather than lowercasing every URL in the run.
	if i := strings.Index(u, "%0A"); i > 0 {
		u = u[:i]
	} else if i := strings.Index(u, "%0a"); i > 0 {
		u = u[:i]
	}
	u = strings.TrimRight(u, "\r\n")

	// A default port never matches how an archive stored the capture, and it
	// splits one endpoint into two results.
	if parsed, err := url.Parse(u); err == nil && parsed.Host != "" {
		if port := parsed.Port(); port == "80" || port == "443" {
			parsed.Host = parsed.Hostname()
			u = parsed.String()
		}
	}
	return u
}

// bannerHeadBytes is how much of a response body is searched for an error
// notice. Every such notice these APIs return is a short page, so scanning
// further would only cost work on the large bodies it can never match.
const bannerHeadBytes = 4096

// containsFoldPrefix reports whether the head of body contains sub, ignoring
// case. It exists so a banner check on a multi-megabyte response does not
// lowercase the whole thing.
func containsFoldPrefix(body []byte, sub string) bool {
	head := body
	if len(head) > bannerHeadBytes {
		head = head[:bannerHeadBytes]
	}
	return bytes.Contains(bytes.ToLower(head), []byte(sub))
}

// InScope reports whether a discovered URL actually belongs to the target.
//
// Archives routinely return neighbours of the requested host - GhostArchive's
// search in particular is a text match - so every result is re-checked. The
// host is compared on a label boundary rather than as a substring, because a
// substring check would accept notexample.com as example.com.
//
// A path-scoped target additionally requires the path prefix. Both paths are
// percent-decoded first, so a URL containing %20 still matches an input typed
// with a literal space.
func InScope(candidate string, t Target) bool {
	host := hostOf(candidate)
	if host == "" {
		return false
	}
	if !hostInScope(host, t.Host) {
		return false
	}
	if !t.HasPath() {
		return true
	}
	return strings.HasPrefix(strings.ToLower(decode(pathOf(candidate))), strings.ToLower(decode(t.Path)))
}

// hostInScope reports whether host is the target host or one of its subdomains.
func hostInScope(host, target string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	target = strings.ToLower(target)
	return host == target || strings.HasSuffix(host, "."+target)
}

// hostOf extracts the hostname from a URL, tolerating a missing scheme.
func hostOf(raw string) string {
	if parsed, err := url.Parse(raw); err == nil && parsed.Hostname() != "" {
		return parsed.Hostname()
	}

	// Without a scheme, url.Parse puts everything in Path, so split by hand.
	trimmed := raw
	if i := strings.Index(trimmed, "://"); i >= 0 {
		trimmed = trimmed[i+3:]
	}
	if i := strings.IndexAny(trimmed, "/?#"); i >= 0 {
		trimmed = trimmed[:i]
	}
	if i := strings.LastIndex(trimmed, "@"); i >= 0 {
		trimmed = trimmed[i+1:]
	}
	if i := strings.LastIndex(trimmed, ":"); i >= 0 {
		trimmed = trimmed[:i]
	}
	return strings.TrimSpace(trimmed)
}

// pathOf extracts the path from a URL, tolerating a missing scheme.
func pathOf(raw string) string {
	if parsed, err := url.Parse(raw); err == nil && parsed.Hostname() != "" {
		return parsed.Path
	}

	trimmed := raw
	if i := strings.Index(trimmed, "://"); i >= 0 {
		trimmed = trimmed[i+3:]
	}
	if i := strings.Index(trimmed, "/"); i >= 0 {
		return strings.SplitN(trimmed[i:], "?", 2)[0]
	}
	return ""
}

func decode(s string) string {
	if decoded, err := url.QueryUnescape(s); err == nil {
		return decoded
	}
	return s
}

// Allow reports whether a result passes the status, MIME, extension, and
// keyword filters.
//
// An empty status or MIME means the source did not report one; those results
// are kept rather than dropped, since a missing field is not evidence against
// the URL. Callers that want strict filtering should exclude those sources.
func (f *Filters) Allow(rawURL, status, mime string) bool {
	if status != "" {
		if len(f.MatchStatus) > 0 {
			if !containsFold(f.MatchStatus, status) {
				return false
			}
		} else if containsFold(f.FilterStatus, status) {
			return false
		}
	}

	if mime != "" {
		mime = strings.ToLower(mime)
		if mime == revisitMIME {
			return false
		}
		if len(f.MatchMIME) > 0 {
			if !containsFold(f.MatchMIME, mime) {
				return false
			}
		} else if containsFold(f.FilterMIME, mime) {
			return false
		}
	}

	if len(f.blacklist) > 0 && f.blacklist[extensionOf(rawURL)] {
		return false
	}

	if f.keywordRE != nil && !f.keywordRE.MatchString(rawURL) {
		return false
	}
	return true
}

// AllowTimestamp reports whether a capture falls inside --from/--to. An empty
// timestamp is kept, because a source that reports no date is not evidence the
// URL is out of range.
func (f *Filters) AllowTimestamp(ts string) bool {
	if ts == "" {
		return true
	}
	if f.fromTS != "" && ts < f.fromTS {
		return false
	}
	if f.toTS != "" && ts > f.toTS {
		return false
	}
	return true
}

// HasDateRange reports whether any date bound was set.
func (f *Filters) HasDateRange() bool { return f.fromTS != "" || f.toTS != "" }

// FromYear returns the four-digit year of --from, or 0 when unset.
func (f *Filters) FromYear() int { return yearOf(f.fromTS) }

// ToYear returns the four-digit year of --to, or 0 when unset.
func (f *Filters) ToYear() int { return yearOf(f.toTS) }

func yearOf(ts string) int {
	if len(ts) < 4 {
		return 0
	}
	year := 0
	for _, r := range ts[:4] {
		year = year*10 + int(r-'0')
	}
	return year
}

// extensionOf returns the lowercase extension of a URL path, without its dot.
func extensionOf(raw string) string {
	trimmed := raw
	if i := strings.IndexAny(trimmed, "?#"); i >= 0 {
		trimmed = trimmed[:i]
	}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Path != "" {
		trimmed = parsed.Path
	}
	return strings.ToLower(strings.TrimPrefix(path.Ext(trimmed), "."))
}

// EndpointKey renders the host+path identity used by --no-params, so URLs that
// differ only in their query string collapse to one result.
func EndpointKey(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return strings.ToLower(parsed.Host) + parsed.Path
}

// TimestampFromISO converts the date formats the JSON sources report
// ("2021-03-04T05:06:07", "2021-03-04 05:06:07", "2021-03-04") into the
// 14-digit form the CDX indexes use, so one comparison covers every source.
// It returns "" when the value carries less than a full date.
func TimestampFromISO(s string) string {
	digits := make([]byte, 0, 14)
	for i := 0; i < len(s) && len(digits) < 14; i++ {
		if s[i] >= '0' && s[i] <= '9' {
			digits = append(digits, s[i])
		}
	}
	if len(digits) < 8 {
		return ""
	}
	return string(digits) + strings.Repeat("0", 14-len(digits))
}

func containsFold(list []string, value string) bool {
	for _, item := range list {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		// A single flag value may carry a comma-separated list.
		for _, part := range strings.Split(v, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func lowerList(values []string) []string {
	for i, v := range values {
		values[i] = strings.ToLower(v)
	}
	return values
}
