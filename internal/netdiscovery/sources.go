package netdiscovery

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

var cidrPattern = regexp.MustCompile(`(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/\d{1,2})`)

// BGPHEResult represents a parsed result from bgp.he.net search
type BGPHEResult struct {
	Result      string `json:"result"`      // AS number or CIDR
	Type        string `json:"type"`        // "ASN" or "Route"
	Description string `json:"description"` // Organization name
	Country     string `json:"country"`     // Country from flag title
}

// String returns just the result (AS/CIDR) for backward compatibility
func (r BGPHEResult) String() string {
	return r.Result
}

// Detailed returns formatted output with all fields
func (r BGPHEResult) Detailed() string {
	if r.Country != "" && r.Country != "Unknown" {
		return fmt.Sprintf("%s | %s | %s | %s", r.Result, r.Type, r.Description, r.Country)
	}
	return fmt.Sprintf("%s | %s | %s", r.Result, r.Type, r.Description)
}

// queryASNLookup queries asnlookup.com for ASN data (JSON API, no Chrome needed).
func queryASNLookup(client *retryablehttp.Client, asn string) []string {
	asn = strings.TrimPrefix(strings.ToUpper(asn), "AS")
	u := fmt.Sprintf("https://asnlookup.com/api/lookup?asn=AS%s", asn)

	body, err := httpclient.Get(client, u)
	if err != nil {
		output.Error("asnlookup.com failed: %v", err)
		return nil
	}

	results := cidrPattern.FindAllString(body, -1)
	output.Verbose("asnlookup.com returned %d CIDRs for AS%s", len(results), asn)
	return results
}

// queryIPInfo queries ipinfo.io — tries Chrome first, falls back to plain HTTP.
func queryIPInfo(client *retryablehttp.Client, ip string, timeoutSec int) []string {
	u := fmt.Sprintf("https://ipinfo.io/%s", ip)

	// Try Chrome first for JS-rendered data
	body, err := httpclient.ChromeGet(u, timeoutSec)
	if err != nil {
		output.Verbose("ipinfo.io Chrome failed: %v — falling back to HTTP", err)
		body, err = httpclient.Get(client, u+"/json")
		if err != nil {
			output.Error("ipinfo.io HTTP fallback also failed: %v", err)
			return nil
		}
	}

	results := cidrPattern.FindAllString(body, -1)
	output.Verbose("ipinfo.io returned %d CIDRs for %s", len(results), ip)
	return results
}

// queryBGPHE queries bgp.he.net — tries Chrome first, falls back to plain HTTP.
func queryBGPHE(client *retryablehttp.Client, query string, timeoutSec int) []BGPHEResult {
	u := fmt.Sprintf("https://bgp.he.net/search?search%%5Bsearch%%5D=%s&commit=Search", query)

	body, err := httpclient.ChromeGet(u, timeoutSec)
	if err != nil {
		output.Verbose("bgp.he.net Chrome failed: %v — falling back to HTTP", err)
		body, err = httpclient.Get(client, u)
		if err != nil {
			output.Error("bgp.he.net HTTP fallback also failed: %v", err)
			return nil
		}
	}

	// Parse HTML table
	results := parseBGPHETable(body)

	// Fallback to regex if parsing fails
	if len(results) == 0 {
		output.Verbose("bgp.he.net HTML parsing returned 0 results, falling back to regex")
		results = parseBGPHERegex(body)
	}

	output.Verbose("bgp.he.net returned %d results for %q", len(results), query)
	return results
}

// parseBGPHETable parses the HTML table from bgp.he.net search results
func parseBGPHETable(htmlBody string) []BGPHEResult {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlBody))
	if err != nil {
		output.Debug("bgp.he.net HTML parse error: %v", err)
		return nil
	}

	var results []BGPHEResult

	// Find table rows in the search results table
	doc.Find("table tbody tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() < 3 {
			return // Skip header or incomplete rows
		}

		result := BGPHEResult{
			Result:      strings.TrimSpace(tds.Eq(0).Text()),
			Type:        strings.TrimSpace(tds.Eq(1).Text()),
			Description: extractDescription(tds.Eq(2)),
			Country:     extractCountry(tds.Eq(2)),
		}

		// Decode HTML entities (e.g., &#39; -> ')
		result.Description = decodeHTMLEntities(result.Description)

		if result.Result != "" {
			results = append(results, result)
		}
	})

	return results
}

// extractDescription extracts clean description text from the table cell
func extractDescription(td *goquery.Selection) string {
	// Clone the selection and remove the flag div to get clean description
	clone := td.Clone()
	clone.Find("div.flag").Remove()
	text := strings.TrimSpace(clone.Text())
	return text
}

// extractCountry extracts country information from the flag image
func extractCountry(td *goquery.Selection) string {
	country, exists := td.Find("img").Attr("title")
	if !exists || country == "" {
		return "Unknown"
	}
	return country
}

// decodeHTMLEntities decodes common HTML entities
func decodeHTMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	return s
}

// parseBGPHERegex provides fallback regex extraction if HTML parsing fails
func parseBGPHERegex(htmlBody string) []BGPHEResult {
	var results []BGPHEResult
	for _, cidr := range cidrPattern.FindAllString(htmlBody, -1) {
		results = append(results, BGPHEResult{
			Result: cidr,
			Type:   "Route",
		})
	}
	return results
}
