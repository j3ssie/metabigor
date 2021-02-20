package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

func init() {
	var netCmd = &cobra.Command{
		Use:   "netd",
		Short: "Discover Network Information about targets (same with net command but use 3rd data)",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Framework but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runNetD,
	}

	netCmd.Flags().Bool("asn", false, "Take input as ASN")
	netCmd.Flags().Bool("org", false, "Take input as Organization")
	netCmd.Flags().Bool("ip", false, "Take input as a single IP address")
	netCmd.Flags().Bool("domain", false, "Take input as a domain")
	netCmd.Flags().BoolP("accurate", "x", false, "Only get from highly trusted source")

	RootCmd.AddCommand(netCmd)
}

func runNetD(cmd *cobra.Command, _ []string) error {
	asn, _ := cmd.Flags().GetBool("asn")
	org, _ := cmd.Flags().GetBool("org")
	ip, _ := cmd.Flags().GetBool("ip")
	domain, _ := cmd.Flags().GetBool("domain")
	options.Net.Optimize, _ = cmd.Flags().GetBool("accurate")

	var inputs []string

	if options.Input == "-" || options.Input == "" {
		core.ErrorF("No input found")
		os.Exit(1)
	}

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
				} else if ip {
					options.Net.IP = job
					osintResult = runSingle(job, options)
				} else if domain {
					options.Net.Domain = job
					osintResult = runSingle(job, options)
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

func runSingle(input string, options core.Options) []string {
	core.BannerF("Starting get ASN from: ", input)
	var data []string
	ans := modules.ASNFromIP(options)

	// get more IP by result ASN
	for _, item := range ans {
		if strings.HasPrefix(strings.ToLower(item), "as") {
			data = append(data, runASN(item, options)...)
		}
	}
	return data
}

func runASN(input string, options core.Options) []string {
	core.BannerF("Starting get IP Info from ASN: ", input)
	options.Net.Asn = input
	var data []string
	var wg sync.WaitGroup

	//wg.Add(1)
	//go func() {
	//	data = append(data, modules.ASNBgpDotNet(options)...)
	//	wg.Done()
	//}()

	wg.Add(1)
	go func() {
		data = append(data, modules.GetIPInfo(options)...)
		wg.Done()
	}()

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
		data = append(data, modules.OrgBgbView(options)...)
		wg.Done()
	}()

	// disable when enable trusted source
	if !options.Net.Optimize {
		wg.Add(1)
		go func() {
			data = append(data, modules.ASNLookup(options)...)
			wg.Done()
		}()
	}
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
	if len(data) == 0 {
		core.ErrorF("Empty data to write")
		return
	}

	fmt.Println(strings.Join(data, "\n"))
	_, err := core.AppendToContent(options.Output, strings.Join(data, "\n"))
	if err == nil {
		core.InforF("Write output to: %v", options.Output)
	}
}