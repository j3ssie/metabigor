package ipinfo

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/httpclient"
	"github.com/projectdiscovery/mapcidr"
)

// InternetDBResult represents a Shodan InternetDB response.
type InternetDBResult struct {
	IP        string   `json:"ip"`
	Ports     []int    `json:"ports"`
	Hostnames []string `json:"hostnames"`
	CPEs      []string `json:"cpes"`
	Vulns     []string `json:"vulns"`
	Tags      []string `json:"tags"`
}

// LookupInternetDB queries Shodan's InternetDB for a single IP.
func LookupInternetDB(client *retryablehttp.Client, ip string) (*InternetDBResult, error) {
	url := fmt.Sprintf("https://internetdb.shodan.io/%s", ip)
	body, err := httpclient.Get(client, url)
	if err != nil {
		return nil, err
	}

	var result InternetDBResult
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}

// ExpandCIDR returns all IPs in a CIDR range.
func ExpandCIDR(cidr string) []string {
	// Try mapcidr first
	ips, err := mapcidr.IPAddressesAsStream(cidr)
	if err == nil {
		var result []string
		for ip := range ips {
			result = append(result, ip)
		}
		return result
	}

	// Fallback to manual expansion
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return []string{cidr}
	}
	var result []string
	for ip := ip.Mask(ipNet.Mask); ipNet.Contains(ip); incIP(ip) {
		result = append(result, ip.String())
	}
	return result
}

// FormatFlat formats a result as IP:PORT lines.
func FormatFlat(r *InternetDBResult) []string {
	var lines []string
	for _, port := range r.Ports {
		lines = append(lines, fmt.Sprintf("%s:%d", r.IP, port))
	}
	return lines
}

// FormatCSV formats a result as a CSV line.
func FormatCSV(r *InternetDBResult) string {
	ports := make([]string, len(r.Ports))
	for i, p := range r.Ports {
		ports[i] = fmt.Sprintf("%d", p)
	}
	return fmt.Sprintf("%s,%s,%s,%s,%s",
		r.IP,
		strings.Join(ports, ";"),
		strings.Join(r.Hostnames, ";"),
		strings.Join(r.Vulns, ";"),
		strings.Join(r.Tags, ";"),
	)
}

// IsCIDR checks whether the input is a CIDR notation.
func IsCIDR(input string) bool {
	_, _, err := net.ParseCIDR(input)
	return err == nil
}

// IsIP checks if the input is a valid IP.
func IsIP(input string) bool {
	return net.ParseIP(input) != nil
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
