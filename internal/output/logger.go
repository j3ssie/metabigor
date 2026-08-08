// Package output provides logging, result formatting, and file writing utilities.
//
// Results go to stdout; every log line goes to stderr, so `metabigor cert x.com |
// other-tool` stays clean no matter which log level is active.
package output

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	quietMode      bool
	verboseEnabled bool
	debugEnabled   bool
)

// SetupLogger configures log verbosity. --debug implies --verbose, and --quiet
// wins over both so `-q` always yields a silent run apart from errors.
func SetupLogger(quiet, verbose, debug, noColor bool) {
	quietMode = quiet
	debugEnabled = debug
	verboseEnabled = verbose || debug
	if noColor {
		color.NoColor = true
	}
}

// Info prints a one-line progress message. Hidden by --quiet.
func Info(format string, a ...any) {
	if quietMode {
		return
	}
	logf(color.HiBlueString("[info] "), format, a...)
}

// Good prints a success message. Hidden by --quiet.
func Good(format string, a ...any) {
	if quietMode {
		return
	}
	logf(color.HiGreenString("[+] "), format, a...)
}

// Warn prints a warning. Hidden by --quiet.
func Warn(format string, a ...any) {
	if quietMode {
		return
	}
	logf(color.HiYellowString("[!] "), format, a...)
}

// Error prints an error. Always shown, including under --quiet.
func Error(format string, a ...any) {
	logf(color.HiRedString("[-] "), format, a...)
}

// Verbose prints step-by-step progress. Requires --verbose or --debug.
func Verbose(format string, a ...any) {
	if !verboseEnabled || quietMode {
		return
	}
	logf(color.HiCyanString("[verbose] "), format, a...)
}

// Debug prints internal traces, including HTTP requests. Requires --debug.
func Debug(format string, a ...any) {
	if !debugEnabled {
		return
	}
	logf(color.HiMagentaString("[debug] "), format, a...)
}

func logf(prefix, format string, a ...any) {
	fmt.Fprintf(os.Stderr, prefix+format+"\n", a...)
}
