// Package cli implements the command-line interface for metabigor using Cobra.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/j3ssie/metabigor/internal/core"
	"github.com/j3ssie/metabigor/internal/options"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

// Command groups, so `metabigor --help` reads as a table of contents rather
// than an alphabetical pile.
const (
	groupRecon  = "recon"
	groupEnrich = "enrich"
	groupUtil   = "util"
)

var opt options.Options

var rootCmd = &cobra.Command{
	Use:               core.NAME,
	Short:             core.DESC,
	Long:              rootLong,
	Example:           rootExample,
	Version:           core.VERSION,
	SilenceErrors:     true, // errors are printed by Execute, with our prefix
	PersistentPreRunE: setupGlobals,
}

func init() {
	// Keep flags in declaration order (input, output, execution, logging)
	// rather than alphabetical, which scatters related flags. Cobra copies
	// this setting from Flags() when it renders help.
	rootCmd.Flags().SortFlags = false

	// Registering --help and --version ourselves replaces Cobra's terse
	// defaults and keeps both meta flags together at the top of the list.
	// Cobra still handles them, since it only adds its own when absent.
	rootCmd.Flags().BoolP("help", "h", false, "Show help")
	rootCmd.Flags().Bool("version", false, "Print the version and exit")

	pf := rootCmd.PersistentFlags()
	pf.SortFlags = false

	pf.StringVarP(&opt.Input, "input", "i", "", "Target to look up (also accepts arguments or stdin)")
	pf.StringVarP(&opt.InputFile, "input-file", "I", "", "File of targets, one per line (use - for stdin)")
	pf.StringVarP(&opt.Output, "output", "o", "", "Also write results to this file")
	pf.StringVarP(&opt.Format, "format", "f", string(output.FormatText),
		"Output format: "+strings.Join(output.FormatNames(), ", "))
	pf.BoolVar(&opt.Append, "append", false, "Append to the output file instead of overwriting it")
	pf.IntVarP(&opt.Concurrency, "concurrency", "c", 10, "Number of parallel workers")
	pf.IntVarP(&opt.Timeout, "timeout", "t", 40, "Request timeout in seconds")
	pf.IntVar(&opt.Retry, "retry", 3, "Retries per failed request")
	pf.StringVar(&opt.Proxy, "proxy", "", "Upstream proxy, e.g. http://127.0.0.1:8080")
	pf.BoolVarP(&opt.Verbose, "verbose", "v", false, "Show step-by-step progress")
	pf.BoolVarP(&opt.Quiet, "quiet", "q", false, "Show results and errors only")
	pf.BoolVar(&opt.Debug, "debug", false, "Show HTTP traffic and internal traces (implies --verbose)")
	pf.BoolVar(&opt.NoColor, "no-color", false, "Disable colored log output")

	rootCmd.MarkFlagsMutuallyExclusive("verbose", "quiet")

	rootCmd.AddGroup(
		&cobra.Group{ID: groupRecon, Title: "Recon commands:"},
		&cobra.Group{ID: groupEnrich, Title: "Enrichment commands:"},
		&cobra.Group{ID: groupUtil, Title: "Maintenance commands:"},
	)
	// Park the built-in help and completion commands in a group so they stop
	// forming their own "Additional Commands" section.
	rootCmd.SetHelpCommandGroupID(groupUtil)
	rootCmd.SetCompletionCommandGroupID(groupUtil)
	rootCmd.SetVersionTemplate(versionTemplate())
}

// setupGlobals validates the global flags once, before any command runs.
func setupGlobals(cmd *cobra.Command, _ []string) error {
	// From here on, failures are runtime problems rather than usage mistakes,
	// so report them without dumping the whole usage block. Flag parse errors
	// happen before this hook and still show usage.
	cmd.SilenceUsage = true

	output.SetupLogger(opt.Quiet, opt.Verbose, opt.Debug, opt.NoColor)

	// Reject a bad --format up front, before a command loads a database or
	// starts a browser. setup() re-parses the validated value.
	if _, err := output.ParseFormat(opt.Format); err != nil {
		return err
	}

	if opt.Concurrency < 1 {
		return fmt.Errorf("--concurrency must be at least 1, got %d", opt.Concurrency)
	}
	if opt.Timeout < 1 {
		return fmt.Errorf("--timeout must be at least 1 second, got %d", opt.Timeout)
	}
	return nil
}

