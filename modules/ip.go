package modules

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/j3ssie/metabigor/core"
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
		return ""
	}
	defer resp.Body.Close()

	// Create a buffer to store response body
	var bodyBuilder strings.Builder

	// Read response body into the buffer
	_, err = io.Copy(&bodyBuilder, resp.Body)
	if err != nil {
		return ""
	}

	return bodyBuilder.String()
}
