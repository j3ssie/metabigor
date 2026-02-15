package cli

import (
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/related"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

func init() {
	relatedCmd.Flags().StringVarP(&opt.Related.Source, "source", "s", "all", "Source: crt, whois, ua, gtm, all")
	rootCmd.AddCommand(relatedCmd)
}

var relatedCmd = &cobra.Command{
	Use:   "related",
	Short: "Find related domains via WHOIS, crt.sh, analytics, builtwith",
	Long:  relatedLong,
	// Example is set in helptext.go init()
	Run: runRelated,
}

func runRelated(_ *cobra.Command, args []string) {
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

	src := opt.Related.Source
	output.Info("Finding related domains for %d input(s) (source: %s)", len(inputs), src)
	client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

	runner.RunParallel(inputs, opt.Concurrency, func(domain string) {
		var results []string

		switch src {
		case "crt":
			output.Verbose("Querying crt.sh for %s", domain)
			results = related.CRTRelated(client, domain)
		case "whois":
			output.Verbose("Querying viewdns.info reverse WHOIS for %s", domain)
			results = related.WhoisRelated(client, domain)
		case "ua", "gtm":
			output.Verbose("Extracting analytics IDs from %s", domain)
			results = related.AnalyticsRelated(client, domain)
		case "all":
			seen := make(map[string]bool)
			output.Verbose("Querying crt.sh for %s", domain)
			for _, r := range related.CRTRelated(client, domain) {
				if !seen[r] {
					seen[r] = true
					results = append(results, r)
				}
			}
			output.Verbose("Querying viewdns.info for %s", domain)
			for _, r := range related.WhoisRelated(client, domain) {
				if !seen[r] {
					seen[r] = true
					results = append(results, r)
				}
			}
			output.Verbose("Extracting analytics IDs from %s", domain)
			for _, r := range related.AnalyticsRelated(client, domain) {
				if !seen[r] {
					seen[r] = true
					results = append(results, r)
				}
			}
		default:
			output.Error("Unknown source: %s (use crt, whois, ua, gtm, or all)", src)
			return
		}

		output.Verbose("Found %d related domain(s) for %s", len(results), domain)
		for _, r := range results {
			w.WriteString(r)
		}
	})
}
