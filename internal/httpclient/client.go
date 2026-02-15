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

// Get performs a GET request and returns the body as a string.
func Get(client *retryablehttp.Client, targetURL string) (string, error) {
	output.Debug("HTTP GET %s", targetURL)
	req, err := retryablehttp.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Connection", "close")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="119", "Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)

	resp, err := client.Do(req)
	if err != nil {
		output.Debug("HTTP GET %s failed: %v", targetURL, err)
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	output.Debug("HTTP GET %s -> %d", targetURL, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	output.Debug("HTTP GET %s -> %d bytes", targetURL, len(body))
	return string(body), nil
}

// Post performs a POST request with a custom content type and body.
func Post(client *retryablehttp.Client, targetURL, contentType string, body io.Reader) (string, error) {
	output.Debug("HTTP POST %s", targetURL)
	req, err := retryablehttp.NewRequest("POST", targetURL, body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", defaultUA)
	req.Header.Set("Connection", "close")
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="119", "Chromium";v="119", "Not?A_Brand";v="24"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)

	resp, err := client.Do(req)
	if err != nil {
		output.Debug("HTTP POST %s failed: %v", targetURL, err)
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	output.Debug("HTTP POST %s -> %d", targetURL, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	output.Debug("HTTP POST %s -> %d bytes", targetURL, len(respBody))
	return string(respBody), nil
}

// ErrNoChrome is returned when Chrome is not available.
var ErrNoChrome = errors.New("chrome/chromium not found")
