package cli

import (
	"fmt"

	"github.com/j3ssie/metabigor/internal/core"
	"github.com/spf13/cobra"
)

var (
	appCommit    = "none"
	appBuildDate = "unknown"
)

// SetVersion overrides the build metadata baked in via ldflags.
func SetVersion(version, commit, buildDate string) {
	if version != "" {
		rootCmd.Version = version
	}
	if commit != "" {
		appCommit = commit
	}
	if buildDate != "" {
		appBuildDate = buildDate
	}
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print version and build information",
	Long:    "Print the version, build date, commit hash, and author.",
	GroupID: groupUtil,
	Args:    cobra.NoArgs,
	Example: examples(
		"show full build information", "metabigor version",
		"show just the version", "metabigor --version",
	),
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("%s - %s\n", core.NAME, core.DESC)
		fmt.Printf("Version: %s\n", rootCmd.Version)
		fmt.Printf("Build:   %s\n", appBuildDate)
		fmt.Printf("Commit:  %s\n", appCommit)
		fmt.Printf("Author:  %s\n", core.AUTHOR)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
