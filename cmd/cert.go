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
	var certCmd = &cobra.Command{
		Use:   "cert",
		Short: "Certificates search",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Tool but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runCert,
	}

	certCmd.Flags().StringSliceP("query", "q", []string{}, "Query to search (Multiple -q flags are accepted)")
	certCmd.Flags().BoolVarP(&options.Cert.Clean, "clean", "C", false, "Auto clean the result")
	certCmd.Flags().BoolVarP(&options.Cert.OnlyWildCard, "wildcard", "W", false, "Only get wildcard domain")
	RootCmd.AddCommand(certCmd)
}

func runCert(cmd *cobra.Command, _ []string) error {
	queries, _ := cmd.Flags().GetStringSlice("query")

	// auto increase timeout
	options.Timeout = 120
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
				searchResult := runCertSearch(job, options)
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

func runCertSearch(input string, options core.Options) []string {
	var data []string
	core.BannerF(fmt.Sprintf("Search on %v for: ", "crt.sh"), input)
	//options.Search.Query = input

	result := modules.CrtSHOrg(input, options)
	data = append(data, result...)

	return data
}
