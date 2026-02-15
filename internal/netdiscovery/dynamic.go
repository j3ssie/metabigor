// Package netdiscovery provides ASN/CIDR discovery from online sources and local databases.
package netdiscovery

import (
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/output"
)

// DynamicLookup queries live online sources for ASN/CIDR data.
func DynamicLookup(client *retryablehttp.Client, input string, timeoutSec int) []string {
	inputType := DetectType(input)
	var allResults []string
	seen := make(map[string]bool)

	add := func(source string, results []string) {
		added := 0
		for _, r := range results {
			r = strings.TrimSpace(r)
			if r != "" && !seen[r] {
				seen[r] = true
				allResults = append(allResults, r)
				added++
			}
		}
		output.Verbose("Source %s returned %d new results (%d total from source)", source, added, len(results))
	}

	output.Info("Dynamic lookup for %q (detected type: %s)", input, inputType)

	switch inputType {
	case TypeASN:
		output.Verbose("Querying asnlookup.com for %s ...", input)
		add("asnlookup.com", queryASNLookup(client, input))

	case TypeIP:
		output.Verbose("Querying ipinfo.io for %s ...", input)
		add("ipinfo.io", queryIPInfo(client, input, timeoutSec))

	case TypeDomain:
		output.Verbose("Querying bgp.he.net for %s ...", input)
		bgpResults := queryBGPHE(client, input, timeoutSec)
		add("bgp.he.net", extractResultStrings(bgpResults))

	case TypeOrg:
		output.Verbose("Querying bgp.he.net for %q ...", input)
		bgpResults := queryBGPHE(client, input, timeoutSec)
		add("bgp.he.net", extractResultStrings(bgpResults))
	}

	if len(allResults) == 0 {
		output.Warn("Dynamic lookup for %q returned no results", input)
	} else {
		output.Good("Dynamic lookup for %q: %d total results", input, len(allResults))
	}

	return allResults
}

// extractResultStrings converts BGPHEResult to strings for backward compatibility
func extractResultStrings(results []BGPHEResult) []string {
	strs := make([]string, len(results))
	for i, r := range results {
		strs[i] = r.Result
	}
	return strs
}

// DynamicLookupDetailed returns structured BGP results with all metadata
func DynamicLookupDetailed(client *retryablehttp.Client, input string, timeoutSec int) []BGPHEResult {
	inputType := DetectType(input)

	output.Info("Dynamic detailed lookup for %q (detected type: %s)", input, inputType)

	switch inputType {
	case TypeDomain, TypeOrg:
		results := queryBGPHE(client, input, timeoutSec)
		if len(results) == 0 {
			output.Warn("Dynamic lookup for %q returned no results", input)
		} else {
			output.Good("Dynamic lookup for %q: %d results", input, len(results))
		}
		return results
	default:
		output.Warn("Detailed lookup only supported for domain/org inputs")
		return nil
	}
}
