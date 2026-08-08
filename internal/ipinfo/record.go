package ipinfo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/j3ssie/metabigor/internal/output"
)

// Text renders one summary line per IP: address, open ports, then whatever
// hostname, vulnerability, and tag data InternetDB returned.
func (r InternetDBResult) Text() []string {
	return []string{output.Columns(
		r.IP,
		joinPorts(r.Ports, ","),
		strings.Join(r.Hostnames, ","),
		strings.Join(r.Vulns, ","),
		strings.Join(r.Tags, ","),
	)}
}

// Flat renders one IP:PORT per open port, ready to pipe into a scanner.
func (r InternetDBResult) Flat() []string {
	lines := make([]string, 0, len(r.Ports))
	for _, port := range r.Ports {
		lines = append(lines, fmt.Sprintf("%s:%d", r.IP, port))
	}
	return lines
}

// CSV renders every InternetDB field, with multi-values joined by semicolons.
func (r InternetDBResult) CSV() ([]string, [][]string) {
	return []string{"ip", "ports", "hostnames", "cpes", "vulns", "tags"},
		[][]string{{
			r.IP,
			joinPorts(r.Ports, ";"),
			strings.Join(r.Hostnames, ";"),
			strings.Join(r.CPEs, ";"),
			strings.Join(r.Vulns, ";"),
			strings.Join(r.Tags, ";"),
		}}
}

// HasFindings reports whether InternetDB knows anything about this IP.
func (r InternetDBResult) HasFindings() bool {
	return len(r.Ports) > 0 || len(r.Hostnames) > 0 || len(r.Vulns) > 0
}

// Text renders one line per ASN cluster, ordered by how many IPs landed in it.
func (c IPCluster) Text() []string {
	return []string{output.Columns(
		c.ASN,
		c.CIDR,
		fmt.Sprintf("%d IPs", c.Count),
		c.Description,
		c.CountryCode,
	)}
}

// Flat renders the cluster's CIDR, falling back to the ASN when unknown.
func (c IPCluster) Flat() []string {
	if c.CIDR != "" {
		return []string{c.CIDR}
	}
	return []string{c.ASN}
}

// CSV renders the cluster with its member IPs joined by semicolons.
func (c IPCluster) CSV() ([]string, [][]string) {
	return []string{"asn", "cidr", "count", "org", "country", "ips"},
		[][]string{{
			c.ASN,
			c.CIDR,
			strconv.Itoa(c.Count),
			c.Description,
			c.CountryCode,
			strings.Join(c.IPs, ";"),
		}}
}

func joinPorts(ports []int, sep string) string {
	parts := make([]string, len(ports))
	for i, p := range ports {
		parts[i] = strconv.Itoa(p)
	}
	return strings.Join(parts, sep)
}
