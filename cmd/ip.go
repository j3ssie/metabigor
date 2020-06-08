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
	var ipCmd = &cobra.Command{
		Use:   "ip",
		Short: "IP OSINT Search",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Tool but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runIP,
	}

	ipCmd.Flags().StringP("source", "s", "all", "Search Engine Select")
	ipCmd.Flags().StringSliceP("query", "q", []string{}, "Query to search (Multiple -q flags are accepted)")
	RootCmd.AddCommand(ipCmd)
}

func runIP(cmd *cobra.Command, _ []string) error {
	options.Search.Source, _ = cmd.Flags().GetString("source")
	options.Search.Source = strings.ToLower(options.Search.Source)
	options.Search.More, _ = cmd.Flags().GetBool("brute")
	options.Search.Optimize, _ = cmd.Flags().GetBool("optimize")
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

	if options.Search.More {
		inputs = addMoreQuery(inputs, options)
	}

	var wg sync.WaitGroup
	jobs := make(chan string)

	for i := 0; i < options.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// do real stuff here
			for job := range jobs {
				searchResult := runIPSingle(job, options)
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

func runIPSingle(input string, options core.Options) []string {
	var data []string
	core.BannerF(fmt.Sprintf("Search on %v for: ", options.Search.Source), input)
	if options.Search.Source == "all" {
		options.Search.Source = "ony,shodan"
	}
	options.Search.Query = input

	// select source
	if strings.Contains(options.Search.Source, "ony") {
		data = append(data, modules.Onyphe(options.Search.Query, options)...)
	}

	if strings.Contains(options.Search.Source, "sho") {
		data = append(data, modules.Shodan(options.Search.Query, options)...)
	}

	return data
}
