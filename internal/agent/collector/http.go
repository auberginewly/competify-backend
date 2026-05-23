// Package collector provides shared HTTP helpers for data collection agents.
package collector

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

func newHTTPClient() *resty.Client {
	return resty.New().
		SetTimeout(20 * time.Second).
		SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36").
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8").
		SetHeader("Accept-Language", "en-US,en;q=0.5")
}

// fetchText performs a GET request and returns the response body, status code, and error.
func fetchText(ctx context.Context, url string) (string, int, error) {
	resp, err := newHTTPClient().R().SetContext(ctx).Get(url)
	if err != nil {
		return "", 0, err
	}
	return resp.String(), resp.StatusCode(), nil
}

// extractTitle pulls the <title> tag content from raw HTML.
func extractTitle(html string) string {
	re := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	m := re.FindStringSubmatch(html)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// extractMetaDescription pulls the meta description from raw HTML.
func extractMetaDescription(html string) string {
	// Try name="description" content="..."
	re := regexp.MustCompile(`(?i)<meta[^>]*name=["']description["'][^>]*content=["']([^"']+)["']`)
	m := re.FindStringSubmatch(html)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	// Try content="..." name="description"
	re = regexp.MustCompile(`(?i)<meta[^>]*content=["']([^"']+)["'][^>]*name=["']description["']`)
	m = re.FindStringSubmatch(html)
	if len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// truncate limits a string to max runes and appends "..." if truncated.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
