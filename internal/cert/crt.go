// Package cert provides certificate transparency log searching via crt.sh.
package cert

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

// CertEntry represents a certificate transparency log entry.
//
//nolint:revive // Keeping CertEntry name for backward compatibility
type CertEntry struct {
	CertID             string   `json:"cert_id"`
	LoggedAt           string   `json:"logged_at,omitempty"`
	NotBefore          string   `json:"not_before,omitempty"`
	NotAfter           string   `json:"not_after,omitempty"`
	CommonName         string   `json:"common_name"`
	MatchingIdentities []string `json:"matching_identities,omitempty"`
	IssuerName         string   `json:"issuer_name,omitempty"`

	// Deprecated: Use MatchingIdentities instead
	Domain string `json:"domain,omitempty"`
	// Deprecated: Organization info not reliably available
	Org string `json:"org,omitempty"`
}

// DomainGroup represents a domain with all associated certificates.
type DomainGroup struct {
	Domain      string   `json:"domain"`
	CertIDs     []string `json:"cert_ids"`
	Count       int      `json:"count"`
	FirstSeen   string   `json:"first_seen,omitempty"`   // Earliest NotBefore
	LastExpires string   `json:"last_expires,omitempty"` // Latest NotAfter
	Issuers     []string `json:"issuers,omitempty"`      // Unique issuers
	CommonNames []string `json:"common_names,omitempty"` // Associated CNs
}

// SearchCRT queries crt.sh and returns certificate entries.
func SearchCRT(client *retryablehttp.Client, query string, clean, wildcard bool) []CertEntry {
	entries := tryCRT(client, fmt.Sprintf("https://crt.sh/?O=%s", url.QueryEscape(query)))
	if len(entries) == 0 {
		entries = tryCRT(client, fmt.Sprintf("https://crt.sh/?q=%s", url.QueryEscape(query)))
	}

	if clean || wildcard {
		entries = filterEntries(entries, clean, wildcard)
	}
	return entries
}

func tryCRT(client *retryablehttp.Client, targetURL string) []CertEntry {
	body, err := httpclient.Get(client, targetURL)
	if err != nil {
		output.Debug("crt.sh error: %v", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		output.Debug("crt.sh parse error: %v", err)
		return nil
	}

	var entries []CertEntry

	doc.Find("table tr").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return // skip header
		}
		tds := s.Find("td")
		if tds.Length() < 6 {
			return // need at least 6 columns
		}

		// Get cert ID - try link first, fallback to text
		certID := strings.TrimSpace(tds.Eq(0).Find("a").Text())
		if certID == "" {
			certID = strings.TrimSpace(tds.Eq(0).Text())
		}

		// Skip rows with invalid cert IDs (expanded detail rows or headers)
		// Valid cert IDs are numeric
		if certID == "" || !isNumeric(certID) {
			return
		}

		// Get issuer name if column 6 exists
		issuerName := ""
		if tds.Length() >= 7 {
			issuerName = strings.TrimSpace(tds.Eq(6).Find("a").Text())
			if issuerName == "" {
				issuerName = strings.TrimSpace(tds.Eq(6).Text())
			}
		}

		entry := CertEntry{
			CertID:     certID,
			LoggedAt:   strings.TrimSpace(tds.Eq(1).Text()),
			NotBefore:  strings.TrimSpace(tds.Eq(2).Text()),
			NotAfter:   strings.TrimSpace(tds.Eq(3).Text()),
			CommonName: strings.TrimSpace(tds.Eq(4).Text()),
			IssuerName: issuerName,
		}

		// Parse Matching Identities (Column 5) - may contain multiple domains separated by <BR>
		identitiesHTML, _ := tds.Eq(5).Html()
		identities := strings.Split(identitiesHTML, "<br/>")
		for _, id := range identities {
			cleaned := strings.TrimSpace(stripHTMLTags(id))
			cleaned = strings.ToLower(cleaned)
			if cleaned != "" {
				entry.MatchingIdentities = append(entry.MatchingIdentities, cleaned)
			}
		}

		// Populate deprecated Domain field for backward compatibility
		if len(entry.MatchingIdentities) > 0 {
			entry.Domain = entry.MatchingIdentities[0]
		} else {
			entry.Domain = strings.ToLower(strings.TrimSpace(entry.CommonName))
		}

		entries = append(entries, entry)
	})

	output.Verbose("crt.sh returned %d certificate entries", len(entries))
	return entries
}

// stripHTMLTags removes HTML tags from a string.
func stripHTMLTags(html string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(html, "")
}

// isNumeric checks if a string contains only digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// GroupByDomain groups certificate entries by domain, aggregating metadata.
func GroupByDomain(entries []CertEntry) []DomainGroup {
	domainMap := make(map[string]*DomainGroup)

	for _, entry := range entries {
		// Process all matching identities
		for _, domain := range entry.MatchingIdentities {
			if _, exists := domainMap[domain]; !exists {
				domainMap[domain] = &DomainGroup{
					Domain:      domain,
					CertIDs:     []string{},
					Issuers:     []string{},
					CommonNames: []string{},
				}
			}

			group := domainMap[domain]

			// Add cert ID if not already present
			if !contains(group.CertIDs, entry.CertID) {
				group.CertIDs = append(group.CertIDs, entry.CertID)
			}

			// Track earliest NotBefore
			if group.FirstSeen == "" || (entry.NotBefore != "" && entry.NotBefore < group.FirstSeen) {
				group.FirstSeen = entry.NotBefore
			}

			// Track latest NotAfter
			if group.LastExpires == "" || (entry.NotAfter != "" && entry.NotAfter > group.LastExpires) {
				group.LastExpires = entry.NotAfter
			}

			// Track unique issuers
			if entry.IssuerName != "" && !contains(group.Issuers, entry.IssuerName) {
				group.Issuers = append(group.Issuers, entry.IssuerName)
			}

			// Track unique common names
			if entry.CommonName != "" && !contains(group.CommonNames, entry.CommonName) {
				group.CommonNames = append(group.CommonNames, entry.CommonName)
			}
		}
	}

	// Convert map to sorted slice
	var groups []DomainGroup
	for _, group := range domainMap {
		group.Count = len(group.CertIDs)
		groups = append(groups, *group)
	}

	// Sort by domain name
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Domain < groups[j].Domain
	})

	return groups
}

// contains checks if a string slice contains a value.
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func filterEntries(entries []CertEntry, clean, wildcard bool) []CertEntry {
	var filtered []CertEntry
	for _, e := range entries {
		// Filter matching identities
		var filteredIdentities []string
		for _, domain := range e.MatchingIdentities {
			isWild := strings.HasPrefix(domain, "*.")

			if wildcard && !isWild {
				continue
			}

			if clean {
				domain = strings.TrimPrefix(domain, "*.")
			}

			filteredIdentities = append(filteredIdentities, domain)
		}

		// Only include entry if it has matching identities after filtering
		if len(filteredIdentities) > 0 {
			e.MatchingIdentities = filteredIdentities
			// Update deprecated Domain field
			e.Domain = filteredIdentities[0]
			filtered = append(filtered, e)
		}
	}
	return filtered
}
