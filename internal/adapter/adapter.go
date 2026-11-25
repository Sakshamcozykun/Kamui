// Package adapter provides embed-provider adapters that turn an IMDB ID
// into a playable stream URL.
package adapter

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Adapter turns an IMDB ID into a direct stream URL.
type Adapter interface {
	Name() string
	Extract(ctx context.Context, imdbID string) (string, error)
}

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36"

// streamRe matches direct stream URLs in HTML.
var streamRe = regexp.MustCompile(`https?://[^\s"'<>\\]+?\.(?:m3u8|mp4)(?:\?[^\s"'<>\\]*)?`)

// iframeRe matches iframe src attributes.
var iframeRe = regexp.MustCompile(`(?i)<iframe[^>]+src=["']([^"']+)["']`)

// fetchPage fetches a URL and returns its body (up to maxBytes).
func fetchPage(ctx context.Context, client *http.Client, rawURL string, maxBytes int64) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", rawURL)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// extractStream looks for a direct stream URL in HTML. If none found,
// follows the first iframe and tries again.
func extractStream(ctx context.Context, client *http.Client, html string, baseURL string, depth int) string {
	if depth > 2 {
		return ""
	}

	// look for direct stream URLs
	if m := streamRe.FindString(html); m != "" {
		return m
	}

	// follow first iframe
	if m := iframeRe.FindStringSubmatch(html); m != nil {
		iframeURL := m[1]
		if !strings.HasPrefix(iframeURL, "http") {
			iframeURL = baseURL + iframeURL
		}
		if body, err := fetchPage(ctx, client, iframeURL, 512<<10); err == nil {
			if s := extractStream(ctx, client, body, iframeURL, depth+1); s != "" {
				return s
			}
		}
	}
	return ""
}

// newClient returns an HTTP client with reasonable settings.
func newClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}
