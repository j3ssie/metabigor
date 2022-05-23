package modules

import (
    "crypto/tls"
    "fmt"
    "github.com/j3ssie/metabigor/core"
    "io/ioutil"
    "net/http"
    "os"
)

var tr = &http.Transport{
    TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}
var client = &http.Client{Transport: tr}

func InternetDB(IP string) string {
    ipURL := fmt.Sprintf("https://internetdb.shodan.io/%s", IP)
    core.DebugF("Getting information from: %s", ipURL)
    resp, err := client.Get(ipURL)
    if err != nil {
        fmt.Fprintf(os.Stderr, "%v", err)
        return ""
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Fprintf(os.Stderr, "%v", err)
        return ""
    }
    return string(body)
}
