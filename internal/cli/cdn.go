package cli

import (
	"fmt"

	"github.com/j3ssie/metabigor/internal/cdn"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/projectdiscovery/cdncheck"
	"github.com/spf13/cobra"
)

const cdnLong = `Identify which IPs sit behind a CDN or WAF, and which do not.

Accepts bare IPs or URLs. Use --exclude to strip CDN and WAF addresses from a
list, leaving the candidate origin servers, or --only to keep just the
protected ones.`

var cdnCmd = &cobra.Command{
	Use:         "cdn [target...]",
	Short:       "Detect whether IPs are behind a CDN or WAF",
	Long:        cdnLong,
	GroupID:     groupEnrich,
	Annotations: map[string]string{"sample": "1.1.1.1"},
	Example: examples(
		"show the vendor and type for an IP", "metabigor cdn 1.1.1.1",
		"pass the target with a flag instead", "metabigor cdn -i 1.1.1.1 -f json",
		"classify several addresses at once", "metabigor cdn 1.1.1.1 8.8.8.8",
		"hunt for origin servers in a file", "metabigor cdn -I ips.txt --exclude -o origins.txt",
		"keep only the protected addresses", "metabigor cdn -I ips.txt --only",
		"or read the list from stdin", "cat ips.txt | metabigor cdn",
		"resolve domains, then classify them", "cat domains.txt | dnsx -silent -resp-only | metabigor cdn",
	),
	RunE: runCDN,
}

func init() {
	f := cdnCmd.Flags()
	f.BoolVar(&opt.CDN.Exclude, "exclude", false, "Drop CDN/WAF IPs, leaving candidate origins")
	f.BoolVar(&opt.CDN.Only, "only", false, "Keep only CDN/WAF IPs")

	cdnCmd.MarkFlagsMutuallyExclusive("exclude", "only")
	rootCmd.AddCommand(cdnCmd)
}

func runCDN(cmd *cobra.Command, _ []string) error {
	rc, err := setup(cmd)
	if err != nil {
		return err
	}
	defer rc.Close()

	client := cdncheck.New()
	if client == nil {
		return fmt.Errorf("initialize the cdncheck client")
	}

	output.Info("Checking %d target(s) for CDN/WAF", len(rc.Inputs))

	runner.RunParallel(rc.Inputs, opt.Concurrency, func(target string) {
		result, ok := cdn.Classify(client, target)
		if !ok {
			return
		}

		switch {
		case opt.CDN.Exclude && result.IsCDN:
			output.Verbose("Excluding %s (%s/%s)", result.IP, result.Vendor, result.Type)
			return
		case opt.CDN.Only && !result.IsCDN:
			output.Verbose("Skipping %s (not behind a CDN or WAF)", result.IP)
			return
		}

		rc.Writer.Write(result)
	})
	return nil
}
