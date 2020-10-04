package modules

import (
	"fmt"
	"github.com/Jeffail/gabs"
	"github.com/j3ssie/metabigor/core"
	jsoniter "github.com/json-iterator/go"
	"strings"
)

type CVEData struct {
	CVE   string
	Title string
	Desc  string
	Raw   string
	RawQuery   string
}

// Vulners get Org CIDR from asnlookup
func Vulners(options core.Options) []string {
	var result []string

	query := options.Search.Query
	// parsing input
	software, version := PrepareQuery(query)
	body := fmt.Sprintf(`{"apiKey":"","software":"%s","type":"software","version":"%s"}`, software, version)

	url := fmt.Sprintf(`https://vulners.com/api/v3/burp/software/`)
	core.InforF("Get data from: %v", url)
	headers := map[string]string{
		"UserAgent":    "vulners-burpscanner-v-1.2",
		"Content-type": "application/json",
	}
	resp, _ := core.JustSend(options, "POST", url, headers, body)
	if resp.StatusCode != 200 {
		return result
	}
	jsonParsed, err := gabs.ParseJSON([]byte(resp.Body))
	if err != nil {
		core.ErrorF("Error parse JSON Data")
		return result
	}

	content := jsonParsed.S("data").S("search")
	for _, item := range content.Children() {
		var cveData CVEData
		cveData.RawQuery = query
		for k, v := range item.S("_source").ChildrenMap() {
			if k == "description" {
				cveData.Desc = v.Data().(string)
			}
			if k == "title" {
				cveData.Title = v.Data().(string)
			}
			if k == "id" {
				cveData.CVE = v.Data().(string)
			}
		}

		if options.JsonOutput {
			if data, err := jsoniter.MarshalToString(cveData); err == nil {
				result = append(result, data)
			}
			continue
		}
		info := fmt.Sprintf("%s ;; %s ;; %s", cveData.RawQuery, cveData.CVE, cveData.Desc)
		result = append(result, info)
	}

	return result
}

func PrepareQuery(raw string) (string, string) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if !strings.Contains(raw, "|") {
		return raw, "0"
	}

	// apache|1.0
	data := strings.Split(raw, "|")
	software := strings.TrimSpace(data[0])
	version := strings.TrimSpace(data[1])
	return software, version
}
