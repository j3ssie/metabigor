// Package asndb provides ASN (Autonomous System Number) database functionality for IP-to-ASN lookups.
package asndb

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

// DB is an in-memory ASN database sorted by StartIP.
type DB struct {
	records []ASNRecord
}

// Load parses the CSV file into memory.
// Expected format: network,asn,country_code,name,org,domain
func Load(path string) (*DB, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w (run 'metabigor update' to download)", err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	var records []ASNRecord
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
		if len(row) < 4 {
			continue
		}

		cidr := strings.TrimSpace(row[0])
		startIP, endIP := cidrRange(cidr)
		if startIP == nil || endIP == nil {
			continue
		}

		rec := ASNRecord{
			CIDR:    cidr,
			StartIP: startIP,
			EndIP:   endIP,
			ASN:     strings.TrimSpace(row[1]),
		}
		if len(row) > 2 {
			rec.CountryCode = strings.TrimSpace(row[2])
		}
		if len(row) > 3 {
			rec.Name = strings.TrimSpace(row[3])
		}
		if len(row) > 4 {
			rec.Org = strings.TrimSpace(row[4])
		}
		if len(row) > 5 {
			rec.Domain = strings.TrimSpace(row[5])
		}
		records = append(records, rec)
	}

	// Sort by StartIP for binary search
	sort.Slice(records, func(i, j int) bool {
		return bytes.Compare(records[i].StartIP.To16(), records[j].StartIP.To16()) < 0
	})

	output.Verbose("Loaded %d ASN records", len(records))
	return &DB{records: records}, nil
}

// LookupIP finds the ASN record for a given IP address.
func (db *DB) LookupIP(ipStr string) *ASNRecord {
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

// LookupASN returns all records matching the given ASN number.
func (db *DB) LookupASN(asn string) []ASNRecord {
	asn = strings.TrimPrefix(strings.ToUpper(asn), "AS")
	var results []ASNRecord
	for _, rec := range db.records {
		if rec.ASN == asn {
			results = append(results, rec)
		}
	}
	return results
}

// SearchOrg returns records whose name/org contains the query (case-insensitive).
func (db *DB) SearchOrg(query string) []ASNRecord {
	query = strings.ToLower(query)
	var results []ASNRecord
	seen := make(map[string]bool)
	for _, rec := range db.records {
		match := strings.Contains(strings.ToLower(rec.Name), query) ||
			strings.Contains(strings.ToLower(rec.Org), query)
		if match {
			key := rec.ASN + "|" + rec.CIDR
			if !seen[key] {
				seen[key] = true
				results = append(results, rec)
			}
		}
	}
	return results
}

// LookupDomain resolves a domain's IPs and looks up their ASN records.
func (db *DB) LookupDomain(domain string) []ASNRecord {
	ips, err := net.LookupHost(domain)
	if err != nil {
		return nil
	}

	var results []ASNRecord
	seen := make(map[string]bool)
	for _, ipStr := range ips {
		rec := db.LookupIP(ipStr)
		if rec != nil && !seen[rec.ASN] {
			seen[rec.ASN] = true
			results = append(results, *rec)
			// Also include all CIDRs for this ASN
			for _, r := range db.LookupASN(rec.ASN) {
				key := r.ASN + "|" + r.CIDR
				if !seen[key] {
					seen[key] = true
					results = append(results, r)
				}
			}
		}
	}
	return results
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
