package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	"github.com/spf13/cobra"
)

var scanCmd *cobra.Command

func init() {
	// byeCmd represents the bye command
	var scanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Wrapper to run scan from input",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Framework but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runScan,
	}

	scanCmd.Flags().StringP("ports", "p", "0-65535", "Port range for previous command")
	scanCmd.Flags().StringP("rate", "r", "5000", "Port range for previous command")
	scanCmd.Flags().Bool("detail", false, "Do Nmap scan based on previous output")
	scanCmd.Flags().BoolP("flat", "f", false, "format output like this: 1.2.3.4:443")
	scanCmd.Flags().BoolP("skip-masscan", "s", false, "run nmap from input format like this: 1.2.3.4:443")
	scanCmd.Flags().String("nmap-script", "", "nmap scripts")
	RootCmd.AddCommand(scanCmd)

}

func runScan(cmd *cobra.Command, args []string) error {
	options.Scan.NmapScripts, _ = cmd.Flags().GetString("nmap-script")
	options.Scan.Ports, _ = cmd.Flags().GetString("ports")
	options.Scan.Rate, _ = cmd.Flags().GetString("rate")
	options.Scan.Detail, _ = cmd.Flags().GetBool("detail")
	options.Scan.Flat, _ = cmd.Flags().GetBool("flat")
	options.Scan.SkipOverview, _ = cmd.Flags().GetBool("skip-masscan")

	var inputs []string
	if strings.Contains(options.Input, "\n") {
		inputs = strings.Split(options.Input, "\n")
	} else {
		inputs = append(inputs, options.Input)
	}

	var result []string
	// var detailResult []string
	var wg sync.WaitGroup
	jobs := make(chan string)

	for i := 0; i < options.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// do real stuff here
			for job := range jobs {
				if options.Scan.SkipOverview {
					result = directDetail(job, options)
				} else {
					result = runRoutine(job, options)
				}
				StoreData(result, options)
			}
		}()
	}

	for _, input := range inputs {
		jobs <- input
	}

	close(jobs)
	wg.Wait()

	return nil
}

func runRoutine(input string, options core.Options) []string {
	core.BannerF("Run quick scan on: ", input)
	var data []string
	data = append(data, modules.RunMasscan(input, options)...)
	if !options.Scan.Detail {
		return data
	}

	var wg sync.WaitGroup
	var realData []string
	for _, item := range data {
		wg.Add(1)
		go func(item string) {
			realData = append(realData, runDetail(item, options)...)
			wg.Done()
		}(item)
	}
	wg.Wait()
	return realData
}

func runDetail(input string, options core.Options) []string {
	host := strings.Split(input, " - ")[0]
	ports := strings.Split(input, " - ")[1]
	core.BannerF("Run detail scan on: ", fmt.Sprintf("%v %v", host, ports))
	return modules.RunNmap(host, ports, options)
}

func directDetail(input string, options core.Options) []string {
	host := strings.Split(input, ":")[0]
	ports := strings.Split(input, ":")[1]
	core.BannerF("Run detail scan on: ", fmt.Sprintf("%v %v", host, ports))
	return modules.RunNmap(host, ports, options)
}
