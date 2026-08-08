package cert

import (
	"fmt"
	"strconv"
	"strings"
)

// maxDetailCommonNames caps the Common Names line so a domain with hundreds of
// certificates stays readable.
const maxDetailCommonNames = 3

// Text renders the bare domain, or a full certificate block under --detail.
func (g DomainGroup) Text() []string {
	if !g.Detail {
		return []string{g.Domain}
	}

	lines := []string{fmt.Sprintf("Domain: %s", g.Domain)}
	if len(g.CertIDs) > 0 {
		lines = append(lines, fmt.Sprintf("  Certificates (%d): %s", g.Count, strings.Join(g.CertIDs, ", ")))
	}
	if g.FirstSeen != "" {
		lines = append(lines, fmt.Sprintf("  Not Before:   %s", g.FirstSeen))
	}
	if g.LastExpires != "" {
		lines = append(lines, fmt.Sprintf("  Not After:    %s", g.LastExpires))
	}
	if len(g.Issuers) > 0 {
		lines = append(lines, fmt.Sprintf("  Issuers:      %s", strings.Join(g.Issuers, ", ")))
	}
	if len(g.CommonNames) > 0 && len(g.CommonNames) <= maxDetailCommonNames {
		lines = append(lines, fmt.Sprintf("  Common Names: %s", strings.Join(g.CommonNames, ", ")))
	}

	// Trailing blank line separates consecutive blocks.
	return append(lines, "")
}

// Flat renders the bare domain, ignoring --detail so pipelines stay stable.
func (g DomainGroup) Flat() []string {
	return []string{g.Domain}
}

// CSV renders the domain with its certificate metadata.
func (g DomainGroup) CSV() ([]string, [][]string) {
	return []string{"domain", "certificates", "first_seen", "last_expires", "issuers"},
		[][]string{{
			g.Domain,
			strconv.Itoa(g.Count),
			g.FirstSeen,
			g.LastExpires,
			strings.Join(g.Issuers, "; "),
		}}
}
