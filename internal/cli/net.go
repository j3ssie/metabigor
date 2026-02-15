package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/j3ssie/metabigor/internal/asndb"
	"github.com/j3ssie/metabigor/internal/countrydb"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/netdiscovery"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

func init() {
	netCmd.Flags().BoolVar(&opt.Net.ASN, "asn", false, "Force input as ASN")
	netCmd.Flags().BoolVar(&opt.Net.Org, "org", false, "Force input as organization name")
	netCmd.Flags().BoolVar(&opt.Net.IP, "ip", false, "Force input as IP address")
	netCmd.Flags().BoolVar(&opt.Net.Domain, "domain", false, "Force input as domain")
	netCmd.Flags().BoolVarP(&opt.Net.Dynamic, "dynamic", "d", false, "Use live online sources instead of local DB")
	netCmd.Flags().BoolVar(&opt.Net.Detail, "detail", false, "Show detailed info (type, description, country)")
	rootCmd.AddCommand(netCmd)
}

var netCmd = &cobra.Command{
	Use:   "net",
	Short: "Discover network ranges (CIDRs) for ASN, IP, domain, or organization",
	Long:  netLong,
	// Example is set in helptext.go init()
	Run: runNet,
}

func runNet(_ *cobra.Command, args []string) {
	output.SetupLogger(opt.Silent, opt.Debug, opt.NoColor)
	inputs := runner.ReadInputs(opt.Input, opt.InputFile, args)
	if len(inputs) == 0 {
		output.Error("No input provided")
		return
	}

	w, err := output.NewWriter(opt.Output, opt.JSONOutput)
	if err != nil {
		output.Error("%v", err)
		return
	}
	defer w.Close()

	// Determine forced input type
	forceType := netdiscovery.TypeUnknown
	switch {
	case opt.Net.ASN:
		forceType = netdiscovery.TypeASN
	case opt.Net.IP:
		forceType = netdiscovery.TypeIP
	case opt.Net.Domain:
		forceType = netdiscovery.TypeDomain
	case opt.Net.Org:
		forceType = netdiscovery.TypeOrg
	}

	if forceType != netdiscovery.TypeUnknown {
		output.Verbose("Input type forced to: %s", forceType)
	}

	if opt.Net.Dynamic {
		output.Info("Using dynamic (live) sources")
		client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

		// Check if detail mode is requested for domain/org lookups
		if opt.Net.Detail && (opt.Net.Domain || opt.Net.Org || forceType == netdiscovery.TypeDomain || forceType == netdiscovery.TypeOrg) {
			runner.RunParallel(inputs, opt.Concurrency, func(input string) {
				results := netdiscovery.DynamicLookupDetailed(client, input, opt.Timeout)
				for _, r := range results {
					if opt.JSONOutput {
						data, _ := json.Marshal(r)
						w.WriteString(string(data))
					} else {
						w.WriteString(r.Detailed())
					}
				}
			})
			return
		}

		// Default behavior: backward compatible (CIDRs only)
		runner.RunParallel(inputs, opt.Concurrency, func(input string) {
			results := netdiscovery.DynamicLookup(client, input, opt.Timeout)
			for _, r := range results {
				if opt.JSONOutput {
					data, _ := json.Marshal(map[string]string{"input": input, "cidr": r})
					w.WriteString(string(data))
				} else {
					w.WriteString(r)
				}
			}
		})
		return
	}

	// Static mode — use local ASN DB (auto-downloads on first run)
	output.Info("Using static (local DB) mode")
	db, err := asndb.EnsureLoaded()
	if err != nil {
		output.Error("%v", err)
		return
	}

	// Load country database for enrichment
	countryDB, err := countrydb.EnsureLoaded()
	if err != nil {
		output.Warn("Country database unavailable: %v (country info will be omitted)", err)
		countryDB = nil
	}

	runner.RunParallel(inputs, opt.Concurrency, func(input string) {
		results := netdiscovery.StaticLookup(db, input, forceType)
		for _, r := range results {
			// Enrich with country information if available
			enriched := enrichWithCountry(r, countryDB)

			if opt.JSONOutput {
				data, _ := json.Marshal(map[string]string{"input": input, "result": enriched})
				w.WriteString(string(data))
			} else {
				w.WriteString(enriched)
			}
		}
	})
}

// enrichWithCountry adds country information to CIDR results.
func enrichWithCountry(result string, countryDB *countrydb.DB) string {
	if countryDB == nil {
		return result
	}

	// Extract CIDR from result (handles formats like "1.0.0.0/24" or "AS13335 | 1.0.0.0/24 | Description")
	parts := strings.Split(result, "|")
	var cidr string

	if strings.Contains(result, "/") {
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.Contains(part, "/") {
				cidr = part
				break
			}
		}
		// If not found in parts, check if result itself is a CIDR
		if cidr == "" && strings.Contains(result, "/") {
			cidr = strings.TrimSpace(result)
		}
	}

	if cidr == "" {
		return result
	}

	// Lookup country
	countryRec := countryDB.LookupCIDR(cidr)
	if countryRec == nil {
		return result
	}

	// Add country info to result
	if len(parts) > 1 {
		// Format: AS | CIDR | Description | Country
		return fmt.Sprintf("%s | %s (%s)", result, countryRec.CountryCode, countryRec.CountryName)
	}
	// Format: CIDR | Country
	return fmt.Sprintf("%s | %s (%s)", result, countryRec.CountryCode, countryRec.CountryName)
}
