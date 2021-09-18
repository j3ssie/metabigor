package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/j3ssie/metabigor/core"
	"github.com/spf13/cobra"
)

var options = core.Options{}

var RootCmd = &cobra.Command{
	Use:   "metabigor",
	Short: fmt.Sprintf(`Metabigor - Intelligence Tool but without API key - %v by %v`, core.VERSION, core.AUTHOR),
	Long:  fmt.Sprintf(`Metabigor - Intelligence Tool but without API key - %v by %v`, core.VERSION, core.AUTHOR),
}

// Execute main function
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVarP(&options.Scan.TmpOutput, "tmp", "T", "", "Temp Output folder")
	RootCmd.PersistentFlags().StringVar(&options.Proxy, "proxy", "", "Proxy for doing request")
	RootCmd.PersistentFlags().IntVarP(&options.Concurrency, "concurrency", "c", 5, "concurrency")
	RootCmd.PersistentFlags().IntVar(&options.Timeout, "timeout", 40, "timeout")
	RootCmd.PersistentFlags().StringVarP(&options.Input, "input", "i", "-", "input as a string, file or from stdin")
	RootCmd.PersistentFlags().StringVarP(&options.InputFile, "inputFile", "I", "-", "Input file")

	RootCmd.PersistentFlags().StringVarP(&options.Output, "output", "o", "out.txt", "output name")
	RootCmd.PersistentFlags().BoolVar(&options.Debug, "debug", false, "Debug")
	RootCmd.PersistentFlags().BoolVarP(&options.JsonOutput, "json", "j", false, "Output as JSON")
	RootCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose")
	RootCmd.SetHelpFunc(RootMessage)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if options.Debug {
		options.Verbose = true
	}
	core.InitLog(&options)

	if options.Scan.TmpOutput != "" && !core.FolderExists(options.Scan.TmpOutput) {
		core.InforF("Create new temp folder: %v", options.Scan.TmpOutput)
		os.MkdirAll(options.Scan.TmpOutput, 0750)
	}

	// got input from stdin
	if options.Input == "-" {
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			var data []string
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				input := strings.TrimSpace(sc.Text())
				if err := sc.Err(); err == nil && input != "" {
					data = append(data, input)
				}
			}
			options.Input = strings.Join(data, "\n")
		}
	} else {
		// get input from a file or just a string
		if core.FileExists(options.Input) {
			options.Input = core.GetFileContent(options.Input)
		}
	}

	// get input from a file or just a string
	if core.FileExists(options.InputFile) {
		options.Input = core.GetFileContent(options.InputFile)
	}

	core.InforF("Metabigor %v by %v", core.VERSION, core.AUTHOR)
	core.InforF(fmt.Sprintf("Store log file to: %v", options.LogFile))
}

// RootMessage print help message
func RootMessage(cmd *cobra.Command, _ []string) {
	fmt.Printf(cmd.UsageString())
	h := `
Examples Commands
=================

# discovery IP of a company/organization
echo "company" | metabigor net --org -o /tmp/result.txt

# discovery IP of an ASN
echo "ASN1111" | metabigor net --asn -o /tmp/result.txt
cat list_of_ASNs | metabigor net --asn -o /tmp/result.txt

# Only run masscan full ports
echo '1.2.3.4/24' | metabigor scan -o result.txt

# Only run nmap detail scan based on pre-scan data
echo '1.2.3.4:21' | metabigor scan -s
cat list_of_ip_with_port.txt | metabigor scan -c 10 --8 -s -o result.txt
cat list_of_ip_with_port.txt | metabigor scan -c 10 --tmp /tmp/raw-result/ -s -o result.txt
echo '1.2.3.4 -> [80,443,2222]' | metabigor scan -R

# Only run scan with zmap
cat ranges.txt | metabigor scan -p '443,80' -z

# certificate search info on crt.sh
echo 'Target' | metabigor cert

# Get Summary about IP address (powered by @thebl4ckturtle)
cat list_of_ips.txt | metabigor ipc --json

`
	fmt.Printf(h)
}
