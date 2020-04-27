package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/j3ssie/osmedeus/utils"
	"github.com/parnurzeal/gorequest"
)

// RequestWithChrome Do request with real browser
func RequestWithChrome(url string, contentID string, timeout int) string {
	// prepare the chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-automation", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)

	allocCtx, bcancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer bcancel()

	ctx, cancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// run task list
	var data string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.OuterHTML(contentID, &data, chromedp.NodeVisible, chromedp.ByID),
	)
	DebugF(data)

	// clean chromedp-runner folder
	cleanUp()

	if err != nil {
		utils.InforF("[ERRR] %v", err)
		return ""
	}
	return data
}

func cleanUp() {
	tmpFolder := path.Join(os.TempDir(), "chromedp-runner*")
	if _, err := os.Stat("/tmp/"); !os.IsNotExist(err) {
		tmpFolder = path.Join("/tmp/", "chromedp-runner*")
	}
	junks, err := filepath.Glob(tmpFolder)
	if err != nil {
		return
	}
	for _, junk := range junks {
		os.RemoveAll(junk)
	}
}

// SendGET just send GET request
func SendGET(url string, options Options) string {
	req := HTTPRequest{
		Method: "GET",
		URL:    url,
	}
	resp := SendRequest(req, options)
	return resp.Body
}

// SendPOST just send POST request
func SendPOST(url string, options Options) string {
	req := HTTPRequest{
		Method: "POST",
		URL:    url,
	}
	resp := SendRequest(req, options)
	return resp.Body
}

// SendRequest just send GET request
func SendRequest(req HTTPRequest, options Options) HTTPResponse {
	method := req.Method
	url := req.URL
	headers := req.Headers
	body := req.Body
	// default user-agent
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["User-Agent"] = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/82.0.4052.0 Safari/537.36"

	// new client
	client := gorequest.New().TLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	client.Timeout(time.Duration(options.Timeout) * time.Second)
	if options.Proxy != "" {
		client.Proxy(options.Proxy)
	}
	var res HTTPResponse
	// choose method
	switch method {
	case "GET":
		client.Get(url)
		break
	case "POST":
		client.Post(url)
		break
	case "PUT":
		client.Put(url)
		break
	case "HEAD":
		client.Head(url)
		break
	case "PATCH":
		client.Patch(url)
		break
	case "DELETE":
		client.Delete(url)
		break
	}

	timeStart := time.Now()
	for k, v := range headers {
		client.AppendHeader(k, v)
	}
	if body != "" {
		client.Send(body)
	}

	// really sending stuff
	resp, resBody, errs := client.End()
	resTime := time.Since(timeStart).Seconds()

	if len(errs) > 0 && res.StatusCode != 0 {
		return res
	} else if len(errs) > 0 {
		ErrorF("Error sending %v", errs)
		return HTTPResponse{}
	}

	resp.Body.Close()
	// return ParseResponse(resp, resBody, resTime), nil

	return ParseResponse(resp, resBody, resTime)

}

// ParseResponse field to Response
func ParseResponse(resp gorequest.Response, resBody string, resTime float64) (res HTTPResponse) {
	// var res libs.Response
	resLength := len(string(resBody))

	// format the headers
	var resHeaders []map[string]string
	for k, v := range resp.Header {
		element := make(map[string]string)
		element[k] = strings.Join(v[:], "")
		resLength += len(fmt.Sprintf("%s: %s\n", k, strings.Join(v[:], "")))
		resHeaders = append(resHeaders, element)
	}
	// respones time in second
	resHeaders = append(resHeaders,
		map[string]string{"Total Length": strconv.Itoa(resLength)},
		map[string]string{"Response Time": fmt.Sprintf("%f", resTime)},
	)

	// set some variable
	res.Headers = resp.Header
	res.StatusCode = resp.StatusCode
	res.Status = resp.Status
	res.Body = resBody
	res.ResponseTime = resTime
	res.Length = resLength
	return res
}
