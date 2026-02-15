package asndb

import "net"

// ASNRecord represents a single row from the ip-to-asn CSV.
// CSV format: network,asn,country_code,name,org,domain
type ASNRecord struct {
	CIDR        string // e.g. "1.0.0.0/24"
	StartIP     net.IP // first IP of CIDR
	EndIP       net.IP // last IP of CIDR
	ASN         string // e.g. "13335"
	CountryCode string
	Name        string // short name e.g. "CLOUDFLARENET"
	Org         string // full org e.g. "Cloudflare, Inc."
	Domain      string // e.g. "cloudflare.com"
}

// Description returns the best human-readable description.
func (r *ASNRecord) Description() string {
	if r.Org != "" {
		return r.Org
	}
	return r.Name
}
