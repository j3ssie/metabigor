// Package main provides the entry point for the metabigor CLI application.
package main

import (
	"github.com/j3ssie/metabigor/internal/cli"
	"github.com/j3ssie/metabigor/internal/core"
)

var (
	version   = core.VERSION
	commit    = "dev"
	buildDate = "unknown"
)

func main() {
	cli.SetVersion(version, commit, buildDate)
	cli.Execute()
}
