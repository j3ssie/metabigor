package countrydb

import "net"

// CountryRecord represents a single row from the ip-to-country CSV.
// CSV format: network,country_code,country_name
type CountryRecord struct {
	CIDR        string // e.g. "1.0.0.0/24"
	StartIP     net.IP // first IP of CIDR
	EndIP       net.IP // last IP of CIDR
	CountryCode string // e.g. "US"
	CountryName string // e.g. "United States"
}
