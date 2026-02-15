// Package main provides the entry point for the metabigor CLI application.
package main

import "github.com/j3ssie/metabigor/internal/cli"

var (
	version   = "v2.1.0"
	commit    = "dev"
	buildDate = "unknown"
)

func main() {
	cli.SetVersion(version, commit, buildDate)
	cli.Execute()
}
