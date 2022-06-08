package cmd

import (
	"fmt"
	"github.com/thoas/go-funk"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	"github.com/spf13/cobra"
)

func init() {
	var scanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Wrapper to run port scan from provided input",
		Long:  core.DESC,
		RunE:  runScan,
	}
	// scan options
	scanCmd.Flags().StringVarP(&options.Scan.Ports, "ports", "p", "0-65535", "Port range for previous command")
	scanCmd.Flags().StringVarP(&options.Scan.Rate, "rate", "r", "3000", "rate limit for masscan command")
	scanCmd.Flags().BoolVarP(&options.Scan.All, "join", "A", false, "Join all inputs to a file first then do a scan")
	// scan strategy option
	scanCmd.Flags().BoolVarP(&options.Scan.Flat, "flat", "f", true, "format output like this: 1.2.3.4:443")
	scanCmd.Flags().BoolVarP(&options.Scan.NmapOverview, "nmap", "n", false, "Use nmap instead of masscan for overview scan")
	scanCmd.Flags().BoolVarP(&options.Scan.ZmapOverview, "zmap", "z", false, "Only scan range with zmap")
	scanCmd.Flags().BoolVarP(&options.Scan.SkipOverview, "skip-masscan", "s", false, "run nmap from input format like this: 1.2.3.4:443")
	scanCmd.Flags().BoolVarP(&options.Scan.InputFromRustScan, "rstd", "R", false, "run nmap from rustscan input format like: 1.2.3.4 -> [80,443,8080,8443,8880]")
	// more nmap options
	scanCmd.Flags().StringVarP(&options.Scan.NmapScripts, "script", "S", "", "nmap scripts")
	scanCmd.Flags().StringVar(&options.Scan.NmapTemplate, "nmap-command", "nmap -sSV -sC -p {{.ports}} {{.input}} {{.script}} -T4 --open -oA {{.output}}", "Nmap template command to run")
	scanCmd.Flags().StringVar(&options.Scan.GrepString, "grep", "", "match string to confirm script success")
	scanCmd.Flags().String("result-folder", "", "Result folder")
	scanCmd.Flags().BoolVar(&options.Scan.IPv4, "4", true, "Filter input to only get ipv4")
	scanCmd.Flags().BoolVar(&options.Scan.Skip80And443, "8", false, "Skip ports 80 and 443. Useful when you want to look for service behind the list of pre-scanned data")
	//scanCmd.Flags().Bool("6",  false, "Filter input to only get ipv4")
	scanCmd.Flags().BoolP("detail", "D", false, "Do Nmap scan based on previous output")
	scanCmd.Flags().Bool("uniq", true, "Unique input first")
	scanCmd.SetHelpFunc(ScanHelp)
	RootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, _ []string) error {
	// only parse result
	resultFolder, _ := cmd.Flags().GetString("result-folder")
	uniq, _ := cmd.Flags().GetBool("uniq")
	if resultFolder != "" {
		parseResult(resultFolder, options)
		os.Exit(0)
	}

	if options.Scan.InputFromRustScan {
		options.Scan.SkipOverview = true
	}

	// make sure input is valid
	if options.Scan.IPv4 {
		// only filter when run zmap
		if !options.Scan.SkipOverview {
			options.Inputs = core.FilterIpv4(options.Inputs)
		}
	}
	if uniq {
		options.Inputs = funk.UniqString(options.Inputs)
	}
	if len(options.Inputs) == 0 {
		core.ErrorF("No input provided")
		os.Exit(1)
	}

	var result []string
	var wg sync.WaitGroup
	jobs := make(chan string)

	if options.Scan.All || options.Scan.ZmapOverview {
		options.Scan.InputFile = StoreTmpInput(options.Inputs, options)
		core.DebugF("Store temp input in: %v", options.Scan.InputFile)

		if options.Scan.ZmapOverview {
			ports := core.GenPorts(options.Scan.Ports)
			core.DebugF("Run port scan with: %v", strings.Trim(strings.Join(ports, ","), ","))
			if options.Scan.InputFile == "" || len(ports) == 0 {
				core.ErrorF("Error gen input or ports")
				return nil
			}
			for i := 0; i < options.Concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for job := range jobs {
						// do real stuff here
						core.BannerF("Run zmap scan on port ", job)
						result = modules.RunZmap(options.Scan.InputFile, job, options)
						StoreData(result, options)
					}
				}()
			}
			for _, port := range ports {
				jobs <- port
			}
			close(jobs)
			wg.Wait()
			return nil
		}

		core.BannerF("Run overview scan on port ", options.Scan.InputFile)
		if options.Scan.NmapOverview {
			result = modules.RunNmap(options.Scan.InputFile, "", options)
		} else {
			result = modules.RunMasscan(options.Scan.InputFile, options)
		}
		StoreData(result, options)
		return nil
	}

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

	for _, input := range options.Inputs {
		jobs <- input
	}

	close(jobs)
	wg.Wait()

	return nil
}

