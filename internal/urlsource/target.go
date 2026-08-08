package urlsource

import (
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

// Target is a parsed input: a hostname, plus an optional path prefix when the
// caller scoped the search to one part of a site.
type Target struct {
	// Raw is the input exactly as the user typed it.
	Raw string
	// Host is the hostname, lowercased and punycode-encoded.
	Host string
	// Path is the path prefix including its leading slash, or "" when the
	// input was a bare hostname. Case is preserved, because URL paths are
	// case-sensitive even though hostnames are not.
	Path string
	// Registrable is the eTLD+1 ("example.co.uk" for "a.b.example.co.uk").
	Registrable string
}

// ParseTarget splits an input into hostname and optional path prefix.
//
// A scheme is tolerated and dropped, a lone trailing slash is treated as "no
// path", and only the hostname is lowercased.
func ParseTarget(raw string) (Target, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Target{}, fmt.Errorf("empty target")
	}

	// Drop a scheme if present, so both example.com and https://example.com work.
	if idx := strings.Index(trimmed, "://"); idx >= 0 {
		trimmed = trimmed[idx+3:]
	}
	// A query or fragment scopes nothing useful for an archive search.
	trimmed = strings.SplitN(trimmed, "?", 2)[0]
	trimmed = strings.SplitN(trimmed, "#", 2)[0]
	trimmed = strings.Trim(trimmed, ".")

	host, path := trimmed, ""
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		host = trimmed[:idx]
		path = trimmed[idx:]
		// A lone trailing slash means "the whole site", not a path prefix.
		if path == "/" {
			path = ""
		}
	}

	// Strip any port: archives index by hostname, and :80/:443 in particular
	// would never match a stored capture.
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		host = host[:idx]
	}

	host = strings.ToLower(strings.Trim(host, "."))
	if host == "" {
		return Target{}, fmt.Errorf("target %q has no hostname", raw)
	}
	if !strings.Contains(host, ".") {
		return Target{}, fmt.Errorf("target %q is not a domain (expected something like example.com)", raw)
	}
	if strings.ContainsAny(host, " \t") {
		return Target{}, fmt.Errorf("target %q contains whitespace in the hostname", raw)
	}

	// Archives store unicode domains in punycode, so a unicode input would
	// otherwise silently match nothing.
	if ascii, err := idna.ToASCII(host); err == nil && ascii != "" {
		host = ascii
	}

	registrable := host
	if etld1, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil && etld1 != "" {
		registrable = etld1
	}

	return Target{Raw: raw, Host: host, Path: path, Registrable: registrable}, nil
}

// HasPath reports whether the input scoped the search to a path prefix.
func (t Target) HasPath() bool { return t.Path != "" }

// IsSubdomain reports whether the host sits below its registrable domain.
func (t Target) IsSubdomain() bool { return t.Host != t.Registrable }

// CDXScope renders the url= value for CDX-style APIs (Wayback, Common Crawl).
//
// With subdomains enabled and no path prefix this is `*.host/*`, which is what
// makes these sources return a target's whole estate instead of a single
// hostname. A path prefix pins the host, so the subdomain wildcard would only
// widen the scope past what the user asked for and is not applied.
func (t Target) CDXScope(includeSubs bool) string {
	if t.HasPath() {
		return t.Host + t.Path + "*"
	}
	if includeSubs {
		return "*." + t.Host + "/*"
	}
	return t.Host + "/*"
}

// EscapeScope percent-encodes a CDX url= value while leaving `*` and `/`
// intact. Both characters are load-bearing: the CDX APIs infer their match type
// from the wildcard, and encoding it changes the query's meaning.
func EscapeScope(scope string) string {
	escaped := url.QueryEscape(scope)
	escaped = strings.ReplaceAll(escaped, "%2A", "*")
	escaped = strings.ReplaceAll(escaped, "%2F", "/")
	return escaped
}

// OTXIndicator returns the OTX indicator type to query t.Host as. OTX has no
// wildcard, so the closest match for the requested scope is a domain lookup
// when the input is a registrable domain and a hostname lookup otherwise.
// Widening a subdomain input to its parent would return URLs outside scope.
func (t Target) OTXIndicator() string {
	if t.IsSubdomain() {
		return "hostname"
	}
	return "domain"
}

// GhostArchiveTerm renders the search term for ghostarchive.org. A registrable
// domain is prefixed with a dot so the search anchors on a label boundary and
// does not also match hosts that merely end with the same string.
func (t Target) GhostArchiveTerm() string {
	if t.IsSubdomain() {
		return t.Host
	}
	return "." + t.Host
}
