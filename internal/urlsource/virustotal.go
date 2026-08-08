package urlsource

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/j3ssie/metabigor/internal/output"
)

const (
	virusTotalAPI = "https://www.virustotal.com/api/v3"
	// virusTotalPageSize is the largest page the v3 relationship endpoints serve.
	virusTotalPageSize = 40
)

// virusTotalPage is one page of a v3 relationship response.
type virusTotalPage struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			URL                 string `json:"url"`
			LastHTTPCode        int    `json:"last_http_response_code"`
			LastHTTPContentType string `json:"last_http_response_content_type"`
			LastAnalysisDate    int64  `json:"last_analysis_date"`
			LastModified        int64  `json:"last_modification_date"`
		} `json:"attributes"`
	} `json:"data"`
	Links struct {
		Next string `json:"next"`
	} `json:"links"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// fetchVirusTotal pulls URLs from the VirusTotal v3 domain relationships.
//
// Two relationships are walked: `urls`, which is the URL data proper, and
// `subdomains`, whose entries are hostnames. Hostnames are emitted with a
// scheme so every result this command produces is a URL.
//
// This uses v3 rather than the legacy v2 domain/report endpoint that other
// tools call: v2 is deprecated, and v3 paginates instead of truncating. Note
// that a free key allows only four requests per minute, so a large target will
// spend most of its time backing off - --limit-requests is the lever there.
func (r *run) fetchVirusTotal() error {
	apiKey := r.key(SourceVirusTotal)
	if apiKey == "" {
		return fmt.Errorf("no API key configured")
	}

	headers := map[string]string{"x-apikey": apiKey}
	b := r.newBudget(SourceVirusTotal)
	host := url.PathEscape(r.target.Host)

	for _, relationship := range []string{"urls", "subdomains"} {
		next := fmt.Sprintf("%s/domains/%s/%s?limit=%d", virusTotalAPI, host, relationship, virusTotalPageSize)

		for next != "" {
			resp, err := r.claimGet(b, next, headers)
			if err != nil {
				return err
			}
			if resp == nil {
				return nil
			}

			switch resp.StatusCode {
			case 401, 403:
				return fmt.Errorf("API key rejected (HTTP %d)", resp.StatusCode)
			case 404:
				// Nothing known about this domain.
				next = ""
				continue
			case 429:
				output.Verbose("virustotal: quota reached while reading %s; results are partial", relationship)
				next = ""
				continue
			}
			if !resp.OK() {
				return fmt.Errorf("virustotal.com returned HTTP %d", resp.StatusCode)
			}

			var page virusTotalPage
			if err := json.Unmarshal(resp.Body, &page); err != nil {
				return fmt.Errorf("parse response: %w", err)
			}
			if page.Error.Message != "" {
				return fmt.Errorf("virustotal: %s", page.Error.Message)
			}

			for _, item := range page.Data {
				status := ""
				if item.Attributes.LastHTTPCode > 0 {
					status = strconv.Itoa(item.Attributes.LastHTTPCode)
				}
				timestamp := timestampFromUnix(item.Attributes.LastAnalysisDate)
				if timestamp == "" {
					timestamp = timestampFromUnix(item.Attributes.LastModified)
				}

				// A url object carries the URL in its attributes; a subdomain
				// object carries the hostname as its id.
				value := item.Attributes.URL
				if value == "" {
					value = withScheme(item.ID)
				}
				r.add(SourceVirusTotal, value, status, item.Attributes.LastHTTPContentType, timestamp)
			}

			next = page.Links.Next
		}
	}
	return nil
}

// timestampFromUnix renders a Unix timestamp in the 14-digit CDX form.
func timestampFromUnix(sec int64) string {
	if sec <= 0 {
		return ""
	}
	return time.Unix(sec, 0).UTC().Format("20060102150405")
}
