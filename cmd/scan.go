package cmd

import (
	"fmt"
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
		Short: "Wrapper to run scan from input",
		Long:  fmt.Sprintf(`Metabigor - Intelligence Tool but without API key - %v by %v`, core.VERSION, core.AUTHOR),
		RunE:  runScan,
	}

	scanCmd.Flags().StringP("ports", "p", "0-65535", "Port range for previous command")
	scanCmd.Flags().StringP("rate", "r", "3000", "rate limit for masscan command")
	scanCmd.Flags().BoolP("detail", "D", false, "Do Nmap scan based on previous output")
	scanCmd.Flags().BoolP("all", "A", false, "Join all inputs to a file first")

	scanCmd.Flags().BoolP("flat", "f", true, "format output like this: 1.2.3.4:443")
	scanCmd.Flags().BoolP("nmap", "n", false, "Use nmap instead of masscan for overview scan")
	scanCmd.Flags().BoolP("zmap", "z", false, "Only scan range with zmap")
	scanCmd.Flags().BoolP("skip-masscan", "s", false, "run nmap from input format like this: 1.2.3.4:443")

	scanCmd.Flags().StringP("script", "S", "", "nmap scripts")
	scanCmd.Flags().String("nmap-command", "sudo nmap -sSV -p {{.ports}} {{.input}} {{.script}} -T4 --open -oA {{.output}}", "Nmap template command to run")
	scanCmd.Flags().StringP("grep", "g", "", "match string to confirm script success")
	// only parse scan
	scanCmd.Flags().StringP("result-folder", "R", "", "Result folder")
	scanCmd.SetHelpFunc(ScanHelp)
	RootCmd.AddCommand(scanCmd)

}

func runScan(cmd *cobra.Command, _ []string) error {
	options.Scan.NmapTemplate, _ = cmd.Flags().GetString("nmap-command")
	options.Scan.NmapScripts, _ = cmd.Flags().GetString("script")
	options.Scan.GrepString, _ = cmd.Flags().GetString("grep")
	options.Scan.Ports, _ = cmd.Flags().GetString("ports")
	options.Scan.Rate, _ = cmd.Flags().GetString("rate")
	options.Scan.Detail, _ = cmd.Flags().GetBool("detail")
	options.Scan.Flat, _ = cmd.Flags().GetBool("flat")
	options.Scan.All, _ = cmd.Flags().GetBool("all")
	options.Scan.NmapOverview, _ = cmd.Flags().GetBool("nmap")
	options.Scan.ZmapOverview, _ = cmd.Flags().GetBool("zmap")
	options.Scan.SkipOverview, _ = cmd.Flags().GetBool("skip-masscan")
	// only parse result
	resultFolder, _ := cmd.Flags().GetString("result-folder")
	if resultFolder != "" {
		parseResult(resultFolder, options)
		os.Exit(0)
	}

	if options.Input == "-" || options.Input == "" {
		core.ErrorF("No input found")
		os.Exit(1)
	}

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

	if options.Scan.All || options.Scan.ZmapOverview {
		options.Scan.InputFile = StoreTmpInput(inputs, options)
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

	for _, input := range inputs {
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
	if input == "" {
		return []string{}
	}
	if len(strings.Split(input, ":")) == 1 {
		return []string{}
	}
	host := strings.Split(input, ":")[0]
	ports := strings.Split(input, ":")[1]
	core.BannerF("Run detail scan on: ", fmt.Sprintf("%v:%v", host, ports))
	return modules.RunNmap(host, ports, options)
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

	// massscan
	for _, file := range Files {
		filename := file.Name()
		core.DebugF("Reading: %v", filename)
		if strings.HasSuffix(file.Name(), "xml") && strings.HasPrefix(filename, "masscan") {
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
	tmpFile, _ := ioutil.TempFile(options.Scan.TmpOutput, "zmap-*.txt")
	if tmpDest != "" {
		tmpFile, _ = ioutil.TempFile(tmpDest, "zmap-input-*.txt")
	}
	tmpDest = tmpFile.Name()
	core.WriteToFile(tmpDest, strings.Join(raw, "\n"))
	return tmpDest
}

// ScanHelp print help message
func ScanHelp(cmd *cobra.Command, _ []string) {
	fmt.Println(cmd.UsageString())
	h := "\nExample Commands:\n"
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
