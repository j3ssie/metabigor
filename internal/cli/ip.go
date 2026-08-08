package cli

import (
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/ipinfo"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

// largeScanThreshold is the number of expanded hosts above which a CIDR scan is
// worth warning about, since each host costs one InternetDB request.
const largeScanThreshold = 4096

const ipLong = `Look up open ports, hostnames, CVEs, and tags for an IP using Shodan's free
InternetDB API. No API key required.

CIDRs are expanded to their individual hosts, so one request is made per host.
IPs that InternetDB knows nothing about are skipped unless --all is set.`

var ipCmd = &cobra.Command{
	Use:         "ip [target...]",
	Short:       "Enrich IPs with ports, hostnames, and CVEs (Shodan InternetDB)",
	Long:        ipLong,
	GroupID:     groupEnrich,
	Annotations: map[string]string{"sample": "1.1.1.1"},
	Example: examples(
		"enrich a single IP", "metabigor ip 1.1.1.1",
		"pass the target with a flag instead", "metabigor ip -i 1.1.1.1 -f json",
		"scan a whole range", "metabigor ip 1.1.1.0/28",
		"emit IP:PORT pairs for a scanner", "metabigor ip 1.1.1.0/28 -f flat",
		"bulk scan a file with more workers", "metabigor ip -I ips.txt -c 30 -f flat -o ports.txt",
		"export findings to a spreadsheet", "metabigor ip -I ips.txt -f csv -o ports.csv",
		"or read the batch from stdin", "cat ips.txt | metabigor ip",
		"chain from network discovery", "metabigor net AS13335 -f flat | metabigor ip -f flat",
	),
	RunE: runIP,
}

func init() {
	f := ipCmd.Flags()
	f.BoolVar(&opt.IP.All, "all", false, "Include IPs that InternetDB has no data for")
	rootCmd.AddCommand(ipCmd)
}

func runIP(cmd *cobra.Command, _ []string) error {
	rc, err := setup(cmd)
	if err != nil {
		return err
	}
	defer rc.Close()

	targets := expandTargets(rc.Inputs)
	if len(targets) > largeScanThreshold {
		output.Warn("Expanded to %d hosts — this makes one request per host and will take a while", len(targets))
	}

	output.Info("Querying Shodan InternetDB for %d host(s)", len(targets))
	client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

	runner.RunParallel(targets, opt.Concurrency, func(ip string) {
		result, err := ipinfo.LookupInternetDB(client, ip)
		if err != nil {
			output.Debug("InternetDB lookup failed for %s: %v", ip, err)
			return
		}
		if !result.HasFindings() && !opt.IP.All {
			output.Debug("InternetDB has nothing for %s, skipping", ip)
			return
		}
		output.Verbose("InternetDB: %s has %d port(s) and %d hostname(s)", ip, len(result.Ports), len(result.Hostnames))
		rc.Writer.Write(*result)
	})
	return nil
}

// expandTargets turns any CIDR inputs into their individual host addresses.
func expandTargets(inputs []string) []string {
	var targets []string
	for _, in := range inputs {
		if !ipinfo.IsCIDR(in) {
			targets = append(targets, in)
			continue
		}
		hosts := ipinfo.ExpandCIDR(in)
		output.Verbose("Expanded %s to %d host(s)", in, len(hosts))
		targets = append(targets, hosts...)
	}
	return targets
}
