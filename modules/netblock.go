package modules

import (
    "encoding/json"
    "fmt"
    jsoniter "github.com/json-iterator/go"
    "net"
    "strings"

    "github.com/PuerkitoBio/goquery"
    "github.com/j3ssie/metabigor/core"
    "github.com/thoas/go-funk"
)

func getAsnNum(raw string) string {
    if strings.HasPrefix(strings.ToLower(raw), "as") {
        return raw[2:]
    }
    return raw
}

// RangeInfo infor about range IP
type RangeInfo struct {
    Cidr    string `json:"cidr"`
    Desc    string `json:"desc"`
    Asn     string `json:"asn"`
    Country string `json:"country"`
}

// GetIPInfo get CIDR from ASN
func GetIPInfo(options core.Options) []string {
    asn := getAsnNum(options.Net.Asn)
    url := fmt.Sprintf(`https://ipinfo.io/AS%v`, asn)
    var result []string
    core.InforF("Get data from: %v", url)
    content := core.RequestWithChrome(url, "ipTabContent", options.Timeout)
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
    if err != nil {
        return result
    }

    var country string
    doc.Find(".flag").Each(func(i int, s *goquery.Selection) {
        href, ok := s.Attr("href")
        if ok {
            if strings.HasPrefix(href, "/countries/") {
                country = s.Text()
                return
            }
        }
    })

    // searching for data
    doc.Find("tr").Each(func(i int, s *goquery.Selection) {
        s.Find("address").First()
        if !strings.Contains(s.Text(), "Netblock") {
            data := strings.Split(strings.TrimSpace(s.Text()), "  ")
            cidr := strings.TrimSpace(data[0])
            desc := strings.TrimSpace(data[len(data)-1])
            if len(data) > 2 {
                desc = strings.TrimSpace(data[1])
            }

            if options.JsonOutput {
                output := RangeInfo{
                    Cidr:    cidr,
                    Desc:    desc,
                    Asn:     asn,
                    Country: country,
                }
                if out, err := jsoniter.MarshalToString(output); err == nil {
                    core.InforF(out)
                    result = append(result, out)
                }
            } else {
                core.InforF(fmt.Sprintf("%s - %s", cidr, desc))
                result = append(result, fmt.Sprintf("%s", cidr))
            }
        }
    })
    return result
}

// IPv4Info get CIDR from ASN via ipv4info.com
func IPv4Info(options core.Options) []string {
    asn := getAsnNum(options.Net.Asn)
    url := fmt.Sprintf(`http://ipv4info.com/?act=check&ip=AS%v`, asn)
    var result []string

    core.InforF("Get data from: %v", url)
    content := core.SendGET(url, options)
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
    if err != nil {
        return result
    }

    // finding ID of block
    var ASNLink []string
    doc.Find("a").Each(func(i int, s *goquery.Selection) {
        href, ok := s.Attr("href")
        if ok {
            if strings.HasPrefix(href, "/org/") {
                ASNLink = append(ASNLink, href)
            }
        }
    })

    // searching for data
    ASNLink = funk.Uniq(ASNLink).([]string)
    for _, link := range ASNLink {
        core.InforF("Get data from: %v", link)
        URL := fmt.Sprintf(`http://ipv4info.com%v`, link)
        core.InforF("Get data from: %v", URL)
        content := core.SendGET(url, options)
        // finding ID of block
        doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
        if err != nil {
            return result
        }

        doc.Find("td").Each(func(i int, s *goquery.Selection) {
            style, _ := s.Attr("style")
            class, _ := s.Attr("class")
            if style == "padding: 0 0 0 0;" && class == "bold" {
                data := s.Text()
                result = append(result, data)
            }

        })
    }
    core.InforF("\n%v", strings.Join(result, "\n"))
    return result
}

// ASNBgpDotNet get ASN infor from bgp.net
func ASNBgpDotNet(options core.Options) []string {
    asn := getAsnNum(options.Net.Asn)
    url := fmt.Sprintf(`https://bgp.he.net/AS%v#_prefixes`, asn)
    core.InforF("Get data from: %v", url)
    var result []string
    content := core.RequestWithChrome(url, "prefixes", options.Timeout*4)
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
    if err != nil {
        return result
    }
    // searching for data
    doc.Find("tr").Each(func(i int, s *goquery.Selection) {
        data := strings.Split(strings.TrimSpace(s.Text()), "  ")
        cidr := strings.TrimSpace(data[0])
        if !strings.Contains(cidr, "Prefix") {
            desc := strings.TrimSpace(data[len(data)-1])
            core.InforF(fmt.Sprintf("%s - %s", cidr, desc))
            result = append(result, fmt.Sprintf("%s", cidr))
        }
    })
    return result
}

