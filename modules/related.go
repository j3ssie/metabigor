package modules

import (
    "fmt"
    "github.com/PuerkitoBio/goquery"
    "github.com/fatih/color"
    "github.com/j3ssie/metabigor/core"
    jsoniter "github.com/json-iterator/go"
    "github.com/thoas/go-funk"
    "net/url"
    "regexp"
    "strings"
)

// CrtSH get IPInfo from https://crt.sh
func CrtSH(raw string, options core.Options) (result []core.RelatedDomain) {
    technique := "certificate"
    core.InforF("Get more related domains with technique: " + color.HiMagentaString("%v:%v", technique, raw))

    targetURL := fmt.Sprintf(`https://crt.sh/?O=%v`, url.QueryEscape(raw))
    core.DebugF("Get data from: %v", targetURL)
    content, err := core.GetResponse(targetURL, options)
    if content == "" || err != nil {
        core.ErrorF("Error sending request to: %v", targetURL)
        return result
    }

    infos := ParseCertSH(content, options)
    for _, info := range infos {

        var tldInfo core.RelatedDomain
        if data, err := jsoniter.MarshalToString(info); err == nil {
            tldInfo.RawData = data
        }
        tldInfo.Domain = info.Domain
        tldInfo.Technique = technique
        tldInfo.Source = "https://crt.sh"
        tldInfo.Output = fmt.Sprintf("[%s] %s ;; %s ;; %s", technique, info.Domain, info.Org, info.CertInfo)
        result = append(result, tldInfo)
    }
    return result
}

func ReverseWhois(raw string, options core.Options) (result []core.RelatedDomain) {
    technique := "whois"
    core.InforF("Get more related domains with technique: " + color.HiMagentaString("%v:%v", technique, raw))

    targetURL := fmt.Sprintf(`https://viewdns.info/reversewhois/?q=%v`, url.QueryEscape(raw))
    core.DebugF("Get data from: %v", targetURL)
    content, err := core.GetResponse(targetURL, options)
    if content == "" || err != nil {
        core.ErrorF("Error sending request to: %v", targetURL)
        return result
    }

    doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
    doc.Find("tr").Each(func(i int, s *goquery.Selection) {
        var tldInfo core.RelatedDomain
        s.Find("td").Each(func(j int, td *goquery.Selection) {
            if strings.Count(td.Text(), "\n") > 1 || len(td.Text()) > 1000 {
                return
            }

            if j == 0 {
                tldInfo.Domain = strings.TrimSpace(td.Text())
            }

            tldInfo.Technique = technique
            tldInfo.Source = "https://viewdns.info"

            var createdDate, description string
            if j == 1 {
                createdDate = strings.TrimSpace(td.Text())
            }
            if j == 2 {
                description = strings.TrimSpace(td.Text())
            }
            tldInfo.RawData = fmt.Sprintf("%s ;; %s ;; %s", tldInfo.Domain, createdDate, description)
            tldInfo.Output = fmt.Sprintf("[%s] %s ;; %s ;; %s", technique, tldInfo.Domain, createdDate, description)
        })

        if strings.Contains(tldInfo.Domain, ".") {
            result = append(result, tldInfo)
        }
    })
    return result
}

func GoogleAnalytic(targetURL string, options core.Options) (result []core.RelatedDomain) {
    if strings.HasPrefix(targetURL, "UA-") {
        return BuiltwithUA(targetURL, options)
    }

    if !strings.HasPrefix(targetURL, "http") {
        targetURL = fmt.Sprintf("https://%s", strings.TrimSpace(targetURL))
    }
    core.InforF("Extract Google Analytics ID from: %s", targetURL)
    content, err := core.GetResponse(targetURL, options)
    if content == "" || err != nil {
        return result
    }

    core.DebugF("Extracting UA-ID from: %v", targetURL)
    UAIds := ExtractUAID(content)

    core.DebugF("Extracting GoogleTagManager from: %v", targetURL)
    gtms := ExtractGoogleTagManger(content)
    for _, gtm := range gtms {
        core.DebugF("Extracting UA-ID from: %v", gtm)
        content, err := core.GetResponse(gtm, options)
        if err == nil {
            UAIds = append(UAIds, ExtractUAID(content)...)
        }
    }

    UAIds = funk.UniqString(UAIds)

    // clean up UA IDs
    var cleanedUAIds []string
    for _, rawUAId := range UAIds {
        UAId := strings.Join(strings.Split(rawUAId, "-")[:2], "-")
        cleanedUAIds = append(cleanedUAIds, UAId)
    }

    cleanedUAIds = funk.UniqString(cleanedUAIds)
    for _, uaId := range cleanedUAIds {
        result = append(result, BuiltwithUA(uaId, options)...)
    }

    return result
}

func BuiltwithUA(UAID string, options core.Options) (result []core.RelatedDomain) {
    technique := "google-analytic"
    core.InforF("Get more related domains with technique: " + color.HiMagentaString("%v:%v", technique, UAID))
    dataURL := fmt.Sprintf(`https://builtwith.com/relationships/tag/%v`, url.QueryEscape(UAID))
    core.DebugF("Get data from: %v", dataURL)
    content, err := core.GetResponse(dataURL, options)
    // simple retry if something went wrong
    if err != nil {
        content, err = core.GetResponse(dataURL, options)
    }
    if content == "" {
        core.ErrorF("Error sending request to: %v", dataURL)
        return result
    }

    regex := regexp.MustCompile(`/relationships/[a-z0-9\-\_\.]+\.[a-z]+`)
    rawDomains := regex.FindAllStringSubmatch(content, -1)
    if len(rawDomains) == 0 {
        core.ErrorF("Error extracting domains from: %v", dataURL)
    }

    for _, domain := range rawDomains {
        var tldInfo core.RelatedDomain
        cleanedDomain := strings.ReplaceAll(domain[0], "/relationships/", "")
        tldInfo.Domain = cleanedDomain
        tldInfo.Technique = technique
        tldInfo.Source = dataURL
        tldInfo.Output = fmt.Sprintf("[%s] %s ;; %s", technique, tldInfo.Domain, dataURL)
        result = append(result, tldInfo)
    }

    return result
}
