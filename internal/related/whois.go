package related

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

// WhoisRelated finds related domains via viewdns.info reverse WHOIS.
func WhoisRelated(client *retryablehttp.Client, domain string) []string {
	targetURL := fmt.Sprintf("https://viewdns.info/reversewhois/?q=%s", url.QueryEscape(domain))
	body, err := httpclient.Get(client, targetURL)
	if err != nil {
		output.Debug("viewdns.info error: %v", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var results []string

	doc.Find("table#null tr td:first-child").Each(func(_ int, s *goquery.Selection) {
		d := strings.TrimSpace(strings.ToLower(s.Text()))
		if d != "" && strings.Contains(d, ".") && !seen[d] && d != "domain name" {
			seen[d] = true
			results = append(results, d)
		}
	})

	output.Verbose("viewdns.info: %d domains for %s", len(results), domain)
	return results
}
