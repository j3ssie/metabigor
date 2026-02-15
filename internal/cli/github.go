package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/j3ssie/metabigor/internal/gitsearch"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

func init() {
	githubCmd.Flags().BoolVar(&opt.Github.Detail, "detail", false, "Show formatted code snippets with repo, path, and content")
	githubCmd.Flags().IntVar(&opt.Github.Page, "page", 0, "Maximum number of pages to fetch (0 = unlimited)")
	rootCmd.AddCommand(githubCmd)
}

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "Search code on grep.app (GitHub code search)",
	Long:  githubLong,
	// Example is set in helptext.go init()
	Run: runGithub,
}

func runGithub(_ *cobra.Command, args []string) {
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

	output.Info("Searching grep.app for %d query(ies)", len(inputs))
	client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

	// Set up glamour renderer for --detail mode
	var renderer *glamour.TermRenderer
	if opt.Github.Detail && !opt.JSONOutput {
		renderer, err = glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(120))
		if err != nil {
			output.Debug("glamour init error: %v, falling back to plain", err)
		}
	}

	runner.RunParallel(inputs, 1, func(query string) {
		maxPages := opt.Github.Page
		if maxPages == 0 {
			output.Verbose("Querying grep.app for %q (auto-paginating with 5s delay, unlimited pages)", query)
		} else {
			output.Verbose("Querying grep.app for %q (max %d pages, 5s delay)", query, maxPages)
		}
		hits := gitsearch.SearchAll(client, query, 5*time.Second, maxPages)

		if opt.JSONOutput {
			for _, h := range hits {
				// Output raw JSON per hit with cleaned snippet
				obj := map[string]any{
					"owner_id":      h.OwnerID,
					"repo":          h.Repo,
					"branch":        h.Branch,
					"path":          h.Path,
					"content":       map[string]string{"snippet": h.CleanSnippet()},
					"total_matches": h.TotalMatches,
				}
				data, _ := json.Marshal(obj)
				w.WriteString(string(data))
			}
			return
		}

		if opt.Github.Detail {
			for _, h := range hits {
				// Use new ParseSnippet() instead of CleanSnippet()
				snippet := h.ParseSnippet()

				// Create header with metadata on one line
				header := fmt.Sprintf("Repo: %s | Path: %s | Branch: %s | Matches: %s",
					h.Repo, h.Path, h.Branch, h.TotalMatches)

				// Format as markdown with code block for glamour
				md := fmt.Sprintf("### %s\n---\n```\n%s\n```\n", header, snippet)

				if renderer != nil {
					rendered, err := renderer.Render(md)
					if err == nil {
						fmt.Print(rendered)
						fmt.Println("---") // Separator between results
						continue
					}
				}

				// Fallback: plain text with clear formatting
				fmt.Printf("%s\n---\n%s\n---\n\n", header, snippet)
			}
			return
		}

		// Default: extract subdomains matching the input domain
		subdomains := gitsearch.ExtractSubdomains(hits, query)
		if len(subdomains) > 0 {
			output.Good("Found %d subdomain(s) matching %q", len(subdomains), query)
			for _, s := range subdomains {
				w.WriteString(s)
			}
		} else {
			// If no subdomain matches, show repo|path summary
			output.Verbose("No subdomain matches for %q, showing repo|path", query)
			for _, h := range hits {
				w.WriteString(fmt.Sprintf("%s | %s", h.Repo, h.Path))
			}
		}
	})
}