// ASNSpyse get ASN infor from spyse.com
func ASNSpyse(options core.Options) []string {
    asn := getAsnNum(options.Net.Asn)
    url := fmt.Sprintf(`https://spyse.com/target/as/%v#c-domain__anchor--3--%v`, asn, asn)
    var result []string
    core.InforF("Get data from: %v", url)
    content := core.RequestWithChrome(url, "asn-ipv4-ranges", options.Timeout)
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
    if err != nil {
        return result
    }
    // searching for data
    doc.Find("tr").Each(func(i int, s *goquery.Selection) {
        data := strings.Split(strings.TrimSpace(s.Text()), "  ")
        cidr := strings.TrimSpace(data[0])
        if !strings.Contains(cidr, "CIDR") {
            desc := strings.Split(data[len(data)-2], "\n")
            realDesc := desc[len(desc)-1]
            core.InforF(fmt.Sprintf("%s - %s", cidr, realDesc))
            result = append(result, fmt.Sprintf("%s", cidr))
        }
    })
    return result
}

/* Get IP range from Organization */

// OrgBgpDotNet get Org infor from bgp.net
func OrgBgpDotNet(options core.Options) []string {
    org := options.Net.Org
    url := fmt.Sprintf(`https://bgp.he.net/search?search%%5Bsearch%%5D=%v&commit=Search`, org)
    core.InforF("Get data from: %v", url)
    var result []string
    content := core.RequestWithChrome(url, "search", options.Timeout)
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
    if err != nil {
        return result
    }

    // searching for data
    doc.Find("tr").Each(func(i int, s *goquery.Selection) {
        if !strings.Contains(s.Text(), "Result") && !strings.Contains(s.Text(), "Description") {
            data := strings.Split(strings.TrimSpace(s.Text()), "  ")[0]
            realdata := strings.Split(data, "\n")
            cidr := strings.TrimSpace(realdata[0])
            desc := strings.TrimSpace(realdata[len(realdata)-1])
            core.InforF(fmt.Sprintf("%s - %s", cidr, desc))
            result = append(result, fmt.Sprintf("%s", cidr))
        }
    })
    return result
}

// OrgBgbView get Org infor from bgpview.io
func OrgBgbView(options core.Options) []string {
    org := options.Net.Org
    url := fmt.Sprintf(`https://bgpview.io/search/%v`, org)
    core.InforF("Get data from: %v", url)
    var result []string
    content := core.RequestWithChrome(url, "results-tabs", options.Timeout)
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
    if err != nil {
        return result
    }

    // searching for data
    doc.Find("a").Each(func(i int, s *goquery.Selection) {
        href, _ := s.Attr("href")
        // https://bgpview.io/prefix/176.96.254.0/24
        if strings.Contains(href, "https://bgpview.io/prefix/") {
            //cidr := strings.Replace(href, "https://bgpview.io/prefix/", "", -1)
            cidr := s.Text()
            result = append(result, fmt.Sprintf("%s", cidr))
        }
    })
    return result
}

// ASNLookup get Org CIDR from asnlookup
func ASNLookup(options core.Options) []string {
    org := options.Net.Org
    url := fmt.Sprintf(`http://asnlookup.com/api/lookup?org=%v`, org)
    core.InforF("Get data from: %v", url)
    data := core.SendGET(url, options)
    var result []string
    if data == "" {
        return result
    }
    err := json.Unmarshal([]byte(data), &result)
    if err != nil {
        return result
    }

    for _, item := range result {
        core.InforF(item)
    }
    return result
}

// ASNFromIP get ip or domain from ultratools.com
func ASNFromIP(options core.Options) []string {
    var result []string
    ip := options.Net.IP
    // resolve IP
    if ip == "" && options.Net.Domain != "" {
        if resolved, err := net.LookupHost(options.Net.Domain); err == nil {
            ip = resolved[0]
        }
    }
    url := fmt.Sprintf(`https://www.ultratools.com/tools/asnInfoResult?domainName=%v`, ip)
    core.InforF("Get data from: %v", url)
    content := core.SendGET(url, options)
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
    if err != nil {
        return result
    }

    // searching for data
    asn := doc.Find(".tool-results-heading").Text()
    if asn != "" {
        result = append(result, strings.TrimSpace(asn))
    }

    return result
}
