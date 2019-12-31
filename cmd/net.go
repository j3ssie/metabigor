package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

var netCmd *cobra.Command

func init() {
	// byeCmd represents the bye command
	var netCmd = &cobra.Command{
		Use:   "net",
		Short: "Discover Network Inforamtion about targets",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Framework but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runNet,
	}

	netCmd.Flags().Bool("asn", false, "Take input as ASN")
	netCmd.Flags().Bool("org", false, "Take input as Organization")
	RootCmd.AddCommand(netCmd)
}

func runNet(cmd *cobra.Command, args []string) error {
	asn, _ := cmd.Flags().GetBool("asn")
	org, _ := cmd.Flags().GetBool("org")
	var inputs []string

	if strings.Contains(options.Input, "\n") {
		inputs = strings.Split(options.Input, "\n")
	} else {
		inputs = append(inputs, options.Input)
	}

	var wg sync.WaitGroup
	jobs := make(chan string)

	for i := 0; i < options.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// do real stuff here
			for job := range jobs {
				var osintResult []string
				if asn {
					osintResult = runASN(job, options)
				} else if org {
					osintResult = runOrg(job, options)
				}
				StoreData(osintResult, options)
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
	return nil
}

func runASN(input string, options core.Options) []string {
	core.BannerF("Starting get IP Info from ASN: ", input)
	options.Net.Asn = input
	var data []string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		data = append(data, modules.ASNBgpDotNet(options)...)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		data = append(data, modules.IPInfo(options)...)
		wg.Done()
	}()

	// wg.Add(1)
	// go func() {
	// 	data = append(data, modules.ASNSpyse(options)...)
	// 	wg.Done()
	// }()

	wg.Wait()
	return data
}

func runOrg(input string, options core.Options) []string {
	core.BannerF("Starting get IP Info for Organization: ", input)
	options.Net.Org = input
	var data []string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		data = append(data, modules.OrgBgpDotNet(options)...)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		data = append(data, modules.ASNLookup(options)...)
		wg.Done()
	}()
	wg.Wait()

	var cidrs []string
	// get more IP by result ASN
	for _, item := range data {
		// get more range from ASN
		if strings.HasPrefix(strings.ToLower(item), "as") {
			wg.Add(1)
			go func(item string) {
				cidrs = append(cidrs, runASN(item, options)...)
				wg.Done()
			}(item)
			continue
		} else if core.StartWithNum(item) {
			cidrs = append(cidrs, item)
		}
	}
	wg.Wait()
	return funk.Uniq(cidrs).([]string)
}

// StoreData store data to output
func StoreData(data []string, options core.Options) {
	if len(data) > 0 {
		fmt.Println(strings.Join(data, "\n"))
		_, err := core.AppendToContent(options.Output, strings.Join(data, "\n"))
		if err == nil {
			core.InforF("Write output to: %v", options.Output)
		}
	}
}
