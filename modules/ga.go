package modules

import (
    "fmt"
    "regexp"
    "strings"
)

func ExtractUAID(content string) (results []string) {
    regex := regexp.MustCompile(`UA-[0-9]+-[0-9]+`)
    ua := regex.FindAllStringSubmatch(content, -1)
    for _, id := range ua {
        results = append(results, id[0])
    }

    return results
}

func ExtractGoogleTagManger(content string) (results []string) {
    // regex 1
    regex := regexp.MustCompile(`www\.googletagmanager\.com/ns\.html\?id=[A-Z0-9\-]+`)
    data := regex.FindStringSubmatch(content)

    if len(data) > 0 {
        gtm := strings.Split(data[0], "id=")[1]
        results = append(results, fmt.Sprintf("https://www.googletagmanager.com/gtm.js?id=%s", gtm))
        return results
    }

    // regex 2
    regex = regexp.MustCompile("GTM-[A-Z0-9]+")
    data = regex.FindStringSubmatch(content)
    if len(data) > 0 {
        results = append(results, fmt.Sprintf("https://www.googletagmanager.com/gtm.js?id=%s", data[0]))
        return results
    }

    // regex 3
    //regex = regexp.MustCompile(`UA-[0-9]+-[0-9]+`)
    //ua := regex.FindAllStringSubmatch(content, -1)
    //for _, id := range ua {
    //    results = append(results, id[0])
    //}
    return results
}
