package cli

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/projectdiscovery/cdncheck"
	"github.com/spf13/cobra"
)

func init() {
	cdnCmd.Flags().BoolVar(&opt.CDN.StripCDN, "strip-cdn", false, "Only output non-CDN IPs (remove CDN/WAF IPs from output)")
	rootCmd.AddCommand(cdnCmd)
}

var cdnCmd = &cobra.Command{
	Use:   "cdn",
	Short: "Check if IPs belong to a CDN or WAF provider",
	Long:  cdnLong,
	// Example is set in helptext.go init()
	Run: runCDN,
}

func runCDN(_ *cobra.Command, args []string) {
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

	cdnClient := cdncheck.New()
	if cdnClient == nil {
		output.Error("Failed to initialize CDN check client")
		return
	}

	output.Info("Checking %d target(s) for CDN/WAF", len(inputs))

	runner.RunParallel(inputs, opt.Concurrency, func(input string) {
		ip := input

		// handle http(s) URLs
		if strings.HasPrefix(ip, "http") {
			u, err := url.Parse(ip)
			if err != nil {
				output.Debug("Failed to parse URL: %s", ip)
				return
			}
			ip = u.Hostname()
		}

		parsed := net.ParseIP(ip)
		if parsed == nil {
			output.Debug("Invalid IP: %s", ip)
			return
		}

		matched, vendor, ipType, err := cdnClient.Check(parsed)
		if err != nil {
			output.Debug("CDN check error for %s: %v", ip, err)
			return
		}

		if vendor == "" {
			vendor = "unknown"
		}
		if ipType == "" {
			ipType = "none"
		}

		isCDN := matched && (ipType == "cdn" || ipType == "waf")

		if opt.CDN.StripCDN && isCDN {
			output.Verbose("Stripping CDN/WAF IP: %s (%s/%s)", ip, vendor, ipType)
			return
		}

		if opt.JSONOutput {
			w.WriteJSON(cdnResult{
				IP:     ip,
				IsCDN:  isCDN,
				Vendor: vendor,
				Type:   ipType,
			})
		} else {
			// Default: show IP with vendor and type (verbose is now default)
			w.WriteString(fmt.Sprintf("%s | %s | %s", ip, vendor, ipType))
		}
	})
}

type cdnResult struct {
	IP     string `json:"ip"`
	IsCDN  bool   `json:"is_cdn"`
	Vendor string `json:"vendor"`
	Type   string `json:"type"`
}
