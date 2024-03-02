package cmd

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	jsoniter "github.com/json-iterator/go"
	"github.com/panjf2000/ants"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/thoas/go-funk"
)

func init() {
	var netCmd = &cobra.Command{
		Use:   "net",
		Short: "Discover Network Information about targets (same with net command but use static data)",
		Long:  core.DESC,
		RunE:  runNet,
	}

	netCmd.Flags().Bool("asn", false, "Take input as ASN")
	netCmd.Flags().Bool("org", false, "Take input as Organization")
	netCmd.Flags().Bool("ip", false, "Take input as a single IP address")
	netCmd.Flags().Bool("domain", false, "Take input as a domain")
	netCmd.Flags().BoolVarP(&options.Net.ExactMatch, "exact", "x", false, "Only get from highly trusted source")
	RootCmd.AddCommand(netCmd)

	var netdCmd = &cobra.Command{
		Use:   "netd",
		Short: "Discover Network Information about targets (similar with 'net' command but use 3rd data)",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Framework but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runNetD,
	}

	netdCmd.Flags().Bool("asn", false, "Take input as ASN")
	netdCmd.Flags().Bool("org", false, "Take input as Organization")
	netdCmd.Flags().Bool("ip", false, "Take input as a single IP address")
	netdCmd.Flags().Bool("domain", false, "Take input as a domain")
	netdCmd.Flags().BoolP("accurate", "x", false, "Only get from highly trusted source")
	RootCmd.AddCommand(netdCmd)
}

var ASNMap modules.AsnMap

func runNet(cmd *cobra.Command, _ []string) error {
	asn, _ := cmd.Flags().GetBool("asn")
	org, _ := cmd.Flags().GetBool("org")
	ip, _ := cmd.Flags().GetBool("ip")
	domain, _ := cmd.Flags().GetBool("domain")

	if asn {
		options.Net.SearchType = "asn"
	} else if org {
		options.Net.SearchType = "org"
	} else if ip {
		options.Net.SearchType = "ip"
	} else if domain {
		options.Net.SearchType = "domain"
	}
	if options.Net.SearchType == "" {
		fmt.Fprintf(os.Stderr, "You need to specify search type with one of these flag: --asn, --org or --ip")
		os.Exit(-1)
	}

	var err error
	ASNMap, err = modules.GetAsnMap()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error to generate asn info")
		os.Exit(-1)
	}

	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(options.Concurrency, func(i interface{}) {
		job := i.(string)
		var osintResult []string
		osintResult = runNetJob(job, options)
		StoreData(osintResult, options)
		wg.Done()
	}, ants.WithPreAlloc(true))
	defer p.Release()

	for _, target := range options.Inputs {
		wg.Add(1)
		_ = p.Invoke(strings.TrimSpace(target))
	}
	wg.Wait()

	if options.Output != "" && !core.FileExists(options.Output) {
		core.ErrorF("No data found")
	}
	return nil
}

func runNetJob(input string, options core.Options) []string {
	var data []string
	var asnInfos []modules.ASInfo

	if !options.Net.ExactMatch {
		input = strings.ToLower(input)

	}

	switch options.Net.SearchType {
	case "asn":
		input = strings.ToLower(input)
		if strings.Contains(input, "as") {
			input = strings.ReplaceAll(input, "as", "")
		}
		asInfos := ASNMap.ASInfo(cast.ToInt(input))
		if len(asInfos) > 0 {
			asnInfos = append(asnInfos, asInfos...)
		}

	case "org":
		asnNums := ASNMap.ASDesc(input)
		if len(asnNums) > 0 {
			for _, asnNum := range asnNums {
				asnInfos = append(asnInfos, ASNMap.ASInfo(asnNum)...)
			}
		}

	case "ip":
		asnInfos = append(asnInfos, searchByIP(input)...)

	case "domain":
		ips, err := net.LookupHost(input)
		if err == nil {
			for _, ip := range ips {
				asnInfos = append(asnInfos, searchByIP(ip)...)
			}
		}
	}

	if len(asnInfos) == 0 {
		core.ErrorF("No result found for: %s", input)
		return data
	}

	for _, asnInfo := range asnInfos {
		line := genOutput(asnInfo)
		data = append(data, line)
	}

	return data
}

func genOutput(asnInfo modules.ASInfo) string {
	var line string
	if options.JsonOutput {
		if content, err := jsoniter.MarshalToString(asnInfo); err == nil {
			return content
		}
		return line
	}
	if options.Verbose {
		line = fmt.Sprintf("%d - %s - %s - %s", asnInfo.Number, asnInfo.CIDR, asnInfo.Description, asnInfo.CountryCode)
	} else {
		line = asnInfo.CIDR
	}
	return line
}

func searchByIP(input string) []modules.ASInfo {
	var asnInfo []modules.ASInfo

	ip, err := netip.ParseAddr(input)
	if err != nil {
		return asnInfo
	}

	if asn := ASNMap.ASofIP(ip); asn.AS != 0 {
		return ASNMap.ASInfo(asn.AS)
	}
	return asnInfo
}

/////////// netd command

func runNetD(cmd *cobra.Command, _ []string) error {
	asn, _ := cmd.Flags().GetBool("asn")
	org, _ := cmd.Flags().GetBool("org")
	ip, _ := cmd.Flags().GetBool("ip")
	domain, _ := cmd.Flags().GetBool("domain")
	options.Net.Optimize, _ = cmd.Flags().GetBool("accurate")

	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(options.Concurrency, func(i interface{}) {
		job := i.(string)
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

		wg.Done()
	}, ants.WithPreAlloc(true))
	defer p.Release()

	for _, target := range options.Inputs {
		wg.Add(1)
		_ = p.Invoke(strings.TrimSpace(target))
	}

	wg.Wait()

	if options.Output != "" && !core.FileExists(options.Output) {
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
	if len(data) == 0 && !options.PipeTheOutput {
		core.ErrorF("Empty data to write")
		return
	}

	fmt.Println(strings.Join(data, "\n"))
	_, err := core.AppendToContent(options.Output, strings.Join(data, "\n"))
	if err == nil {
		core.InforF("Write output to: %v", options.Output)
	}
}
