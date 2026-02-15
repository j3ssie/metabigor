package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	// AppName is the name of the application
	AppName = "metabigor"
	// AppVersion is the current version
	AppVersion = "v2.1.0"
	// Author is the author of the application
	Author = "@j3ssie"
)

var (
	appVersion   = AppVersion
	appCommit    = "none"
	appBuildDate = "unknown"
)

// SetVersion sets the version info from ldflags.
func SetVersion(version, commit, buildDate string) {
	if version != "" {
		appVersion = version
	}
	if commit != "" {
		appCommit = commit
	}
	if buildDate != "" {
		appBuildDate = buildDate
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  versionLong,
	// Example is set in helptext.go init()
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("%s - OSINT power without API key hassle\n", AppName)
		fmt.Printf("Version: %s\n", appVersion)
		fmt.Printf("Build: %s\n", appBuildDate)
		fmt.Printf("Commit: %s\n", appCommit)
		fmt.Printf("Author: %s\n", Author)
	},
}
