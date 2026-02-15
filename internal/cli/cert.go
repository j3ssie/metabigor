package cli

import (
	"fmt"
	"strings"

	"github.com/j3ssie/metabigor/internal/cert"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

func init() {
	certCmd.Flags().BoolVar(&opt.Cert.Clean, "clean", false, "Strip wildcard prefix (*.) from domains")
	certCmd.Flags().BoolVar(&opt.Cert.Wildcard, "wildcard", false, "Show only wildcard entries")
	certCmd.Flags().BoolVar(&opt.Cert.Simple, "simple", false, "Output only domain names (simple list, old behavior)")
	rootCmd.AddCommand(certCmd)
}

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Search certificate transparency logs (crt.sh)",
	Long:  certLong,
	// Example is set in helptext.go init()
	Run: runCert,
}

func runCert(_ *cobra.Command, args []string) {
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

	output.Info("Searching crt.sh for %d query(ies)", len(inputs))
	client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

	runner.RunParallel(inputs, opt.Concurrency, func(input string) {
		output.Verbose("Querying crt.sh for %q (clean=%v, wildcard=%v, simple=%v)", input, opt.Cert.Clean, opt.Cert.Wildcard, opt.Cert.Simple)
		entries := cert.SearchCRT(client, input, opt.Cert.Clean, opt.Cert.Wildcard)
		output.Verbose("crt.sh returned %d certificate entries for %q", len(entries), input)

		// Simple mode: just domain names (backward compatible)
		if opt.Cert.Simple {
			for _, e := range entries {
				for _, domain := range e.MatchingIdentities {
					w.WriteString(domain)
				}
			}
			return
		}

		// DEFAULT: Grouped view with full details
		groups := cert.GroupByDomain(entries)
		output.Verbose("Grouped into %d unique domains for %q", len(groups), input)

		for _, g := range groups {
			if opt.JSONOutput {
				w.WriteJSON(g)
			} else {
				// Table-style grouped output (DEFAULT)
				w.WriteString(fmt.Sprintf("Domain: %s", g.Domain))
				w.WriteString(fmt.Sprintf("  Cert IDs (%d): %s", g.Count, strings.Join(g.CertIDs, ", ")))
				if g.FirstSeen != "" {
					w.WriteString(fmt.Sprintf("  Not Before: %s", g.FirstSeen))
				}
				if g.LastExpires != "" {
					w.WriteString(fmt.Sprintf("  Not After: %s", g.LastExpires))
				}
				if len(g.Issuers) > 0 {
					w.WriteString(fmt.Sprintf("  Issuers: %s", strings.Join(g.Issuers, ", ")))
				}
				if len(g.CommonNames) > 0 && len(g.CommonNames) <= 3 {
					// Only show common names if there are a few
					w.WriteString(fmt.Sprintf("  Common Names: %s", strings.Join(g.CommonNames, ", ")))
				}
				w.WriteString("") // blank line between domains
			}
		}
	})
}
