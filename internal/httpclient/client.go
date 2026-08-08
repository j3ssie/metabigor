// Package httpclient provides HTTP client utilities with retry logic and Chrome automation.
package httpclient

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/j3ssie/metabigor/internal/output"
)

const defaultUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36"

// NewClient returns a retryablehttp client that follows redirects.
func NewClient(timeoutSec, retries int, proxy string) *retryablehttp.Client {
	return newClient(timeoutSec, retries, proxy, true)
}

// NewClientNoRedirect returns a retryablehttp client that does NOT follow redirects.
func NewClientNoRedirect(timeoutSec, retries int, proxy string) *retryablehttp.Client {
	return newClient(timeoutSec, retries, proxy, false)
}

func newClient(timeoutSec, retries int, proxy string, followRedirects bool) *retryablehttp.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(timeoutSec) * time.Second,
			KeepAlive: 0,
		}).DialContext,
		DisableKeepAlives: true,
	}
	if proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			output.Debug("Using proxy: %s", proxy)
		}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(timeoutSec) * time.Second,
	}
	if !followRedirects {
		httpClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	client := retryablehttp.NewClient()
	client.HTTPClient = httpClient
	client.RetryMax = retries
	client.Logger = nil
	return client
}

// setDefaultHeaders applies the browser fingerprint every request shares. It
// lives in one place because it is the first thing to tune when a scraped site
// starts answering with a bot challenge, and a fix that reached only some of
// the callers would be worse than no fix.
func setDefaultHeaders(req *retryablehttp.Request) {
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Connection", "close")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="119", "Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
}

// Get performs a GET request and returns the body as a string. Use it when a
// non-2xx status is simply a failure; reach for Fetch when the status matters.
func Get(client *retryablehttp.Client, targetURL string) (string, error) {
	resp, err := Fetch(client, "GET", targetURL, nil, nil)
	if err != nil {
		return "", err
	}
	return resp.Text(), nil
}

// Post performs a POST request with a custom content type and body.
func Post(client *retryablehttp.Client, targetURL, contentType string, body io.Reader) (string, error) {
	resp, err := Fetch(client, "POST", targetURL, map[string]string{"Content-Type": contentType}, body)
	if err != nil {
		return "", err
	}
	return resp.Text(), nil
}

// ErrNoChrome is returned when Chrome is not available.
var ErrNoChrome = errors.New("chrome/chromium not found")

// Response is a completed HTTP exchange: the status code and the fully read
// body.
//
// Get and Post cover the common case of "give me the body or an error". Fetch
// exists for callers that must react to the status code itself - an API key
// rejected with 401, a rate limit at 429, an index reporting 404 for "nothing
// archived" - or that need raw bytes rather than a string.
type Response struct {
	StatusCode int
	Body       []byte
}

// OK reports whether the status is in the 2xx range.
func (r *Response) OK() bool {
	return r != nil && r.StatusCode >= 200 && r.StatusCode < 300
}

// Text returns the body as a string.
func (r *Response) Text() string {
	if r == nil {
		return ""
	}
	return string(r.Body)
}

// Fetch performs a request with optional custom headers and returns the status
// code alongside the body. A non-2xx status is not an error: the response is
// returned so the caller can decide what it means.
func Fetch(client *retryablehttp.Client, method, targetURL string, headers map[string]string, body io.Reader) (*Response, error) {
	output.Debug("HTTP %s %s", method, targetURL)
	req, err := retryablehttp.NewRequest(method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	setDefaultHeaders(req)
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		output.Debug("HTTP %s %s failed: %v", method, targetURL, err)
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	output.Debug("HTTP %s %s -> %d, %d bytes", method, targetURL, resp.StatusCode, len(payload))
	return &Response{StatusCode: resp.StatusCode, Body: payload}, nil
}
