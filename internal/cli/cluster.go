package cli

import (
	"github.com/j3ssie/metabigor/internal/asndb"
	"github.com/j3ssie/metabigor/internal/ipinfo"
	"github.com/j3ssie/metabigor/internal/output"
	"github.com/spf13/cobra"
)

const clusterLong = `Group a list of IPs by the ASN that owns them, using the local database.

Clusters are sorted by size, so the hosting providers and cloud regions a target
leans on the most appear first. Works offline.`

var clusterCmd = &cobra.Command{
	Use:         "cluster [ip...]",
	Short:       "Group IPs by owning ASN to map infrastructure",
	Long:        clusterLong,
	GroupID:     groupEnrich,
	Annotations: map[string]string{"sample": "1.1.1.1"},
	Example: examples(
		"cluster a few IPs given as arguments", "metabigor cluster 1.1.1.1 8.8.8.8 9.9.9.9",
		"pass a single IP with a flag", "metabigor cluster -i 1.1.1.1",
		"cluster a list from a file", "metabigor cluster -I ips.txt",
		"or read the list from stdin", "cat ips.txt | metabigor cluster",
		"cluster with full ASN detail as JSON", "metabigor cluster -I ips.txt -f json",
		"export the clusters to a spreadsheet", "metabigor cluster -I ips.txt -f csv -o clusters.csv",
		"resolve domains, then cluster the results", "cat domains.txt | dnsx -silent -resp-only | metabigor cluster",
	),
	RunE: runCluster,
}

func init() {
	rootCmd.AddCommand(clusterCmd)
}

func runCluster(cmd *cobra.Command, _ []string) error {
	rc, err := setup(cmd)
	if err != nil {
		return err
	}
	defer rc.Close()

	output.Info("Clustering %d IP(s) by ASN", len(rc.Inputs))
	db, err := asndb.EnsureLoaded()
	if err != nil {
		return err
	}

	clusters := ipinfo.ClusterIPs(db, rc.Inputs)
	output.Verbose("Grouped %d IP(s) into %d cluster(s)", len(rc.Inputs), len(clusters))
	for _, c := range clusters {
		rc.Writer.Write(c)
	}
	return nil
}
