package modules

import (
	"bufio"
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"math/big"
	"net"
	"sort"
	"strconv"
	"strings"

	"net/netip"
)

// Most of the file literally copied from @thebl4ckturtle code

type ASInfo struct {
	Amount      int
	Number      int
	CountryCode string
	Description string
	CIDR        string
}

//go:embed static/ip2asn-combined.tsv.gz
var ip2asnData string

func GetAsnMap() (AsnMap, error) {
	//var asnAsnMap AsnMap
	//f, err := pkger.Open("/static/ip2asn-combined.tsv.gz")
	//if err != nil {
	//	return AsnMap{}, err
	//}
	f := strings.NewReader(ip2asnData)
	asnAsnMap, err := GenAsnData(f)
	if err != nil {
		return AsnMap{}, err
	}
	return *asnAsnMap, nil
}

type IPInfo struct {
	AS   int
	CIDR string
}

type AsnMap struct {
	asName    map[int]string
	asCountry map[int]string
	asDesc    map[string]int
	recs      []rec
}

type rec struct {
	startIP, endIP netip.Addr
	asn            int
}

func (m *AsnMap) ASName(as int) string    { return m.asName[as] }
func (m *AsnMap) ASCountry(as int) string { return m.asCountry[as] }
func (m *AsnMap) ASDesc(name string) (AsnNum []int) {
	name = strings.ToLower(name)
	for k, v := range m.asDesc {
		if strings.Contains(strings.ToLower(k), name) {
			AsnNum = append(AsnNum, v)
		}
	}
	return AsnNum
}

// ASInfo returns 0 on unknown.
func (m *AsnMap) ASInfo(asnNum int) []ASInfo {
	var asnInfos []ASInfo
	if asnNum == 0 {
		return asnInfos
	}

	for _, rec := range m.recs {
		if rec.asn == asnNum {
			var asnInfo ASInfo
			as := IPInfo{AS: rec.asn, CIDR: Range2CIDR(net.ParseIP(rec.startIP.String()), net.ParseIP(rec.endIP.String())).String()}
			asnInfo.Number = asnNum
			asnInfo.CIDR = as.CIDR
			asnInfo.CountryCode = m.ASCountry(asnNum)
			asnInfo.Description = m.ASName(asnNum)
			asnInfos = append(asnInfos, asnInfo)
		}
	}
	return asnInfos
}

// ASofIP returns 0 on unknown.
func (m *AsnMap) ASofIP(ip netip.Addr) IPInfo {
	cand := sort.Search(len(m.recs), func(i int) bool {
		return ip.Less(m.recs[i].startIP)
	})
	return m.recIndexHasIP(cand-1, ip)
}

// recIndexHasIP returns the AS number of m.rec[i] if i is in range and
// the record contains the given IP address.
func (m *AsnMap) recIndexHasIP(i int, ip netip.Addr) (as IPInfo) {
	if i < 0 {
		return IPInfo{AS: 0}
	}
	rec := &m.recs[i]
	if rec.endIP.Less(ip) {
		return IPInfo{AS: 0}
	}
	if ip.Less(rec.startIP) {
		return IPInfo{AS: 0}
	}
	return IPInfo{AS: rec.asn, CIDR: Range2CIDR(net.ParseIP(rec.startIP.String()), net.ParseIP(rec.endIP.String())).String()}
}

func GenAsnData(r io.Reader) (*AsnMap, error) {
	br := bufio.NewReader(r)
	magic, err := br.Peek(2)
	if err != nil {
		return nil, err
	}
	if string(magic) == "\x1f\x8b" {
		zr, err := gzip.NewReader(br)
		if err != nil {
			return nil, err
		}
		br = bufio.NewReader(zr)
	}
	m := &AsnMap{
		asName:    map[int]string{},
		asCountry: map[int]string{},
		asDesc:    map[string]int{},
	}
	for {
		line, err := br.ReadSlice('\n')
		if err == io.EOF {
			return m, nil
		}
		if err != nil {
			return nil, err
		}
		var startIPB, endIPB, asnB, country, desc []byte
		if err := parseTSV(line, &startIPB, &endIPB, &asnB, &country, &desc); err != nil {
			return nil, err
		}
		if string(desc) == "Not routed" {
			continue
		}
		as64, err := strconv.ParseInt(string(asnB), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bogus ASN %q for line %q", asnB, line)
		}
		as := int(as64)
		if _, ok := m.asName[as]; !ok {
			m.asName[as] = string(desc)
			m.asCountry[as] = string(country)
			m.asDesc[string(desc)] = as
		}

		startIP, err := netip.ParseAddr(string(startIPB)) // TODO: add ParseIPBytes
		if err != nil {
			return nil, fmt.Errorf("bogus IP %q for line %q", startIPB, line)
		}
		endIP, err := netip.ParseAddr(string(endIPB)) // TODO: add ParseIPBytes
		if err != nil {
			return nil, fmt.Errorf("bogus IP %q for line %q", endIPB, line)
		}
		m.recs = append(m.recs, rec{startIP, endIP, as})
	}
}

func parseTSV(line []byte, dsts ...*[]byte) error {
	if len(line) > 0 && line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	for i, dst := range dsts {
		last := i == len(dsts)-1
		tab := bytes.IndexByte(line, '\t')
		if tab == -1 && !last {
			return fmt.Errorf("short line: %q", line)
		}
		if tab != -1 {
			*dst, line = line[:tab], line[tab+1:]
		} else {
			*dst = line
		}
	}
	return nil
}

// Range2CIDR turns an IP range into a CIDR.
func Range2CIDR(first, last net.IP) *net.IPNet {
	startip, m := ipToInt(first)
	endip, _ := ipToInt(last)
	newip := big.NewInt(1)
	mask := big.NewInt(1)
	one := big.NewInt(1)

	if startip.Cmp(endip) == 1 {
		return nil
	}

	max := uint(m)
	var bits uint = 1
	newip.Set(startip)
	tmp := new(big.Int)
	for bits < max {
		tmp.Rsh(startip, bits)
		tmp.Lsh(tmp, bits)

		newip.Or(startip, mask)
		if newip.Cmp(endip) == 1 || tmp.Cmp(startip) != 0 {
			bits--
			mask.Rsh(mask, 1)
			break
		}

		bits++
		tmp.Lsh(mask, 1)
		mask.Add(tmp, one)
	}

	cidrstr := first.String() + "/" + strconv.Itoa(int(max-bits))
	_, ipnet, _ := net.ParseCIDR(cidrstr)

	return ipnet
}

func ipToInt(ip net.IP) (*big.Int, int) {
	val := big.NewInt(1)

	val.SetBytes([]byte(ip))
	if IsIPv4(ip) {
		return val, 32
	} else if IsIPv6(ip) {
		return val, 128
	}

	return val, 0
}

// IsIPv4 returns true when the provided net.IP address is an IPv4 address.
func IsIPv4(ip net.IP) bool {
	return strings.Count(ip.String(), ":") < 2
}

// IsIPv6 returns true when the provided net.IP address is an IPv6 address.
func IsIPv6(ip net.IP) bool {
	return strings.Count(ip.String(), ":") >= 2
}
