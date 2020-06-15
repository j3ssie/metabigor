package core

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty"
	"github.com/sirupsen/logrus"
)

var headers map[string]string

// SendGET just send GET request
func SendGET(url string, options Options) string {
	headers = map[string]string{
		"UserAgent":  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36",
		"Accept":     "*/*",
		"AcceptLang": "en-US,en;q=0.8",
	}
	resp, _ := JustSend(options, "GET", url, headers)
	return resp.Body
}

// SendPOST just send POST request
func SendPOST(url string, options Options) string {
	resp, _ := JustSend(options, "POST", url, headers)
	return resp.Body
}

// JustSend just sending request
func JustSend(options Options, method string, url string, headers map[string]string) (res Response, err error) {
	timeout := options.Timeout

	// disable log when retry
	logger := logrus.New()
	if !options.Debug {
		logger.Out = ioutil.Discard
	}

	client := resty.New()
	client.SetLogger(logger)
	client.SetTransport(&http.Transport{
		MaxIdleConns:          100,
		MaxConnsPerHost:       1000,
		IdleConnTimeout:       time.Duration(timeout) * time.Second,
		ExpectContinueTimeout: time.Duration(timeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(timeout) * time.Second,
		TLSHandshakeTimeout:   time.Duration(timeout) * time.Second,
		DisableCompression:    true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	})

	client.SetHeaders(headers)
	client.SetCloseConnection(true)

	client.SetTimeout(time.Duration(timeout) * time.Second)
	client.SetRetryWaitTime(time.Duration(timeout/2) * time.Second)
	client.SetRetryMaxWaitTime(time.Duration(timeout) * time.Second)

	if options.Proxy != "" {
		client.SetProxy(options.Proxy)
	}

	var resp *resty.Response
	// really sending things here
	method = strings.ToLower(strings.TrimSpace(method))
	switch method {
	case "get":
		resp, err = client.R().
			Get(url)
		break
	case "post":
		resp, err = client.R().
			Get(url)
		break
	}

	// in case we want to get redirect stuff
	if res.StatusCode != 0 {
		return res, nil
	}

	if err != nil || resp == nil {
		ErrorF("%v %v", url, err)
		return Response{}, err
	}

	return ParseResponse(*resp), nil
}

// ParseResponse field to Response
func ParseResponse(resp resty.Response) (res Response) {
	// var res libs.Response
	resLength := len(string(resp.Body()))
	// format the headers
	var resHeaders []map[string]string
	for k, v := range resp.RawResponse.Header {
		element := make(map[string]string)
		element[k] = strings.Join(v[:], "")
		resLength += len(fmt.Sprintf("%s: %s\n", k, strings.Join(v[:], "")))
		resHeaders = append(resHeaders, element)
	}
	// response time in second
	resTime := float64(resp.Time()) / float64(time.Second)
	resHeaders = append(resHeaders,
		map[string]string{"Total Length": strconv.Itoa(resLength)},
		map[string]string{"Response Time": fmt.Sprintf("%f", resTime)},
	)

	// set some variable
	res.Headers = resHeaders
	res.StatusCode = resp.StatusCode()
	res.Status = fmt.Sprintf("%v %v", resp.Status(), resp.RawResponse.Proto)
	res.Body = string(resp.Body())
	res.ResponseTime = resTime
	res.Length = resLength
	// beautify
	res.Beautify = BeautifyResponse(res)
	res.BeautifyHeader = BeautifyHeaders(res)
	return res
}

// BeautifyHeaders beautify response headers
func BeautifyHeaders(res Response) string {
	beautifyHeader := fmt.Sprintf("%v \n", res.Status)
	for _, header := range res.Headers {
		for key, value := range header {
			beautifyHeader += fmt.Sprintf("%v: %v\n", key, value)
		}
	}
	return beautifyHeader
}

// BeautifyResponse beautify response
func BeautifyResponse(res Response) string {
	var beautifyRes string
	beautifyRes += fmt.Sprintf("%v \n", res.Status)

	for _, header := range res.Headers {
		for key, value := range header {
			beautifyRes += fmt.Sprintf("%v: %v\n", key, value)
		}
	}

	beautifyRes += fmt.Sprintf("\n%v\n", res.Body)
	return beautifyRes
}
