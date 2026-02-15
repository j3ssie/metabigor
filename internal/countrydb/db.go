// Package countrydb provides IP-to-country geolocation database functionality.
package countrydb

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"strings"

	"github.com/j3ssie/metabigor/internal/output"
)

// DB is an in-memory country database sorted by StartIP.
type DB struct {
	records []CountryRecord
}

// Load parses the CSV file into memory.
// Expected format: network,country_code,country_name
func Load(path string) (*DB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open country db: %w (run 'metabigor update' to download)", err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var records []CountryRecord
	firstRow := true
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		// Skip header
		if firstRow {
			firstRow = false
			if len(row) > 0 && row[0] == "network" {
				continue
			}
		}
		if len(row) < 3 {
			continue
		}

		cidr := strings.TrimSpace(row[0])
		startIP, endIP := cidrRange(cidr)
		if startIP == nil || endIP == nil {
			continue
		}

		rec := CountryRecord{
			CIDR:        cidr,
			StartIP:     startIP,
			EndIP:       endIP,
			CountryCode: strings.TrimSpace(row[1]),
			CountryName: strings.TrimSpace(row[2]),
		}
		records = append(records, rec)
	}

	// Sort by StartIP for binary search
	sort.Slice(records, func(i, j int) bool {
		return bytes.Compare(records[i].StartIP.To16(), records[j].StartIP.To16()) < 0
	})

	output.Verbose("Loaded %d country records", len(records))
	return &DB{records: records}, nil
}

// LookupIP finds the country record for a given IP address.
func (db *DB) LookupIP(ipStr string) *CountryRecord {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil
	}
	ip16 := ip.To16()

	idx := sort.Search(len(db.records), func(i int) bool {
		return bytes.Compare(db.records[i].StartIP.To16(), ip16) > 0
	})

	// The record we want is at idx-1 (the last one whose StartIP <= ip)
	if idx > 0 {
		rec := &db.records[idx-1]
		if bytes.Compare(ip16, rec.StartIP.To16()) >= 0 && bytes.Compare(ip16, rec.EndIP.To16()) <= 0 {
			return rec
		}
	}
	return nil
}

// LookupCIDR finds the country for a CIDR block by checking its first IP.
func (db *DB) LookupCIDR(cidr string) *CountryRecord {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil
	}
	return db.LookupIP(ipNet.IP.String())
}

// cidrRange returns the first and last IP of a CIDR block.
func cidrRange(cidr string) (net.IP, net.IP) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, nil
	}
	start := make(net.IP, len(ipNet.IP))
	copy(start, ipNet.IP)

	end := make(net.IP, len(ipNet.IP))
	for i := range ipNet.IP {
		end[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}
	return start, end
}
