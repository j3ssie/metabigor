package cli

import (
	"fmt"
	"strings"

	"github.com/j3ssie/metabigor/internal/asndb"
	"github.com/j3ssie/metabigor/internal/countrydb"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/spf13/cobra"
)

const updateLong = `Refresh the local IP-to-ASN and IP-to-country databases used by net and cluster.

A copy ships with the binary and is unpacked automatically on first use, so this
is only needed to pick up newer routing data. Both databases live in
~/.metabigor and are sourced from iplocate/ip-address-databases.`

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Refresh the local ASN and country databases",
	Long:    updateLong,
	GroupID: groupUtil,
	Example: examples(
		"refresh both databases", "metabigor update",
		"refresh quietly, for a cron job", "metabigor update -q",
	),
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(_ *cobra.Command, _ []string) error {
	var failed []string

	output.Info("Updating the ASN database ...")
	if err := asndb.Download(); err != nil {
		output.Error("ASN database update failed: %v", err)
		failed = append(failed, "ASN")
	}

	output.Info("Updating the country database ...")
	if err := countrydb.Download(); err != nil {
		output.Error("Country database update failed: %v", err)
		failed = append(failed, "country")
	}

	if len(failed) > 0 {
		return fmt.Errorf("could not update the %s database(s)", strings.Join(failed, " and "))
	}
	output.Good("Databases are up to date")
	return nil
}
