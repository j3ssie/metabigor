package core

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

var headers map[string]string

// SendGET just send GET request
func SendGET(url string, options Options) string {
	headers = map[string]string{
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36",
		"Accept-Encoding": "*/*",
		"Accept-Language": "en-US,en;q=0.8",
	}
	resp, _ := JustSend(options, "GET", url, headers, "")
	return resp.Body
}

func GetResponse(baseUrl string, options Options) (string, error) {
	content, err := getResponse(baseUrl, options)
	// simple retry if something went wrong
	for i := 0; i < options.Retry; i++ {
		if err != nil {
			content, err = getResponse(baseUrl, options)
		} else {
			return content, err
		}
	}

	if err != nil {
		content, err = getResponse(baseUrl, options)
	}
	return content, err
}

func getResponse(baseUrl string, options Options) (string, error) {
	client := &http.Client{
		Timeout: time.Duration(options.Timeout*3) * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Second * 60,
			}).DialContext,
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 500,
			MaxConnsPerHost:     500,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true, Renegotiation: tls.RenegotiateOnceAsClient},
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
	}

	req, _ := http.NewRequest("GET", baseUrl, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.132 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		ErrorF("%v", err)
		return "", err
	}
	defer resp.Body.Close()

	// Create a buffer to store response body
	var bodyBuilder strings.Builder

	// Read response body into the buffer
	_, err = io.Copy(&bodyBuilder, resp.Body)
	if err != nil {
		ErrorF("%v", err)
		return "", err
	}
	return bodyBuilder.String(), nil
}

// SendPOST just send POST request
func SendPOST(url string, options Options) string {
	resp, _ := JustSend(options, "POST", url, headers, "")
	return resp.Body
}

// JustSend just sending request
func JustSend(options Options, method string, url string, headers map[string]string, body string) (res Response, err error) {
	timeout := options.Timeout

	client := resty.New()
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
			SetBody([]byte(body)).
			Post(url)
		break
	}

	// in case we want to get redirect stuff
	if res.StatusCode != 0 {
		return res, nil
	}

	if err != nil || resp == nil {
		ErrorF("%v %v", url, err)
		return res, err
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
