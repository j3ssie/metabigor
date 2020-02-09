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
var config struct {
	defaultSign  string
	secretCollab string
	port         string
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "metabigor",
	Short: "Metabigor",
	Long:  fmt.Sprintf(`Metabigor - Intelligence Framework but without API key - %v by %v`, core.VERSION, core.AUTHOR),
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
	RootCmd.PersistentFlags().StringVar(&options.Scan.TmpOutput, "tmp", "", "Temp Output folder")
	RootCmd.PersistentFlags().StringVar(&options.Proxy, "proxy", "", "Proxy for doing request")
	RootCmd.PersistentFlags().IntVarP(&options.Concurrency, "concurrency", "c", 5, "concurrency")
	RootCmd.PersistentFlags().IntVar(&options.Timeout, "timeout", 15, "timeout")
	RootCmd.PersistentFlags().StringVarP(&options.Input, "input", "i", "-", "input as a string, file or from stdin")
	RootCmd.PersistentFlags().StringVarP(&options.Output, "output", "o", "out.txt", "output name")
	RootCmd.PersistentFlags().BoolVar(&options.Debug, "debug", false, "Debug")
	RootCmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "v", false, "Verbose")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if options.Debug {
		options.Verbose = true
	}
	core.InitLog(options)
	// planned feature
	// if !core.FileExists(options.ConfigFile) {
	// 	core.InitConfig(options)
	// }
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
				data = append(data, sc.Text())
			}
			options.Input = strings.Join(data, "\n")
		}
	} else {
		// get input from a file or just a string
		if core.FileExists(options.Input) {
			options.Input = core.GetFileContent(options.Input)
		}
	}
}
