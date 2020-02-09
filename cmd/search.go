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

var searchCmd *cobra.Command

func init() {
	// byeCmd represents the bye command
	var searchCmd = &cobra.Command{
		Use:   "search",
		Short: "Do Search on popular search engine",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Framework but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runSearch,
	}

	searchCmd.Flags().StringP("source", "s", "fofa", "Search Engine")
	searchCmd.Flags().StringSliceP("query", "q", []string{}, "Query to search")
	searchCmd.Flags().BoolP("brute", "b", false, "Enable Brute Force")
	searchCmd.Flags().BoolP("optimize", "x", false, "Enable Optimize Query")
	RootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
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
				searchResult := runSearchSingle(job, options)
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

func runSearchSingle(input string, options core.Options) []string {
	var data []string
	core.BannerF(fmt.Sprintf("Search on %v for: ", options.Search.Source), input)
	options.Search.Query = input

	switch options.Search.Source {
	case "fofa":
		data = append(data, modules.FoFaSearch(options)...)
		break
	}
	return data
}

// add more query by add the country code with original query
func addMoreQuery(inputs []string, options core.Options) []string {
	var moreQueries []string
	ContriesCode := []string{"AF̵", "AL", "DZ", "AS", "AD", "AO", "AI", "AQ", "AG", "AR", "AM", "AW", "AU", "AT", "AZ", "BS", "BH", "BD", "BB", "BY", "BE", "BZ", "BJ", "BM", "BT", "BO", "BA", "BW", "BV", "BR", "IO", "BN", "BG", "BF", "BI", "KH", "CM", "CA", "CV", "KY", "CF", "TD", "CL", "CN", "CX", "CC", "CO", "KM", "CG", "CD", "CK", "CR", "CI", "HR", "CU", "CY", "CZ", "DK", "DJ", "DM", "DO", "EC", "EG", "EH", "SV", "GQ", "ER", "EE", "ET", "FK", "FO", "FJ", "FI", "FR", "GF", "PF", "TF", "GA", "GM", "GE", "DE", "GH", "GI", "GR", "GL", "GD", "GP", "GU", "GT", "GN", "GW", "GY", "HT", "HM", "HN", "HK", "HU", "IS", "IN", "ID", "IR", "IQ", "IE", "IL", "IT", "JM", "JP", "JO", "KZ", "KE", "KI", "KP", "KR", "KW", "KG", "LA", "LV", "LB", "LS", "LR", "LY", "LI", "LT", "LU", "MO", "MK", "MG", "MW", "MY", "MV", "ML", "MT", "MH", "MQ", "MR", "MU", "YT", "MX", "FM", "MD", "MC", "MN", "MS", "MA", "MZ", "MM", "NA", "NR", "NP", "NL", "AN", "NC", "NZ", "NI", "NE", "NG", "NU", "NF", "MP", "NO", "OM", "PK", "PW", "PS", "PA", "PG", "PY", "PE", "PH", "PN", "PL", "PT", "PR", "QA", "RE", "RO", "RU", "RW", "SH", "KN", "LC", "PM", "VC", "WS", "SM", "ST", "SA", "SN", "CS", "SC", "SL", "SG", "SK", "SI", "SB", "SO", "ZA", "GS", "ES", "LK", "SD", "SR", "SJ", "SZ", "SE", "CH", "SY", "TW", "TJ", "TZ", "TH", "TL", "TG", "TK", "TO", "TT", "TN", "TR", "TM", "TC", "TV", "UG", "UA", "AE", "GB", "US", "UM", "UY", "UZ", "VE", "VU", "VN", "VG", "VI", "WF", "YE", "ZW",}

	for _, input := range inputs {
		options.Search.Query = input
		switch options.Search.Source {
		case "fofa":
			for _, country := range ContriesCode {
				newQuery := fmt.Sprintf(`%v && country="%v"`, input, country)
				moreQueries = append(moreQueries, newQuery)
			}
			break
		}
	}

	return moreQueries
}
