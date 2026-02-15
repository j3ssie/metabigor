// Package related provides functionality to find domains related to a target via analytics and other sources.
package related

import (
	"regexp"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

var (
	uaPattern  = regexp.MustCompile(`UA-\d+-\d+`)
	gtmPattern = regexp.MustCompile(`GTM-[A-Z0-9]+`)
)

// AnalyticsRelated extracts UA/GTM IDs from a target site, then queries builtwith for related domains.
func AnalyticsRelated(client *retryablehttp.Client, domain string) []string {
	targetURL := "https://" + domain
	body, err := httpclient.Get(client, targetURL)
	if err != nil {
		output.Debug("analytics fetch error for %s: %v", domain, err)
		return nil
	}

	var ids []string
	ids = append(ids, uaPattern.FindAllString(body, -1)...)
	ids = append(ids, gtmPattern.FindAllString(body, -1)...)

	if len(ids) == 0 {
		output.Verbose("No analytics IDs found for %s", domain)
		return nil
	}

	output.Verbose("Found analytics IDs for %s: %v", domain, ids)

	seen := make(map[string]bool)
	var results []string
	for _, id := range ids {
		domains := BuiltWithRelated(client, id)
		for _, d := range domains {
			if !seen[d] {
				seen[d] = true
				results = append(results, d)
			}
		}
	}
	return results
}
