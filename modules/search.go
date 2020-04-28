package modules

import (
	"fmt"
	"github.com/thoas/go-funk"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/j3ssie/metabigor/core"
)

// IPSearch doing searching on Onyphe
func IPSearch(options core.Options) []string {
	var result []string
	result = append(result, singleIPSearch(options.Search.Query, options)...)
	if !options.Search.Optimize {
		return result
	}
	return result
}

func singleIPSearch(query string, options core.Options) []string {
	var result []string
	result = append(result, Onyphe(query, options)...)
	return result
}

// FoFaSearch doing searching on FoFa
func FoFaSearch(options core.Options) []string {
	var result []string
	result = append(result, singleFoFaSearch(options.Search.Query, options)...)

	if !options.Search.Optimize {
		return result
	}

	moreQueries := OptimizeFofaQuery(options)
	if len(moreQueries) > 0 {
		var wg sync.WaitGroup
		count := 0
		for _, moreQuery := range moreQueries {
			wg.Add(1)
			go func(query string) {
				defer wg.Done()
				result = append(result, singleFoFaSearch(query, options)...)
			}(moreQuery)
			// limit the pool
			count++
			if count == options.Concurrency {
				wg.Wait()
				count = 0
			}
		}
	}

	return result
}

func singleFoFaSearch(query string, options core.Options) []string {
	core.InforF("Fofa Query: %v", query)
	query = core.Base64Encode(query)
	url := fmt.Sprintf(`https://fofa.so/result?qbase64=%v`, query)
	core.DebugF("Get data from: %v", url)
	var result []string

	content := core.RequestWithChrome(url, "ajax_content", options.Timeout)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return result
	}

	doc.Find(".list_mod_t").Each(func(i int, s *goquery.Selection) {
		doc.Find(".list_mod_t").Each(func(i int, s *goquery.Selection) {
			data := regexp.MustCompile(`[\s\t\r\n]+`).ReplaceAllString(strings.TrimSpace(s.Text()), " ")
			content := strings.Split(data, " ")
			if len(content) <= 1 {
				return
			}

			if len(content) <= 2 {
				line := content[0]
				if strings.HasPrefix(line, "http") {
					line = strings.Replace(line, "http://", "", -1)
					line = strings.Replace(line, "https://", "", -1)
				}
				result = append(result, fmt.Sprintf("%v", line))
				return
			}
			result = append(result, fmt.Sprintf("%v:%v", content[0], content[1]))
		})
	})

	return result
}

// OptimizeFofaQuery find more optimze
func OptimizeFofaQuery(options core.Options) []string {
	var optimzeQueries []string

	query := core.Base64Encode(options.Search.Query)
	url := fmt.Sprintf(`https://fofa.so/search/result_stats?qbase64=%v`, query)
	core.DebugF("Get optimize data from: %v", url)

	headers := map[string]string{
		"UserAgent":        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36",
		"Accept":           "*/*",
		"AcceptLang":       "en-US,en;q=0.8",
		"X-Requested-With": "XMLHttpRequest",
	}
	res, _ := core.JustSend(options, "GET", url, headers)
	content := res.Body

	var result []string
	regex := "qbase64\\=[a-zA-Z0-9%]+"
	r, rerr := regexp.Compile(regex)
	if rerr != nil {
		return result
	}
	matches := r.FindAllString(content, -1)
	if len(matches) == 0 {
		return result
	}

	for _, match := range matches {
		query := core.URLDecode(strings.Replace(match, "qbase64=", "", -1))
		optimizeQuery := strings.TrimSpace(core.Base64Decode(query))
		optimzeQueries = append(optimzeQueries, optimizeQuery)
	}

	return funk.UniqString(optimzeQueries)
}