func runRoutine(input string, options core.Options) []string {
	var data []string
	core.BannerF("Run overview scan on: ", input)
	if options.Scan.NmapOverview {
		data = append(data, modules.RunNmap(input, "", options)...)
	} else {
		data = append(data, modules.RunMasscan(input, options)...)
	}

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
	if options.Scan.Flat {
		return directDetail(input, options)
	}
	if input == "" {
		return []string{}
	}
	if len(strings.Split(input, " - ")) == 1 {
		return []string{}
	}

	host := strings.Split(input, " - ")[0]
	ports := strings.Split(input, " - ")[1]
	core.BannerF("Run detail scan on: ", fmt.Sprintf("%v:%v", host, ports))
	return modules.RunNmap(host, ports, options)
}

func directDetail(input string, options core.Options) []string {
	var out []string
	if options.Scan.Skip80And443 {
		if strings.HasSuffix(input, ":80") && strings.HasSuffix(input, ":443") {
			return out
		}
	}

	if input == "" {
		return out
	}
	var host, ports string

	if options.Scan.InputFromRustScan {
		// 1.1.1.1 -> [80,443,2095,2096,8080,8443,8880]
		if !strings.Contains(input, "->") {
			return out
		}
		host = strings.Split(input, " -> ")[0]
		ports = strings.Split(input, " -> ")[1]
		ports = strings.TrimLeft(strings.TrimRight(ports, "]"), "[")
	} else {
		if len(strings.Split(input, ":")) == 1 {
			return out
		}
		host = strings.Split(input, ":")[0]
		ports = strings.Split(input, ":")[1]
	}

	core.BannerF("Run detail scan on: ", fmt.Sprintf("%v:%v", host, ports))
	out = modules.RunNmap(host, ports, options)
	return out
}

// only parse result
func parseResult(resultFolder string, options core.Options) {
	if !core.FolderExists(resultFolder) {
		core.ErrorF("Result Folder not found: ", resultFolder)
		return
	}
	core.BannerF("Reading result from: ", fmt.Sprintf("%v", resultFolder))
	Files, err := ioutil.ReadDir(resultFolder)
	if err != nil {
		return
	}

	if options.Scan.Detail {
		// nmap
		for _, file := range Files {
			filename := file.Name()
			core.DebugF("Reading: %v", filename)
			if strings.HasSuffix(file.Name(), "xml") && strings.HasPrefix(filename, "nmap") {
				data := core.GetFileContent(filename)
				result := modules.ParseNmap(data, options)
				if len(result) > 0 {
					fmt.Printf(strings.Join(result, "\n"))
				}
			}
		}
		return
	}

	// masscan
	for _, file := range Files {
		filename := file.Name()
		core.DebugF("Reading: %v", filename)
		if strings.HasPrefix(filename, "masscan") {
			data := core.GetFileContent(filename)
			fmt.Println(data)
			rawResult := modules.ParsingMasscan(data)
			fmt.Println(rawResult)
			for k, v := range rawResult {
				for _, port := range v {
					fmt.Printf("%v:%v\n", k, port)
				}
			}
		}
	}
}

// StoreTmpInput store list of string to tmp file
func StoreTmpInput(raw []string, options core.Options) string {
	tmpDest := options.Scan.TmpOutput
	tmpFile, _ := ioutil.TempFile(options.Scan.TmpOutput, "joined-*.txt")
	if tmpDest != "" {
		tmpFile, _ = ioutil.TempFile(tmpDest, "joined-input-*.txt")
	}
	tmpDest = tmpFile.Name()
	core.WriteToFile(tmpDest, strings.Join(raw, "\n"))
	return tmpDest
}

// ScanHelp print help message
func ScanHelp(cmd *cobra.Command, _ []string) {
	fmt.Println(cmd.UsageString())
	h := "\nExample Commands:\n"
	h += "  # Run Nmap with output from rustscan\n"
	h += "  echo '1.2.3.4 -> [80,443,2222]' | metabigor scan -R\n"
	h += "  # Only run masscan full ports\n"
	h += "  echo '1.2.3.4/24' | metabigor scan -o result.txt\n\n"
	h += "  # Only run nmap detail scan\n"
	h += "  echo '1.2.3.4:21' | metabigor scan -s -c 10\n"
	h += "  echo '1.2.3.4:21' | metabigor scan --tmp /tmp/raw-result/ -s -o result.txt\n\n"
	h += "  # Only run scan with zmap \n"
	h += "  cat ranges.txt | metabigor scan -p '443,80' -z\n"
	h += "\n"
	fmt.Printf(h)
}
