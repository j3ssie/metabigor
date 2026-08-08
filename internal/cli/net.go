package cli

import (
	"sync"

	"github.com/j3ssie/metabigor/internal/asndb"
	"github.com/j3ssie/metabigor/internal/countrydb"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/j3ssie/metabigor/internal/netdiscovery"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/internal/runner"
	"github.com/spf13/cobra"
)

const netLong = `Discover the network ranges (CIDRs) behind an ASN, IP, domain, or organization.

The target type is detected automatically; override it with --asn, --ip,
--domain, or --org when a name is ambiguous.

Lookups run against the bundled ASN database, which is fast and works offline.
Use --live to query bgp.he.net and friends instead, which is slower but current.`

var netCmd = &cobra.Command{
	Use:         "net [target...]",
	Short:       "Find network ranges (CIDRs) for an ASN, IP, domain, or org",
	Long:        netLong,
	GroupID:     groupRecon,
	Annotations: map[string]string{"sample": "AS13335"},
	Example: examples(
		"find every CIDR announced by an ASN", "metabigor net AS13335",
		"pass the target with a flag instead", "metabigor net -i AS13335 -f json",
		"several ASNs at once", "metabigor net AS13335 AS15169",
		"see which ASN owns an IP", "metabigor net 1.1.1.1 --detail",
		"look up a company by name", "metabigor net --org Cloudflare",
		"query live sources instead of the local database", "metabigor net --live tesla.com",
		"run a batch from a file", "metabigor net -I asn-list.txt -f csv -o ranges.csv",
		"or read the batch from stdin", "cat asn-list.txt | metabigor net",
		"feed the ranges into an IP scan", "metabigor net AS13335 -f flat | metabigor ip",
	),
	RunE: runNet,
}

func init() {
	f := netCmd.Flags()
	f.BoolVar(&opt.Net.ASN, "asn", false, "Treat the target as an ASN")
	f.BoolVar(&opt.Net.IP, "ip", false, "Treat the target as an IP or CIDR")
	f.BoolVar(&opt.Net.Domain, "domain", false, "Treat the target as a domain")
	f.BoolVar(&opt.Net.Org, "org", false, "Treat the target as an organization name")
	f.BoolVar(&opt.Net.Live, "live", false, "Query live online sources instead of the local database")
	f.BoolVarP(&opt.Net.Detail, "detail", "d", false, "Add ASN, organization, and country columns")

	netCmd.MarkFlagsMutuallyExclusive("asn", "ip", "domain", "org")
	rootCmd.AddCommand(netCmd)
}

func runNet(cmd *cobra.Command, _ []string) error {
	rc, err := setup(cmd)
	if err != nil {
		return err
	}
	defer rc.Close()

	forced := forcedNetType()
	if forced != netdiscovery.TypeUnknown {
		output.Verbose("Treating every target as %s", forced)
	}

	if opt.Net.Live {
		output.Info("Looking up %d target(s) via live sources", len(rc.Inputs))
		client := httpclient.NewClient(opt.Timeout, opt.Retry, opt.Proxy)

		runner.RunParallel(rc.Inputs, opt.Concurrency, func(target string) {
			for _, r := range netdiscovery.LiveLookup(client, target, forced, opt.Timeout) {
				r.Detail = opt.Net.Detail
				rc.Writer.Write(r)
			}
		})
		return nil
	}

	output.Info("Looking up %d target(s) in the local ASN database", len(rc.Inputs))
	db, err := asndb.EnsureLoaded()
	if err != nil {
		return err
	}
	country := lazyCountryLookup()

	runner.RunParallel(rc.Inputs, opt.Concurrency, func(target string) {
		for _, r := range netdiscovery.StaticLookup(db, target, forced, country) {
			r.Detail = opt.Net.Detail
			rc.Writer.Write(r)
		}
	})
	return nil
}

// forcedNetType maps the --asn/--ip/--domain/--org overrides onto an input
// type. Cobra guarantees at most one is set.
func forcedNetType() netdiscovery.InputType {
	switch {
	case opt.Net.ASN:
		return netdiscovery.TypeASN
	case opt.Net.IP:
		return netdiscovery.TypeIP
	case opt.Net.Domain:
		return netdiscovery.TypeDomain
	case opt.Net.Org:
		return netdiscovery.TypeOrg
	default:
		return netdiscovery.TypeUnknown
	}
}

// lazyCountryLookup resolves a country code for CIDRs whose ASN record has
// none. The country database holds two million rows, so it is loaded on first
// use rather than on every run.
func lazyCountryLookup() netdiscovery.CountryLookup {
	var (
		once sync.Once
		db   *countrydb.DB
	)
	return func(cidr string) string {
		once.Do(func() {
			loaded, err := countrydb.EnsureLoaded()
			if err != nil {
				output.Warn("Country database unavailable, leaving country blank: %v", err)
				return
			}
			db = loaded
		})
		if db == nil {
			return ""
		}
		if rec := db.LookupCIDR(cidr); rec != nil {
			return rec.CountryCode
		}
		return ""
	}
}
