// Package output provides logging, formatting, and file writing utilities.
package output

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	silentMode   bool
	debugEnabled bool
)

// SetupLogger configures the logger.
func SetupLogger(silent, debug, noColor bool) {
	silentMode = silent
	debugEnabled = debug
	if noColor {
		color.NoColor = true
	}
}

// Info prints an informational message.
func Info(format string, a ...any) {
	if silentMode {
		return
	}
	prefix := color.HiBlueString("[info] ")
	fmt.Fprintf(os.Stderr, prefix+format+"\n", a...)
}

// Good prints a success message.
func Good(format string, a ...any) {
	if silentMode {
		return
	}
	prefix := color.HiGreenString("[+] ")
	fmt.Fprintf(os.Stderr, prefix+format+"\n", a...)
}

// Warn prints a warning message.
func Warn(format string, a ...any) {
	if silentMode {
		return
	}
	prefix := color.HiYellowString("[!] ")
	fmt.Fprintf(os.Stderr, prefix+format+"\n", a...)
}

// Error prints an error message.
func Error(format string, a ...any) {
	prefix := color.HiRedString("[-] ")
	fmt.Fprintf(os.Stderr, prefix+format+"\n", a...)
}

// Verbose prints detailed progress messages (hidden in silent mode).
func Verbose(format string, a ...any) {
	if silentMode {
		return
	}
	prefix := color.HiCyanString("[verbose] ")
	fmt.Fprintf(os.Stderr, prefix+format+"\n", a...)
}

// Debug prints only when debug mode is on.
func Debug(format string, a ...any) {
	if !debugEnabled {
		return
	}
	prefix := color.HiMagentaString("[debug] ")
	fmt.Fprintf(os.Stderr, prefix+format+"\n", a...)
}