// runContext holds what every command needs once flags are settled.
type runContext struct {
	Inputs []string
	Writer *output.Writer
}

// setup reads the targets and opens the result writer. Every command starts
// with this, so input handling and output formatting stay identical across all
// of them.
func setup(cmd *cobra.Command) (*runContext, error) {
	format, err := output.ParseFormat(opt.Format)
	if err != nil {
		return nil, err
	}

	inputs, err := runner.ReadInputs(opt.Input, opt.InputFile, cmd.Flags().Args())
	if err != nil {
		return nil, err
	}
	if len(inputs) == 0 {
		return nil, noInputError(cmd)
	}

	w, err := output.NewWriter(opt.Output, format, opt.Append)
	if err != nil {
		return nil, err
	}
	return &runContext{Inputs: inputs, Writer: w}, nil
}

// Close flushes the writer and reports when a run produced nothing, which is
// otherwise indistinguishable from a silent failure.
func (rc *runContext) Close() {
	if rc.Writer.Count() == 0 {
		if opt.Output != "" {
			output.Warn("No results found, leaving %s unchanged", opt.Output)
		} else {
			output.Warn("No results found")
		}
	}
	if err := rc.Writer.Close(); err != nil {
		output.Error("close output file: %v", err)
	}
}

// noInputError explains the four ways to pass a target, using the command the
// user actually typed.
func noInputError(cmd *cobra.Command) error {
	name := cmd.CommandPath()
	return fmt.Errorf(`no target given. Provide one of:

  %s %s
  %s -i %s
  %s -I targets.txt
  cat targets.txt | %s`,
		name, cmd.Annotations["sample"],
		name, cmd.Annotations["sample"],
		name,
		name)
}

// versionTemplate keeps {{.Version}} as a placeholder so ldflags-injected
// build metadata is reflected here too.
func versionTemplate() string {
	return fmt.Sprintf("%s {{.Version}} - %s\n", core.NAME, core.DESC)
}

// rootLong is built from the shared constants so the tagline and the credit
// cannot drift from what `metabigor version` prints.
var rootLong = fmt.Sprintf("%s - %s %s", core.NAME, core.DESC, dim("- Crafted with <3 by "+core.AUTHOR)) + `

Map a target's infrastructure without registering for a single API key:
network ranges, certificates, related domains, exposed ports, and code leaks.

Every command takes targets the same four ways - as arguments, with -i, from a
file with -I, or on stdin - and renders results through -f/--format, so any
command pipes cleanly into the next.`

var rootExample = examples(
	"pass a target as an argument", "metabigor net AS13335",
	"or with a flag", "metabigor cert -i hackerone.com",
	"or from a file, one per line", "metabigor ip -I ips.txt",
	"or on stdin", "cat domains.txt | metabigor related",
	"several targets at once", "metabigor cluster 1.1.1.1 8.8.8.8 9.9.9.9",
	"pick an output format", "metabigor net AS13335 -f csv -o ranges.csv",
	"chain commands together", "metabigor net AS13335 -f flat | metabigor ip -f flat",
	"refresh the offline databases", "metabigor update",
)

// Execute runs the root command and maps any failure to a non-zero exit code.
func Execute() {
	// Applied here rather than in each command's init(), which runs before
	// every subcommand has registered itself.
	for _, c := range rootCmd.Commands() {
		c.Flags().SortFlags = false
	}
	if err := rootCmd.Execute(); err != nil {
		output.Error("%v", err)
		os.Exit(1)
	}
}
