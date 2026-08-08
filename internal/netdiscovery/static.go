package netdiscovery

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/j3ssie/metabigor/internal/asndb"
	"github.com/j3ssie/metabigor/internal/output"
)

var asnPattern = regexp.MustCompile(`(?i)^AS\d+$`)

// sourceLocal marks results served from the on-disk ASN database.
const sourceLocal = "local-db"

// CountryLookup fills in a country code for a CIDR when the ASN record has
// none. It may be nil, in which case the country column is simply left blank.
type CountryLookup func(cidr string) string

// InputType classifies what an input string represents.
type InputType int

const (
	// TypeUnknown represents an unclassified input type.
	TypeUnknown InputType = iota
	// TypeASN represents an Autonomous System Number input.
	TypeASN
	// TypeIP represents an IP address input.
	TypeIP
	// TypeCIDR represents a CIDR notation input.
	TypeCIDR
	// TypeDomain represents a domain name input.
	TypeDomain
	// TypeOrg represents an organization name input.
	TypeOrg
)

// String returns a human-readable name for the input type.
func (t InputType) String() string {
	switch t {
	case TypeASN:
		return "ASN"
	case TypeIP:
		return "IP"
	case TypeCIDR:
		return "CIDR"
	case TypeDomain:
		return "domain"
	case TypeOrg:
		return "organization"
	default:
		return "unknown"
	}
}

// DetectType auto-detects the input type.
func DetectType(input string) InputType {
	input = strings.TrimSpace(input)

	// AS12345 or a bare AS number
	if asnPattern.MatchString(input) {
		output.Debug("Detected %q as ASN (matches AS pattern)", input)
		return TypeASN
	}
	if isNumeric(input) {
		output.Debug("Detected %q as ASN (bare number)", input)
		return TypeASN
	}
	if isCIDR(input) {
		output.Debug("Detected %q as CIDR", input)
		return TypeCIDR
	}
	if ip := net.ParseIP(input); ip != nil {
		output.Debug("Detected %q as IP", input)
		return TypeIP
	}
	if strings.Contains(input, ".") && isDomainLike(input) {
		output.Debug("Detected %q as domain", input)
		return TypeDomain
	}
	output.Debug("Detected %q as organization (fallback)", input)
	return TypeOrg
}

// StaticLookup queries the local ASN database for the given input. Pass
// TypeUnknown to auto-detect. country may be nil.
func StaticLookup(db *asndb.DB, input string, forceType InputType, country CountryLookup) []Result {
	if forceType == TypeUnknown {
		forceType = DetectType(input)
	}
	output.Verbose("Local lookup: input=%q type=%s", input, forceType)

	var records []asndb.ASNRecord
	switch forceType {
	case TypeASN:
		records = db.LookupASN(input)
	case TypeIP, TypeCIDR:
		// A CIDR resolves through its first address, so `net 1.1.1.0/24`
		// reports the owning network rather than expanding to 256 hosts.
		if rec := db.LookupTarget(input); rec != nil {
			records = []asndb.ASNRecord{*rec}
		}
	case TypeDomain:
		records = db.LookupDomain(input)
	case TypeOrg:
		records = db.SearchOrg(input)
	}

	results := make([]Result, 0, len(records))
	for _, rec := range records {
		if rec.CIDR == "" {
			continue
		}
		results = append(results, newResult(input, rec, country))
	}

	output.Verbose("Local lookup for %q: %d result(s)", input, len(results))
	if len(results) == 0 {
		output.Warn("No results for %q (detected as %s)", input, forceType)
	}
	return results
}

// newResult converts a database record into a renderable result, consulting
// the country lookup only when the ASN record itself has no country code.
func newResult(input string, rec asndb.ASNRecord, country CountryLookup) Result {
	code := rec.CountryCode
	if code == "" && country != nil {
		code = country(rec.CIDR)
	}
	return Result{
		Input:   input,
		ASN:     fmt.Sprintf("AS%s", rec.ASN),
		CIDR:    rec.CIDR,
		Org:     rec.Description(),
		Country: code,
		Source:  sourceLocal,
	}
}

// isCIDR reports whether s is CIDR notation.
func isCIDR(s string) bool {
	_, _, err := net.ParseCIDR(s)
	return err == nil
}

// isNetworkValue reports whether s names a network: an ASN or a CIDR. It is the
// single rule for what counts as a usable bgp.he.net cell.
func isNetworkValue(s string) bool {
	return asnPattern.MatchString(s) || isCIDR(s)
}

func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func isDomainLike(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return false
	}
	tld := parts[len(parts)-1]
	return len(tld) >= 2 && len(tld) <= 10
}
