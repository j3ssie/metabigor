package modules

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/j3ssie/metabigor/core"
)

// FoFa doing searching on FoFa
func FoFa(options core.Options) string {
	asn := options.Net.Asn
	url := fmt.Sprintf(`https://ipinfo.io/%v`, asn)

	if options.Debug {
		core.DebugF(url)
	}

	content := core.RequestWithChrome(url, "ipTabContent")
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return ""
	}
	// searching for data
	doc.Find("tr").Each(func(i int, s *goquery.Selection) {
		data := strings.Split(strings.TrimSpace(s.Text()), "  ")
		cidr := strings.TrimSpace(data[0])
		desc := strings.TrimSpace(data[len(data)-1])
		fmt.Printf("Link #%d:\ntext: %s - %s\n\n", i, cidr, desc)
	})
	return content
}

// // FofaLogin do login to Fofa
// func FofaSession(options core.Options) bool {
// 	// check session is still valid or not
// 	sessURL := "https://fofa.so/user/users/info"
// 	headers := map[string]string{
// 		"Cookie": core.GetSess("fofa", options),
// 	}
// 	core.DebugF(sessURL)
// 	req := core.HTTPRequest{
// 		Method:  "GET",
// 		URL:     "https://fofa.so/user/users/info",
// 		Headers: headers,
// 	}
// 	resp := core.SendRequest(req, options)
// 	if resp.StatusCode == 200 {
// 		return true
// 	}

// 	return false
// }
