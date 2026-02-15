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

// CRTRelated finds related domains via crt.sh certificate transparency.
func CRTRelated(client *retryablehttp.Client, domain string) []string {
	targetURL := fmt.Sprintf("https://crt.sh/?q=%s", url.QueryEscape(domain))
	body, err := httpclient.Get(client, targetURL)
	if err != nil {
		output.Debug("crt.sh related error: %v", err)
		return nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var results []string

	doc.Find("table tr td").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		text = strings.ToLower(text)
		text = strings.TrimPrefix(text, "*.")
		if text != "" && strings.Contains(text, ".") && !seen[text] {
			seen[text] = true
			results = append(results, text)
		}
	})

	output.Verbose("crt.sh related: %d domains for %s", len(results), domain)
	return results
}
