package cli

import (
	"fmt"

	"github.com/j3ssie/metabigor/internal/asndb"
	"github.com/j3ssie/metabigor/internal/ipinfo"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(ipcCmd)
}

var ipcCmd = &cobra.Command{
	Use:   "ipc",
	Short: "Cluster IPs by ASN using the local database",
	Long:  ipcLong,
	// Example is set in helptext.go init()
	Run: runIPC,
}

func runIPC(_ *cobra.Command, args []string) {
	output.SetupLogger(opt.Silent, opt.Debug, opt.NoColor)
	inputs := runner.ReadInputs(opt.Input, opt.InputFile, args)
	if len(inputs) == 0 {
		output.Error("No input provided")
		return
	}

	w, err := output.NewWriter(opt.Output, opt.JSONOutput)
	if err != nil {
		output.Error("%v", err)
		return
	}
	defer w.Close()

	output.Info("Clustering %d IP(s) by ASN", len(inputs))
	db, err := asndb.EnsureLoaded()
	if err != nil {
		output.Error("%v", err)
		return
	}

	clusters := ipinfo.ClusterIPs(db, inputs)
	output.Verbose("Found %d ASN cluster(s)", len(clusters))
	for _, c := range clusters {
		if opt.JSONOutput {
			w.WriteJSON(c)
		} else {
			w.WriteString(fmt.Sprintf("%s | %s | %d IPs | %s | %s", c.ASN, c.CIDR, c.Count, c.Description, c.CountryCode))
		}
	}
}
