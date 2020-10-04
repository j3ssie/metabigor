package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	"github.com/spf13/cobra"
)

func init() {
	var cveCmd = &cobra.Command{
		Use:   "cve",
		Short: "CVE or Advisory Search",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Tool but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runCVE,
	}

	cveCmd.Flags().StringP("source", "s", "all", "Search Engine Select")
	cveCmd.Flags().StringSliceP("query", "q", []string{}, "Query to search (Multiple -q flags are accepted)")
	RootCmd.AddCommand(cveCmd)
}

func runCVE(cmd *cobra.Command, _ []string) error {
	options.Search.Source, _ = cmd.Flags().GetString("source")
	options.Search.Source = strings.ToLower(options.Search.Source)
	queries, _ := cmd.Flags().GetStringSlice("query")

	var inputs []string
	if options.Input != "-" && options.Input != "" {
		if strings.Contains(options.Input, "\n") {
			inputs = strings.Split(options.Input, "\n")
		} else {
			inputs = append(inputs, options.Input)
		}
	}
	if len(queries) > 0 {
		inputs = append(inputs, queries...)
	}
	if len(inputs) == 0 {
		core.ErrorF("No input found")
		os.Exit(1)
	}


	var wg sync.WaitGroup
	jobs := make(chan string)

	for i := 0; i < options.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// do real stuff here
			for job := range jobs {
				searchResult := runCVESingle(job, options)
				StoreData(searchResult, options)
			}
		}()
	}

	for _, input := range inputs {
		jobs <- input
	}

	close(jobs)
	wg.Wait()

	if !core.FileExists(options.Output) {
		core.ErrorF("No data found")
	}
	core.DebugF("Unique Output: %v", options.Output)
	core.Unique(options.Output)
	return nil
}

func runCVESingle(input string, options core.Options) []string {
	var data []string
	core.BannerF(fmt.Sprintf("Search on %v for: ", options.Search.Source), input)
	if options.Search.Source == "all" {
		options.Search.Source = "vulners"
	}
	options.Search.Query = input

	// select source
	if strings.Contains(options.Search.Source, "vulner") {
		data = append(data, modules.Vulners(options)...)
	}
	return data
}
