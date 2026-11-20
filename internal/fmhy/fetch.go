package fmhy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// rawBase is the raw.githubusercontent.com root for the FMHY wiki. Raw URLs
// serve the file exactly as committed, with no HTML page around it.
const rawBase = "https://raw.githubusercontent.com/fmhy/edit/main/docs/"

// userAgent identifies kamui. GitHub asks API clients to be identifiable, and
// it costs nothing to be polite.
const userAgent = "kamui/0.1 (+https://github.com/sakshamcozykun/kamui)"

// Client downloads wiki pages and keeps a copy on disk.
//
// The cache does two jobs: it makes repeat runs instant, and it lets kamui
// still work when GitHub is unreachable or rate-limiting.
type Client struct {
	HTTP *http.Client
	Dir  string // directory holding cached pages, e.g. ~/.cache/kamui
}

// NewClient returns a Client that caches into dir. An empty dir disables
// caching entirely, which is handy in tests.
func NewClient(dir string) *Client {
	return &Client{
		HTTP: &http.Client{Timeout: 20 * time.Second},
		Dir:  dir,
	}
}

// Fetch returns the contents of one wiki page.
//
// It sends If-None-Match with the ETag we saved last time, so GitHub can
// answer "304 Not Modified" and we reuse the cached bytes instead of
// re-downloading ~120 KB. If the network fails but a cache exists, the cache
// wins over an error: a slightly stale source list beats no source list.
func (c *Client) Fetch(ctx context.Context, doc string) ([]byte, error) {
	bodyPath, etagPath := c.paths(doc)

	cached, cachedErr := os.ReadFile(bodyPath)
	etag, _ := os.ReadFile(etagPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawBase+doc, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %s: %w", doc, err)
	}
	req.Header.Set("User-Agent", userAgent)
	if cachedErr == nil && len(etag) > 0 {
		req.Header.Set("If-None-Match", strings.TrimSpace(string(etag)))
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		if cachedErr == nil {
			return cached, nil
		}
		return nil, fmt.Errorf("fetch %s: %w", doc, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified && cachedErr == nil {
		return cached, nil
	}
	if resp.StatusCode != http.StatusOK {
		if cachedErr == nil {
			return cached, nil
		}
		return nil, fmt.Errorf("fetch %s: %s", doc, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", doc, err)
	}

	c.store(bodyPath, etagPath, body, resp.Header.Get("ETag"))

	return body, nil
}

func (c *Client) paths(doc string) (body, etag string) {
	if c.Dir == "" {
		return "", ""
	}
	body = filepath.Join(c.Dir, "fmhy", doc)
	return body, body + ".etag"
}

// store writes the cache as a best-effort side effect. Failing to cache is not
// a reason to fail the command: we already have the bytes we need.
func (c *Client) store(bodyPath, etagPath string, body []byte, etag string) {
	if bodyPath == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o755); err != nil {
		return
	}
	if err := os.WriteFile(bodyPath, body, 0o644); err != nil {
		return
	}
	if etag != "" {
		_ = os.WriteFile(etagPath, []byte(etag), 0o644)
	} else {
		_ = os.Remove(etagPath)
	}
}

// ErrNoSources is returned by helpers that need at least one usable source.
var ErrNoSources = errors.New("no sources found")
