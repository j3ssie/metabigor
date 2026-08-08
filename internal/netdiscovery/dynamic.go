// Package netdiscovery provides ASN/CIDR discovery from online sources and local databases.
package netdiscovery

import (
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/output"
)

// LiveLookup queries live online sources for ASN/CIDR data. Pass TypeUnknown to
// auto-detect the input type.
func LiveLookup(client *retryablehttp.Client, input string, forceType InputType, timeoutSec int) []Result {
	if forceType == TypeUnknown {
		forceType = DetectType(input)
	}
	output.Info("Live lookup for %q (as %s)", input, forceType)

	var results []Result
	seen := make(map[string]bool)

	add := func(source string, batch []Result) {
		added := 0
		for _, r := range batch {
			key := r.Value()
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			r.Input = input
			r.Source = source
			results = append(results, r)
			added++
		}
		output.Verbose("%s returned %d new result(s) of %d", source, added, len(batch))
	}

	switch forceType {
	case TypeASN:
		output.Verbose("Querying asnlookup.com for %s ...", input)
		add("asnlookup.com", cidrResults(queryASNLookup(client, input)))

		// asnlookup.com sits behind a bot challenge that plain HTTP clients
		// cannot clear, so fall back to bgp.he.net, which Chrome can reach.
		if len(results) == 0 {
			output.Verbose("asnlookup.com returned nothing, falling back to bgp.he.net ...")
			add("bgp.he.net", bgpResults(queryBGPHE(client, input, timeoutSec)))
		}

	case TypeIP, TypeCIDR:
		output.Verbose("Querying ipinfo.io for %s ...", input)
		add("ipinfo.io", cidrResults(queryIPInfo(client, input, timeoutSec)))

	case TypeDomain, TypeOrg:
		output.Verbose("Querying bgp.he.net for %q ...", input)
		add("bgp.he.net", bgpResults(queryBGPHE(client, input, timeoutSec)))
	}

	if len(results) == 0 {
		output.Warn("Live lookup for %q returned no results", input)
	} else {
		output.Good("Live lookup for %q: %d result(s)", input, len(results))
	}
	return results
}

// cidrResults wraps bare CIDR strings from the simpler sources.
func cidrResults(cidrs []string) []Result {
	results := make([]Result, 0, len(cidrs))
	for _, c := range cidrs {
		if c = strings.TrimSpace(c); c != "" {
			results = append(results, Result{CIDR: c})
		}
	}
	return results
}

// bgpResults maps bgp.he.net rows, which mix ASNs and routes, onto Result.
// Rows are already filtered by validBGPRows; anything else is skipped rather
// than guessed at.
func bgpResults(rows []BGPHEResult) []Result {
	results := make([]Result, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row.Result)
		if !isNetworkValue(value) {
			continue
		}

		r := Result{Org: row.Description}
		if asnPattern.MatchString(value) {
			r.ASN = strings.ToUpper(value)
		} else {
			r.CIDR = value
		}
		if row.Country != "" && row.Country != "Unknown" {
			r.Country = row.Country
		}
		results = append(results, r)
	}
	return results
}
