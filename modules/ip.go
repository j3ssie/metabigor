package modules

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/genkiroid/cert"
	"github.com/j3ssie/metabigor/core"
	"strings"
)

// Onyphe get IPInfo from https://www.onyphe.io
func Onyphe(query string, options core.Options) []string {
	url := fmt.Sprintf(`https://www.onyphe.io/search/?query=%v`, query)
	var result []string
	core.InforF("Get data from: %v", url)
	content := core.SendGET(url, options)
	if content == "" {
		return result
	}
	info := ParseOnyphe(content)
	if !options.Verbose {
		result = append(result, fmt.Sprintf("[onyphe] %v ports|%v", query, info["ports"]))
		return result
	}
	for key, value := range info {
		if key != "port" {
			result = append(result, fmt.Sprintf("[onyphe] %v %v|%v", query, key, value))
		}
	}
	cert := CertsInfo(query, info["ports"])
	if cert != "" {
		result = append(result, cert)
	}
	return result
}

// Shodan get IPInfo from https://www.shodan.io
func Shodan(query string, options core.Options) []string {
	url := fmt.Sprintf(`https://www.shodan.io/host/%v`, query)
	var result []string
	core.InforF("Get data from: %v", url)
	content := core.SendGET(url, options)
	if content == "" {
		core.DebugF("Error in sending to Shodan")
		return result
	}
	info := ParseShodan(content)
	if !options.Verbose {
		result = append(result, fmt.Sprintf("[shodan] %v ports|%v", query, info["ports"]))
		return result
	}
	for key, value := range info {
		if key != "port" {
			result = append(result, fmt.Sprintf("[shodan] %v %v|%v", query, key, value))
		}
	}
	cert := CertsInfo(query, info["ports"])
	if cert != "" {
		result = append(result, cert)
	}
	return result
}

// ParseOnyphe parsing data from Onyphe
func ParseOnyphe(content string) map[string]string {
	info := make(map[string]string)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		core.DebugF("Error parsing HTML")
		return info
	}

	// searching for data
	doc.Find(".features-list").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		// basic info part
		if strings.Contains(text, "geoloc") {
			s.Find("tr").Each(func(i int, tr *goquery.Selection) {
				text := tr.Text()
				if strings.Contains(text, "organization") {
					organization := strings.Replace(text, "organization", "", -1)
					info["organization"] = organization
				}

				if strings.Contains(text, "asn") {
					asn := strings.Replace(text, "asn", "", -1)
					info["asn"] = asn
				}

				if strings.Contains(text, "subnet") {
					subnet := strings.Replace(text, "subnet", "", -1)
					info["subnet"] = subnet
				}

				if strings.Contains(text, "city") {
					city := strings.Replace(text, "city", "", -1)
					info["city"] = city
				}

				if strings.Contains(text, "country") {
					country := strings.Replace(text, "country", "", -1)
					info["country"] = country
				}
			})
		}

		// open port
		if strings.Contains(text, "synscan") {
			var ports []string
			s.Find("a").Each(func(i int, tag *goquery.Selection) {
				href, _ := tag.Attr("href")
				if strings.Contains(href, "port") {
					port := tag.Text()
					ports = append(ports, port)
				}
			})
			info["ports"] = strings.Join(ports, ",")
		}
	})
	return info
}

// ParseShodan parsing data from Onyphe
func ParseShodan(content string) map[string]string {
	info := make(map[string]string)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		core.DebugF("Error parsing HTML")
		return info
	}

	// searching for data
	doc.Find(".table").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		// basic info part
		if strings.Contains(text, "Country") {
			s.Find("tr").Each(func(i int, tr *goquery.Selection) {
				text := tr.Text()
				if strings.Contains(text, "Organization") {
					organization := strings.Replace(text, "Organization", "", -1)
					info["organization"] = strings.TrimSpace(organization)
				}

				if strings.Contains(text, "ASN") {
					asn := strings.Replace(text, "ASN", "", -1)
					info["asn"] = strings.TrimSpace(asn)
				}

				if strings.Contains(text, "ISP") {
					ISP := strings.Replace(text, "ISP", "", -1)
					info["isp"] = strings.TrimSpace(ISP)
				}

				if strings.Contains(text, "Hostnames") {
					hostnames := strings.Replace(text, "Hostnames", "", -1)
					info["hostnames"] = strings.TrimSpace(hostnames)
				}

				if strings.Contains(text, "Country") {
					country := strings.Replace(text, "Country", "", -1)
					info["country"] = strings.TrimSpace(country)
				}
			})
		}
	})

	// ports part
	var ports []string
	doc.Find(".services .service-long").Each(func(i int, s *goquery.Selection) {
		port := strings.Replace(strings.TrimSpace(s.Find(".service-details").Text()), "\n", "/", -1)
		port = strings.Replace(port, "///", "", -1)
		if port != "" {
			ports = append(ports, port)
		}
	})
	//fmt.Println(ports)
	info["ports"] = strings.Join(ports, ",")
	return info
}

// SecurityTrails get IPInfo from https://securitytrails.com/list/ip/196.3.50.77
func SecurityTrails(query string, options core.Options) []string {
	url := fmt.Sprintf(`https://securitytrails.com/list/ip/%v`, query)
	var result []string
	core.InforF("Get data from: %v", url)
	content := core.RequestWithChrome(url, "root", options.Timeout)
	if content == "" {
		core.DebugF("Error in sending to SecurityTrails")
		return result
	}
	datas := ParseSecurityTrails(content)
	for key, value := range datas {
		result = append(result, fmt.Sprintf("[securitytrails] %v %v|%v", query, key, value))
	}
	return result
}

// ParseSecurityTrails parsing data from Onyphe
func ParseSecurityTrails(content string) []map[string]string {
	var result []map[string]string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		core.DebugF("Error parsing HTML")
		return result
	}

	// searching for data
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		info := make(map[string]string)
		//text := s.Text()
		// basic info part
		s.Find("tr").Each(func(i int, tr *goquery.Selection) {
			text := tr.Text()
			if len(text) > 2 && !strings.Contains(text, "www") && !strings.Contains(text, "apex_domain") {
				info["data"] = text[1:]
				return
			}

		})
		result = append(result, info)
	})

	return result
}

// CertsInfo get cert info
func CertsInfo(query string, rports string) string {
	var certs cert.Certs
	var err error
	var domains []string
	cert.SkipVerify = true
	ports := strings.Split(rports, ",")

	for _, port := range ports {
		if strings.Contains(port, "/") {
			port = strings.Split(port, "/")[0]
		}
		raw := fmt.Sprintf("%v:%v", query, port)
		core.DebugF("SSL Info: %v", raw)
		certs, err = cert.NewCerts([]string{raw})
		if err != nil {
			continue
		}
		for _, certItem := range certs {
			for _, domain := range certItem.SANs {
				domains = append(domains, domain)
			}
		}
	}
	if len(domains) == 0 {
		return ""
	}
	return fmt.Sprintf("[Cert] %v %v", query, strings.Join(domains, ","))
}
