package cli

import (
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/ipinfo"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

func init() {
	ipCmd.Flags().BoolVar(&opt.IP.Flat, "flat", false, "Output as IP:PORT lines")
	ipCmd.Flags().BoolVar(&opt.IP.CSV, "csv", false, "Output as CSV")
	rootCmd.AddCommand(ipCmd)
}

var ipCmd = &cobra.Command{
	Use:   "ip",
	Short: "Query Shodan InternetDB for IP information",
	Long:  ipLong,
	// Example is set in helptext.go init()
	Run: runIP,
}

func runIP(_ *cobra.Command, args []string) {
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

	// Expand CIDRs
	var targets []string
	for _, in := range inputs {
		if ipinfo.IsCIDR(in) {
			expanded := ipinfo.ExpandCIDR(in)
			output.Verbose("Expanded CIDR %s to %d IPs", in, len(expanded))
			targets = append(targets, expanded...)
		} else {
			targets = append(targets, in)
		}
	}

	output.Info("Querying Shodan InternetDB for %d target(s)", len(targets))
	client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

	runner.RunParallel(targets, opt.Concurrency, func(ip string) {
		output.Debug("Looking up %s on InternetDB", ip)
		result, err := ipinfo.LookupInternetDB(client, ip)
		if err != nil {
			output.Debug("InternetDB error for %s: %v", ip, err)
			return
		}
		if len(result.Ports) == 0 && len(result.Hostnames) == 0 {
			output.Debug("InternetDB: %s has no open ports or hostnames, skipping", ip)
			return
		}

		output.Verbose("InternetDB: %s has %d ports, %d hostnames", ip, len(result.Ports), len(result.Hostnames))

		switch {
		case opt.JSONOutput:
			w.WriteJSON(result)
		case opt.IP.Flat:
			for _, line := range ipinfo.FormatFlat(result) {
				w.WriteString(line)
			}
		case opt.IP.CSV:
			w.WriteString(ipinfo.FormatCSV(result))
		default:
			w.WriteJSON(result)
		}
	})
}
