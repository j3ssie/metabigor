package modules

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/j3ssie/metabigor/core"
	jsoniter "github.com/json-iterator/go"
	"github.com/thoas/go-funk"
	"net/url"
	"strings"
)

// CrtSHOrg get IPInfo from https://crt.sh
func CrtSHOrg(org string, options core.Options) []string {
	crtURL := fmt.Sprintf(`https://crt.sh/?O=%v`, url.QueryEscape(org))
	var result []string
	core.InforF("Get data from: %v", crtURL)
	content, err := core.GetResponse(crtURL, options)
	// simple retry if something went wrong
	if err != nil {
		content, err = core.GetResponse(crtURL, options)
	}
	if content == "" {
		core.ErrorF("Error sending request to: %v", crtURL)
		return result
	}

	infos := ParseCertSH(content, options)
	for _, info := range infos {
		var data string
		if options.JsonOutput {
			if options.JsonOutput {
				if data, err := jsoniter.MarshalToString(info); err == nil {
					result = append(result, data)

					if !options.Quiet {
						fmt.Println(data)
					}

				}
				continue
			}
		}

		if options.Verbose {
			data = fmt.Sprintf("%s ;; %s ;; %s", info.Domain, info.Org, info.CertInfo)
		} else {
			data = info.Domain
		}
		result = append(result, data)
	}
	result = funk.UniqString(result)

	if !options.Quiet {
		fmt.Println(strings.Join(result, "\n"))
	}
	return result
}

type CertData struct {
	Domain   string
	CertInfo string
	Org      string
	WildCard bool
}

func ParseCertSH(content string, options core.Options) []CertData {
	var results []CertData
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		core.ErrorF("Error parsing body: %v", err)
		return results
	}

	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		var certInfo CertData
		s.Find("td").Each(func(j int, td *goquery.Selection) {
			if j == 4 {
				certInfo.Domain = strings.TrimSpace(td.Text())
			}
			if j == 5 {
				certInfo.Org = strings.TrimSpace(td.Text())
			}
			if j == 6 {
				certInfo.CertInfo = strings.TrimSpace(td.Text())
			}
		})
		// remove some noise
		if strings.Contains(certInfo.Domain, ".") && !strings.Contains(certInfo.Domain, " ") {
			if options.Cert.Clean {
				certInfo.Domain = strings.ReplaceAll(certInfo.Domain, "*.", "")
				results = append(results, certInfo)
			} else if options.Cert.OnlyWildCard {
				if strings.Contains(certInfo.Domain, "*") {
					certInfo.WildCard = true
					results = append(results, certInfo)
				}
			} else {
				results = append(results, certInfo)
			}
		}

	})

	return results
}
