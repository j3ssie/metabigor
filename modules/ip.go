package modules

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
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
	data := ParseOnyphe(content, options)

	for _, item := range data {
		result = append(result, fmt.Sprintf("%v:%v", query, item))
	}
	return result
}

// ParseOnyphe parsing data from Onyphe
func ParseOnyphe(content string, options core.Options) []string {
	var result []string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return result
	}

	// searching for data
	info := make(map[string]string)
	doc.Find(".features-list").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		// basic info part
		if strings.Contains(text, "geoloc") {
			tds := s.Find("td").Children()
			if strings.Contains(tds.Text(), "organization") {
				for i := range tds.Nodes {
					if i == 0 {
						tag := tds.Eq(i)
						data := tds.Eq(i + 1)
						info[tag.Text()] = data.Text()
						continue
					}
					if i%2 != 0 {
						tag := tds.Eq(i)
						data := tds.Eq(i + 1)
						info[tag.Text()] = data.Text()
					}
				}
			}
		}
		// open port
		if strings.Contains(text, "synscan") {
			var port string
			s.Find("a").Each(func(i int, tag *goquery.Selection) {
				href, _ := tag.Attr("href")
				if strings.Contains(href, "port") {
					port = tag.Text()
					result = append(result, port)
				}
			})
		}
	})

	// more info in verbose mode
	if options.Verbose {
		for k,v := range info {
			data := fmt.Sprintf("%v|%v", k,v)
			result = append(result, data)
		}
	}
	return result
}
