// Package ipinfo provides IP clustering and organization using Shodan InternetDB.
package ipinfo

import (
	"fmt"
	"sort"

	"github.com/j3ssie/metabigor/internal/asndb"
)

// IPCluster represents a group of IPs sharing the same ASN.
type IPCluster struct {
	ASN         string `json:"asn"`
	CIDR        string `json:"cidr"`
	Count       int    `json:"count"`
	Description string `json:"description"`
	CountryCode string `json:"country_code"`
	IPs         []string `json:"ips,omitempty"`
}

// ClusterIPs groups a list of IPs by their ASN using the local database.
func ClusterIPs(db *asndb.DB, ips []string) []IPCluster {
	clusters := make(map[string]*IPCluster)

	for _, ip := range ips {
		rec := db.LookupIP(ip)
		if rec == nil {
			key := "unknown"
			if c, ok := clusters[key]; ok {
				c.Count++
				c.IPs = append(c.IPs, ip)
			} else {
				clusters[key] = &IPCluster{
					ASN:         "unknown",
					Description: "unknown",
					Count:       1,
					IPs:         []string{ip},
				}
			}
			continue
		}

		key := fmt.Sprintf("AS%s|%s", rec.ASN, rec.CIDR)
		if c, ok := clusters[key]; ok {
			c.Count++
			c.IPs = append(c.IPs, ip)
		} else {
			clusters[key] = &IPCluster{
				ASN:         fmt.Sprintf("AS%s", rec.ASN),
				CIDR:        rec.CIDR,
				Count:       1,
				Description: rec.Description(),
				CountryCode: rec.CountryCode,
				IPs:         []string{ip},
			}
		}
	}

	// Sort by count descending
	var result []IPCluster
	for _, c := range clusters {
		result = append(result, *c)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	return result
}
