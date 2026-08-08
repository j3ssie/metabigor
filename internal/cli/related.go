package cli

import (
	"strings"

	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/related"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

const relatedLong = `Discover domains that belong to the same owner as the target.

Sources:
  crt        Certificate Transparency logs (crt.sh)
  whois      Reverse WHOIS lookups (viewdns.info)
  analytics  Shared Google Analytics and Tag Manager IDs (builtwith.com)

All three run by default. Pass --sources to pick a subset; results are
deduplicated across sources and tagged with the one that found them first.`

var relatedCmd = &cobra.Command{
	Use:         "related [domain...]",
	Short:       "Find other domains owned by the same target",
	Long:        relatedLong,
	GroupID:     groupRecon,
	Annotations: map[string]string{"sample": "hackerone.com"},
	Example: examples(
		"query every source", "metabigor related hackerone.com",
		"pass the target with a flag instead", "metabigor related -i hackerone.com",
		"pick one source", "metabigor related tesla.com --sources crt",
		"combine a few sources", "metabigor related -i tesla.com --sources crt,whois",
		"run a batch from a file", "metabigor related -I domains.txt -o related.txt",
		"or read the batch from stdin", "cat domains.txt | metabigor related",
		"emit bare domains for the next tool", "metabigor related tesla.com -f flat | metabigor cert",
	),
	RunE: runRelated,
}

func init() {
	f := relatedCmd.Flags()
	f.StringSliceVarP(&opt.Related.Sources, "sources", "s", nil,
		"Sources to query: "+strings.Join(related.SourceNames(), ", ")+", or all (default all)")
	rootCmd.AddCommand(relatedCmd)
}

func runRelated(cmd *cobra.Command, _ []string) error {
	// Validate the sources before opening the writer so a typo fails
	// immediately rather than once per target.
	sources, err := related.ParseSources(opt.Related.Sources)
	if err != nil {
		return err
	}

	rc, err := setup(cmd)
	if err != nil {
		return err
	}
	defer rc.Close()

	output.Info("Finding related domains for %d target(s) via %d source(s)", len(rc.Inputs), len(sources))
	client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

	runner.RunParallel(rc.Inputs, opt.Concurrency, func(domain string) {
		results := related.Find(client, domain, sources)
		output.Verbose("Found %d related domain(s) for %s", len(results), domain)
		for _, r := range results {
			rc.Writer.Write(r)
		}
	})
	return nil
}
