package cli

import (
	"github.com/j3ssie/metabigor/internal/asndb"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download/update the local ASN database",
	Long:  updateLong,
	// Example is set in helptext.go init()
	Run: func(_ *cobra.Command, _ []string) {
		output.SetupLogger(opt.Silent, opt.Debug, opt.NoColor)
		if err := asndb.Download(); err != nil {
			output.Error("Failed to update ASN database: %v", err)
			return
		}
		output.Good("ASN database updated successfully")
	},
}
