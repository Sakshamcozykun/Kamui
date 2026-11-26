// Package search fans a title out across a set of listed sources concurrently.
//
// It deliberately uses generic search patterns rather than site-specific
// scrapers. That means it works on many plain, server-rendered sites with
// zero configuration, and quietly returns nothing on JavaScript-only ones.
package search

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

// dedupeByHost keeps the highest-scored result per host domain.
// This ensures cross-provider diversity — e.g. "Dune from Site A",
// "Dune from Site B" — instead of three results from the same site.
func dedupeByHost(results []Result) []Result {
	seen := make(map[string]bool)
	out := make([]Result, 0, len(results))
	for _, r := range results {
		u, err := url.Parse(r.URL)
		if err != nil {
			continue
		}
		host := u.Hostname()
		if seen[host] {
			continue // already have a result from this provider
		}
		seen[host] = true
		out = append(out, r)
	}
	return out
}

// Result is one candidate page found on a source site.
type Result struct {
	Site  string
	Title string
	URL   string
	Score float64
	Order int
}

// Attempt records what happened on one site, for --debug.
type Attempt struct {
	Host    string
	URL     string
	Code    int
	Anchors int
	Matched int
	Err     string
}

// Config controls the run.
type Config struct {
	Workers    int
	Timeout    time.Duration
	MaxBytes   int64
	PerSite    int
	Limit      int
	NoDedupe   bool // skip host dedupe (caller will dedupe after its own filtering)
	OnProgress func(done, total int)
}

// DedupeByHost keeps the highest-scored result per host domain.
// Exported so callers can dedupe after their own filtering.
func DedupeByHost(results []Result) []Result {
	return dedupeByHost(results)
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Workers:  12,
		Timeout:  8 * time.Second,
		MaxBytes: 512 << 10,
		PerSite:  3,
		Limit:    25,
	}
}

const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36 kamui/0.2"

var (
	// A very small HTML parser: anchor tags with a quoted href. It covers
	// server-rendered result lists and nothing else. See the package comment.
	anchorRe = regexp.MustCompile(`(?is)<a\s+[^>]*?href=['"]([^'"]+)['"][^>]*>(.*?)</a>`)
	tagRe    = regexp.MustCompile(`(?s)<[^>]+>`)
	spaceRe  = regexp.MustCompile(`\s+`)
)

// searchPaths are the common query shapes used by small streaming sites. The
// first one that yields a match wins, so the most common shape comes first.
// %[1]s is the URL-encoded query, %[2]s is the hyphenated slug. Explicit
// indexes let every template take the same two arguments.
var searchPaths = []string{
	"/search?q=%[1]s",
	"/?s=%[1]s",
	"/search/%[2]s",
	"/search?keyword=%[1]s",
}

// Run searches every source concurrently. It returns matched results and one
// attempt record per source, in the same order.
func Run(ctx context.Context, sources []fmhy.Source, query string, cfg Config) ([]Result, []Attempt) {
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultConfig().Timeout
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = DefaultConfig().MaxBytes
	}
	if cfg.PerSite < 1 {
		cfg.PerSite = DefaultConfig().PerSite
	}

	tokens := strings.Fields(strings.ToLower(query))
	results := make([]Result, 0, len(sources)*cfg.PerSite)
	attempts := make([]Attempt, len(sources))

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = cfg.Workers
	client := &http.Client{Transport: transport}

	sem := make(chan struct{}, cfg.Workers)
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex // guards the shared results slice
		done int
	)

	for i, source := range sources {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, source fmhy.Source) {
			defer wg.Done()
			defer func() { <-sem }()

			found, attempt := searchOne(ctx, client, source, tokens, query, cfg)
			attempts[i] = attempt
			if len(found) > 0 {
				mu.Lock()
				results = append(results, found...)
				mu.Unlock()
			}
			if cfg.OnProgress != nil {
				mu.Lock()
				done++
				completed := done
				mu.Unlock()
				cfg.OnProgress(completed, len(sources))
			}
		}(i, source)
	}
	wg.Wait()

	slices.SortFunc(results, func(a, b Result) int {
		if c := int(b.Score*1000) - int(a.Score*1000); c != 0 {
			return c
		}
		return a.Order - b.Order
	})

	// Dedupe by host domain for cross-provider diversity.
	if !cfg.NoDedupe {
		results = dedupeByHost(results)
	}

	if cfg.Limit > 0 && len(results) > cfg.Limit {
		results = results[:cfg.Limit]
	}
	return results, attempts
}

