// Package cdn classifies IP addresses as belonging to a CDN or WAF provider.
package cdn

import (
	"net"
	"net/url"
	"strings"

	"github.com/j3ssie/metabigor/internal/output"
	"github.com/projectdiscovery/cdncheck"
)

// Placeholders for the fields cdncheck leaves blank on an unmatched address.
const (
	unknownVendor = "unknown"
	noType        = "none"
)

// Result is one classified address.
type Result struct {
	IP     string `json:"ip"`
	IsCDN  bool   `json:"is_cdn"`
	Vendor string `json:"vendor"`
	Type   string `json:"type"`
}

// Classify resolves a target to an address and reports whether a CDN or WAF
// serves it. It returns false when the target is not an IP address at all.
func Classify(client *cdncheck.Client, target string) (Result, bool) {
	ip := Host(target)
	parsed := net.ParseIP(ip)
	if parsed == nil {
		output.Debug("Not an IP address, skipping: %s", target)
		return Result{}, false
	}

	matched, vendor, kind, err := client.Check(parsed)
	if err != nil {
		output.Debug("CDN check failed for %s: %v", ip, err)
		return Result{}, false
	}

	result := Result{
		IP:     ip,
		IsCDN:  matched && (kind == "cdn" || kind == "waf"),
		Vendor: vendor,
		Type:   kind,
	}
	if result.Vendor == "" {
		result.Vendor = unknownVendor
	}
	if result.Type == "" {
		result.Type = noType
	}
	return result, true
}

// Host accepts either a bare address or a URL and returns the host part.
func Host(target string) string {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return target
	}
	u, err := url.Parse(target)
	if err != nil {
		output.Debug("Cannot parse URL %s: %v", target, err)
		return target
	}
	return u.Hostname()
}

// Text renders the address with the vendor and protection type.
func (r Result) Text() []string {
	return []string{output.Columns(r.IP, r.Vendor, r.Type)}
}

// Flat renders the bare IP, which is what --exclude and --only are usually for.
func (r Result) Flat() []string {
	return []string{r.IP}
}

// CSV renders the classification as columns.
func (r Result) CSV() ([]string, [][]string) {
	isCDN := "false"
	if r.IsCDN {
		isCDN = "true"
	}
	return []string{"ip", "is_cdn", "vendor", "type"},
		[][]string{{r.IP, isCDN, r.Vendor, r.Type}}
}
