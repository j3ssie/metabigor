package related

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
)

var domainPattern = regexp.MustCompile(`([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}`)

// BuiltWithRelated queries builtwith.com for domains sharing the same tracking ID.
func BuiltWithRelated(client *retryablehttp.Client, trackingID string) []string {
	targetURL := fmt.Sprintf("https://builtwith.com/relationships/tag/%s", trackingID)
	body, err := httpclient.Get(client, targetURL)
	if err != nil {
		output.Debug("builtwith error: %v", err)
		return nil
	}

	seen := make(map[string]bool)
	var results []string
	for _, match := range domainPattern.FindAllString(body, -1) {
		d := strings.ToLower(match)
		// Filter out common non-domain matches
		if !seen[d] && !isCommonNonDomain(d) {
			seen[d] = true
			results = append(results, d)
		}
	}

	output.Verbose("builtwith: %d domains for %s", len(results), trackingID)
	return results
}

func isCommonNonDomain(s string) bool {
	skip := []string{
		"builtwith.com", "googleapis.com", "gstatic.com",
		"google.com", "w3.org", "schema.org",
	}
	for _, k := range skip {
		if s == k || strings.HasSuffix(s, "."+k) {
			return true
		}
	}
	return false
}
