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
		return "Domain"
	case TypeOrg:
		return "Org"
	default:
		return "Unknown"
	}
}

// DetectType auto-detects the input type.
func DetectType(input string) InputType {
	input = strings.TrimSpace(input)

	// AS12345 or AS\d+
	if asnPattern.MatchString(input) {
		output.Debug("Detected input %q as ASN (matches AS pattern)", input)
		return TypeASN
	}
	// Pure number → ASN
	if isNumeric(input) {
		output.Debug("Detected input %q as ASN (pure number)", input)
		return TypeASN
	}
	// CIDR notation
	if _, _, err := net.ParseCIDR(input); err == nil {
		output.Debug("Detected input %q as CIDR", input)
		return TypeCIDR
	}
	// Valid IP
	if ip := net.ParseIP(input); ip != nil {
		output.Debug("Detected input %q as IP", input)
		return TypeIP
	}
	// Has a dot and looks like a domain
	if strings.Contains(input, ".") && isDomainLike(input) {
		output.Debug("Detected input %q as Domain", input)
		return TypeDomain
	}
	// Fallback: org search
	output.Debug("Detected input %q as Org (fallback)", input)
	return TypeOrg
}

// StaticLookup queries the local ASN database for the given input.
func StaticLookup(db *asndb.DB, input string, forceType InputType) []string {
	if forceType == TypeUnknown {
		forceType = DetectType(input)
	}
	output.Verbose("Static lookup: input=%q type=%s", input, forceType)

	var results []string
	switch forceType {
	case TypeASN:
		results = lookupASN(db, input)
	case TypeIP:
		results = lookupIP(db, input)
	case TypeCIDR:
		results = expandCIDR(input)
	case TypeDomain:
		results = lookupDomain(db, input)
	case TypeOrg:
		results = searchOrg(db, input)
	}

	output.Verbose("Static lookup for %q: %d results", input, len(results))
	if len(results) == 0 {
		output.Warn("No results found for %q (type=%s)", input, forceType)
	}
	return results
}

func lookupASN(db *asndb.DB, input string) []string {
	records := db.LookupASN(input)
	output.Debug("ASN lookup for %s matched %d records in DB", input, len(records))
	var results []string
	for _, r := range records {
		if r.CIDR != "" {
			results = append(results, r.CIDR)
		} else {
			results = append(results, fmt.Sprintf("%s - %s", r.StartIP, r.EndIP))
		}
	}
	output.Verbose("ASN lookup for %s: %d CIDRs", input, len(results))
	return results
}

func lookupIP(db *asndb.DB, input string) []string {
	rec := db.LookupIP(input)
	if rec == nil {
		output.Debug("IP lookup for %s: no matching record", input)
		return nil
	}
	output.Debug("IP lookup for %s: AS%s %s %s", input, rec.ASN, rec.CIDR, rec.Description())
	return []string{fmt.Sprintf("AS%s | %s | %s | %s", rec.ASN, rec.CIDR, rec.Description(), rec.CountryCode)}
}

func expandCIDR(input string) []string {
	ip, ipNet, err := net.ParseCIDR(input)
	if err != nil {
		return nil
	}
	var results []string
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		results = append(results, ip.String())
	}
	output.Debug("CIDR %s expanded to %d IPs", input, len(results))
	return results
}

func lookupDomain(db *asndb.DB, input string) []string {
	output.Debug("Resolving domain %s", input)
	records := db.LookupDomain(input)
	output.Debug("Domain %s resolved to %d ASN records", input, len(records))
	var results []string
	for _, r := range records {
		if r.CIDR != "" {
			results = append(results, r.CIDR)
		}
	}
	return results
}

func searchOrg(db *asndb.DB, input string) []string {
	records := db.SearchOrg(input)
	output.Debug("Org search for %q matched %d records", input, len(records))
	var results []string
	for _, r := range records {
		if r.CIDR != "" {
			results = append(results, fmt.Sprintf("AS%s | %s | %s", r.ASN, r.CIDR, r.Description()))
		}
	}
	return results
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

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
