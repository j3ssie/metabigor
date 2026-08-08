package cli

import (
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/j3ssie/metabigor/internal/core"
)

// colorsDisabled reports whether help output should be plain. Help can render
// before Cobra parses flags, so --no-color is read straight from os.Args.
func colorsDisabled() bool {
	// color.NoColor already accounts for NO_COLOR, TERM=dumb, and a non-TTY.
	if color.NoColor {
		return true
	}
	for _, arg := range os.Args {
		if arg == "--no-color" {
			return true
		}
	}
	return false
}

// examples renders alternating comment/command pairs as an indented block:
//
//	# find CIDRs for an ASN
//	metabigor net AS13335
func examples(pairs ...string) string {
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("  " + dim("# "+pairs[i]) + "\n")
		b.WriteString("  " + highlight(pairs[i+1]) + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// highlight tints the binary name and any flags inside an example command.
func highlight(cmd string) string {
	if colorsDisabled() {
		return cmd
	}

	tokens := strings.Split(cmd, " ")
	for i, tok := range tokens {
		switch {
		case tok == core.NAME:
			tokens[i] = color.New(color.FgCyan, color.Bold).Sprint(tok)
		case strings.HasPrefix(tok, "-"):
			tokens[i] = color.New(color.FgYellow).Sprint(tok)
		}
	}
	return strings.Join(tokens, " ")
}

func dim(s string) string {
	if colorsDisabled() {
		return s
	}
	return color.New(color.FgHiBlack).Sprint(s)
}
