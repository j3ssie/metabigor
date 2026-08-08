package cli

import (
	"github.com/j3ssie/metabigor/internal/cert"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

const certLong = `Search Certificate Transparency logs on crt.sh for a domain or organization.

Prints one domain per line, so results pipe straight into other tools. Add
--detail to see the certificate IDs, issuers, and validity window behind each
domain.`

var certCmd = &cobra.Command{
	Use:         "cert [target...]",
	Short:       "Find subdomains in Certificate Transparency logs (crt.sh)",
	Long:        certLong,
	GroupID:     groupRecon,
	Annotations: map[string]string{"sample": "hackerone.com"},
	Example: examples(
		"list every domain seen in a certificate", "metabigor cert hackerone.com",
		"pass the target with a flag instead", "metabigor cert -i hackerone.com",
		"search by organization name", "metabigor cert \"HackerOne Inc\"",
		"drop the *. prefix from wildcard entries", "metabigor cert -i example.com --clean",
		"show certificate IDs, issuers, and dates", "metabigor cert example.com --detail",
		"run a batch from a file", "metabigor cert -I domains.txt -o subdomains.txt",
		"or read the batch from stdin", "cat domains.txt | metabigor cert",
		"resolve the results with another tool", "metabigor cert tesla.com | dnsx -silent",
	),
	RunE: runCert,
}

func init() {
	f := certCmd.Flags()
	f.BoolVar(&opt.Cert.Clean, "clean", false, "Strip the *. prefix from wildcard domains")
	f.BoolVar(&opt.Cert.Wildcard, "wildcard", false, "Keep only wildcard (*.) entries")
	f.BoolVarP(&opt.Cert.Detail, "detail", "d", false, "Show certificate IDs, issuers, and validity dates")
	rootCmd.AddCommand(certCmd)
}

func runCert(cmd *cobra.Command, _ []string) error {
	rc, err := setup(cmd)
	if err != nil {
		return err
	}
	defer rc.Close()

	output.Info("Searching crt.sh for %d target(s)", len(rc.Inputs))
	client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

	runner.RunParallel(rc.Inputs, opt.Concurrency, func(target string) {
		entries := cert.SearchCRT(client, target, opt.Cert.Clean, opt.Cert.Wildcard)
		groups := cert.GroupByDomain(entries)
		output.Verbose("crt.sh: %d certificate(s) covering %d domain(s) for %q", len(entries), len(groups), target)

		for _, g := range groups {
			g.Detail = opt.Cert.Detail
			rc.Writer.Write(g)
		}
	})
	return nil
}
