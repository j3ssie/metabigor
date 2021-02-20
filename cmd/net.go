package cmd

import (
	"fmt"
	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"inet.af/netaddr"
	"net"
	"os"
	"strings"
	"sync"
)

func init() {
	var netCmd = &cobra.Command{
		Use:   "net",
		Short: "Discover Network Information about targets (same with net command but use static data)",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Framework but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runNet,
	}

	netCmd.Flags().Bool("asn", false, "Take input as ASN")
	netCmd.Flags().Bool("org", false, "Take input as Organization")
	netCmd.Flags().Bool("ip", false, "Take input as a single IP address")
	netCmd.Flags().Bool("domain", false, "Take input as a domain")
	netCmd.Flags().BoolVarP(&options.Net.ExactMatch, "exact", "x", false, "Only get from highly trusted source")
	RootCmd.AddCommand(netCmd)
}

var ASNMap modules.AsnMap

func runNet(cmd *cobra.Command, _ []string) error {
	asn, _ := cmd.Flags().GetBool("asn")
	org, _ := cmd.Flags().GetBool("org")
	ip, _ := cmd.Flags().GetBool("ip")
	domain, _ := cmd.Flags().GetBool("domain")

	// prepare input
	var inputs []string
	if strings.Contains(options.Input, "\n") {
		inputs = strings.Split(options.Input, "\n")
	} else {
		inputs = append(inputs, options.Input)
	}

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
	jobs := make(chan string)

	for i := 0; i < options.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// do real stuff here
			for job := range jobs {
				var osintResult []string
				osintResult = runNetJob(job, options)
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

	ip, err := netaddr.ParseIP(input)
	if err != nil {
		return asnInfo
	}

	if asn := ASNMap.ASofIP(ip); asn.AS != 0 {
		return ASNMap.ASInfo(asn.AS)
	}
	return asnInfo
}