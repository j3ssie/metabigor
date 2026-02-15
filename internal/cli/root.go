// Package cli implements the command-line interface for metabigor using Cobra.
package cli

import (
	"fmt"
	"os"

	"github.com/j3ssie/metabigor/internal/options"
	"github.com/spf13/cobra"
)

var opt options.Options

var rootCmd = &cobra.Command{
	Use:   "metabigor",
	Short: "OSINT power without API key hassle",
	Long:  rootLong,
	// Example is set in helptext.go init()
}

func init() {
	pf := rootCmd.PersistentFlags()

	pf.StringVarP(&opt.Input, "input", "i", "", "Target to scan (also accepts first argument or stdin)")
	pf.StringVarP(&opt.InputFile, "inputFile", "I", "", "File containing list of targets, one per line")
	pf.StringVarP(&opt.Output, "output", "o", "", "Write results to file (default: stdout only)")
	pf.IntVarP(&opt.Concurrency, "concurrency", "c", 5, "Number of parallel workers")
	pf.IntVarP(&opt.Timeout, "timeout", "t", 40, "Request timeout in seconds")
	pf.IntVar(&opt.Retry, "retry", 3, "Max retries on failed requests")
	pf.StringVar(&opt.Proxy, "proxy", "", "Upstream proxy (e.g. http://127.0.0.1:8080)")
	pf.BoolVarP(&opt.Silent, "silent", "q", false, "Hide progress messages (only show errors)")
	pf.BoolVar(&opt.Debug, "debug", false, "Show HTTP requests, responses, and internal traces")
	pf.BoolVar(&opt.JSONOutput, "json", false, "Format output as JSON lines")
	pf.BoolVar(&opt.NoColor, "no-color", false, "Strip ANSI colors from log output")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