func searchOne(ctx context.Context, client *http.Client, source fmhy.Source, tokens []string, query string, cfg Config) ([]Result, Attempt) {
	attempt := Attempt{Host: source.Host()}
	base := baseURL(source.URL)
	if base == "" {
		attempt.Err = "bad source URL"
		return nil, attempt
	}

	for _, candidate := range candidateURLs(base, query) {
		found, code, anchors, err := fetchAndMatch(ctx, client, candidate, base, tokens, source, cfg)
		attempt.URL, attempt.Code, attempt.Anchors, attempt.Matched = candidate, code, anchors, len(found)
		attempt.Err = err
		if len(found) > 0 {
			return found, attempt
		}
	}
	return nil, attempt
}

func fetchAndMatch(ctx context.Context, client *http.Client, target, base string, tokens []string, source fmhy.Source, cfg Config) ([]Result, int, int, string) {
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, 0, 0, err.Error()
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", base+"/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, 0, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, 0, fmt.Sprintf("http %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, cfg.MaxBytes))
	if err != nil {
		return nil, resp.StatusCode, 0, err.Error()
	}

	root, _ := url.Parse(base)
	seen := make(map[string]bool)
	out := make([]Result, 0, cfg.PerSite)
	anchors := 0

	for _, m := range anchorRe.FindAllStringSubmatch(string(body), -1) {
		anchors++
		title := cleanText(m[2])
		href := cleanText(m[1])
		if title == "" || href == "" || strings.HasPrefix(href, "#") {
			continue
		}
		if strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
			continue
		}

		absolute, err := resolve(root, href)
		if err != nil || absolute.Host != root.Host {
			continue // skip off-site ads and trackers
		}
		if seen[absolute.String()] {
			continue
		}

		score, ok := scoreMatch(title, absolute.String(), tokens, source.Starred)
		if !ok {
			continue
		}
		seen[absolute.String()] = true
		out = append(out, Result{
			Site: source.Name, Title: title, URL: absolute.String(),
			Score: score, Order: source.Order,
		})
		if len(out) >= cfg.PerSite {
			break
		}
	}

	return out, resp.StatusCode, anchors, ""
}

func scoreMatch(title, href string, tokens []string, starred bool) (float64, bool) {
	lowerTitle := strings.ToLower(title)
	lowerHref := strings.ToLower(href)

	// The visible text must actually match. A URL full of the right words is
	// not enough: search pages often put those words in tracking links with
	// useless text like "Most relevant" or "Click here".
	titleHits := 0
	for _, token := range tokens {
		if strings.Contains(lowerTitle, token) {
			titleHits++
		}
	}
	// Every token must appear in the title. Partial matches are how you end
	// up playing Toy Story 3 when you asked for Toy Story 4.
	if titleHits != len(tokens) {
		return 0, false
	}

	score := float64(titleHits) / float64(len(tokens))
	if strings.Contains(lowerHref, strings.Join(tokens, "-")) {
		score += 0.5
	}
	if starred {
		score += 0.3
	}
	if score < 0.75 {
		return 0, false
	}
	return score, true
}

func cleanText(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

func resolve(root *url.URL, href string) (*url.URL, error) {
	ref, err := url.Parse(href)
	if err != nil {
		return nil, err
	}
	return root.ResolveReference(ref), nil
}

func baseURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	// Host, not Hostname(): a local test server (and some real sites) use a
	// non-default port, and dropping it would send requests to port 80/443.
	return u.Scheme + "://" + strings.ToLower(u.Host)
}

func candidateURLs(base, query string) []string {
	q := url.QueryEscape(query)
	slug := url.PathEscape(strings.Join(strings.Fields(strings.ToLower(query)), "-"))

	out := make([]string, 0, len(searchPaths))
	for _, path := range searchPaths {
		out = append(out, base+fmt.Sprintf(path, q, slug))
	}
	return out
}
