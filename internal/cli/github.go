package cli

import (
	"fmt"
	"time"

	"github.com/j3ssie/metabigor/internal/gitsearch"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

// pageDelay paces grep.app pagination so a long search is not throttled.
const pageDelay = 5 * time.Second

const githubLong = `Search public GitHub code through grep.app to surface leaked hostnames,
credentials, and API keys.

Prints one repo and path per match. Add --subs to extract only the subdomains
mentioned in the matches, or --detail to see the matching code.

Requires Chrome or Chromium: grep.app answers plain HTTP clients with a bot
challenge, so queries are driven through headless Chrome. Searches run one at a
time regardless of --concurrency, to stay within grep.app's rate limit.`

var githubCmd = &cobra.Command{
	Use:         "github [query...]",
	Short:       "Search public GitHub code for leaks (via grep.app)",
	Long:        githubLong,
	GroupID:     groupRecon,
	Annotations: map[string]string{"sample": "example.com"},
	Example: examples(
		"find code mentioning a domain", "metabigor github hackerone.com",
		"pass the query with a flag instead", "metabigor github -i hackerone.com",
		"pull out the subdomains in those matches", "metabigor github tesla.com --subs",
		"show the matching code", "metabigor github -i \"api_key=\" --detail",
		"hunt for AWS keys, first 3 pages", "metabigor github AKIA --pages 3 --detail",
		"run a list of search terms from a file", "metabigor github -I keywords.txt -f json -o hits.json",
		"use -i when the query starts with a dash", "metabigor github -i \"--endpoint-url\"",
	),
	RunE: runGithub,
}

func init() {
	f := githubCmd.Flags()
	f.IntVar(&opt.Github.Pages, "pages", 0, "Maximum result pages to fetch (0 fetches all)")
	f.BoolVar(&opt.Github.Subs, "subs", false, "Print only the subdomains found in matches")
	f.BoolVarP(&opt.Github.Detail, "detail", "d", false, "Show the matching code snippet")

	githubCmd.MarkFlagsMutuallyExclusive("subs", "detail")
	rootCmd.AddCommand(githubCmd)
}

func runGithub(cmd *cobra.Command, _ []string) error {
	rc, err := setup(cmd)
	if err != nil {
		return err
	}
	defer rc.Close()

	output.Info("Searching grep.app for %d query(ies)", len(rc.Inputs))

	session, err := httpclient.NewChromeSession(opt.Timeout, opt.Proxy)
	if err != nil {
		return fmt.Errorf("grep.app needs headless Chrome to clear its bot challenge: %w", err)
	}
	defer session.Close()

	// grep.app throttles aggressively, so queries run sequentially even when
	// --concurrency is higher.
	runner.RunParallel(rc.Inputs, 1, func(query string) {
		hits, err := gitsearch.SearchAll(session, query, pageDelay, opt.Github.Pages)
		if err != nil {
			output.Error("grep.app search for %q failed: %v", query, err)
			// Fall through: any hits gathered before the failure are still useful.
		}
		if len(hits) == 0 {
			output.Verbose("No matches for %q", query)
			return
		}
		output.Verbose("grep.app returned %d hit(s) for %q", len(hits), query)

		if opt.Github.Subs {
			subs := gitsearch.ExtractSubdomains(hits, query)
			if len(subs) == 0 {
				output.Verbose("No subdomains of %q appear in the matches", query)
				return
			}
			output.Good("Found %d subdomain(s) of %q", len(subs), query)
			for _, s := range subs {
				rc.Writer.Write(gitsearch.Subdomain{Query: query, Domain: s})
			}
			return
		}

		for _, h := range hits {
			h.Detail = opt.Github.Detail
			rc.Writer.Write(h)
		}
	})
	return nil
}
