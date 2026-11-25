// Package probe checks which FMHY-listed sites are reachable and what kind of
// site each one is.
//
// A third of a typical FMHY list is dead or blocked from any given network.
// Probing turns that list from "300 links someone found once" into live data.
package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

// Status is the coarse outcome of one check.
type Status string

const (
	Alive   Status = "alive"
	Blocked Status = "blocked"
	Dead    Status = "dead"
	Unknown Status = "unknown"
)

// Tier is a family of site, detected by markers in the homepage HTML. Sites
// built from the same open-source project can share one adapter later.
type Tier string

const (
	TierMovieWeb   Tier = "movie-web"
	TierPStream    Tier = "p-stream"
	TierVidSrc     Tier = "vidsrc"
	TierPlain      Tier = "plain"
	TierCloudflare Tier = "cloudflare"
	TierUnknown    Tier = "unknown"
)

// Result is the outcome for one source.
type Result struct {
	Source  fmhy.Source
	Status  Status
	Tier    Tier
	Code    int
	Latency time.Duration
	Err     string
}

// Config controls the run.
type Config struct {
	Workers  int
	Timeout  time.Duration
	MaxBytes int64 // only the start of each page is needed for fingerprinting
}

// DefaultConfig returns sensible, polite defaults.
func DefaultConfig() Config {
	return Config{
		Workers:  16,
		Timeout:  10 * time.Second,
		MaxBytes: 64 << 10, // 64 KiB
	}
}

const userAgent = "kamui/0.2 (+https://github.com/sakshamcozykun/kamui)"

// Run checks every source concurrently.
//
// Results are written to distinct indexes of a preallocated slice, so no
// mutex is needed: goroutines touching different elements of one slice are
// safe. A buffered channel acts as a semaphore to cap in-flight requests.
func Run(ctx context.Context, sources []fmhy.Source, cfg Config) []Result {
	if cfg.Workers < 1 {
		cfg.Workers = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultConfig().Timeout
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = DefaultConfig().MaxBytes
	}

	results := make([]Result, len(sources))
	sem := make(chan struct{}, cfg.Workers)
	client := newClient(cfg)

	var wg sync.WaitGroup
	for i, source := range sources {
		wg.Add(1)
		sem <- struct{}{} // blocks once Workers requests are in flight
		go func(i int, source fmhy.Source) {
			defer wg.Done()
			defer func() { <-sem }()
			results[i] = check(ctx, client, source, cfg)
		}(i, source)
	}
	wg.Wait()

	return results
}

func newClient(cfg Config) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = cfg.Workers * 2
	transport.MaxIdleConnsPerHost = cfg.Workers

	return &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

func check(ctx context.Context, client *http.Client, source fmhy.Source, cfg Config) Result {
	result := Result{Source: source, Tier: TierUnknown}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		result.Status = Dead
		result.Err = err.Error()
		return result
	}
	req.Header.Set("User-Agent", userAgent)
	// A partial read is enough to fingerprint a page and much kinder to sites.
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", cfg.MaxBytes-1))

	start := time.Now()
	resp, err := client.Do(req)
	result.Latency = time.Since(start).Round(time.Millisecond)
	if err != nil {
		result.Status = Dead
		result.Err = classifyError(err)
		return result
	}
	defer resp.Body.Close()

	result.Code = resp.StatusCode
	body, _ := io.ReadAll(io.LimitReader(resp.Body, cfg.MaxBytes))
	result.Tier = detectTier(body)

	switch {
	case result.Tier == TierCloudflare:
		result.Status = Blocked
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		result.Status = Alive
	case resp.StatusCode == http.StatusForbidden,
		resp.StatusCode == http.StatusTooManyRequests,
		resp.StatusCode >= 500:
		result.Status = Blocked
	default:
		result.Status = Dead
	}
	return result
}

func classifyError(err error) string {
	if os.IsTimeout(err) || strings.Contains(err.Error(), "deadline exceeded") {
		return "timeout"
	}
	return err.Error()
}

var tierMarkers = []struct {
	tier    Tier
	needles []string
}{
	{TierCloudflare, []string{"cf-chl", "__cf_chl", "just a moment", "challenge-platform", "cf-browser-verification"}},
	{TierMovieWeb, []string{"movie-web", "movieweb", "mw-backend"}},
	{TierPStream, []string{"p-stream", "pstream", "p_stream"}},
	{TierVidSrc, []string{"vidsrc", "vidlink.pro", "2embed", "embed.su"}},
}

// detectTier fingerprints the first 64 KiB of a homepage. Order matters: a
// Cloudflare challenge often contains fragments of other scripts, so it is
// checked first.
func detectTier(body []byte) Tier {
	lower := strings.ToLower(string(body))

	for _, group := range tierMarkers {
		for _, needle := range group.needles {
			if strings.Contains(lower, needle) {
				return group.tier
			}
		}
	}
	if strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype") {
		return TierPlain
	}
	return TierUnknown
}

// Snapshot is the on-disk form of a probe run.
type Snapshot struct {
	Taken   time.Time `json:"taken"`
	Results []Result  `json:"results"`
}

// Save writes results to path as JSON. A failure to cache is not fatal.
func Save(path string, results []Result) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(Snapshot{Taken: time.Now(), Results: results})
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load reads a snapshot that is no older than maxAge.
func Load(path string, maxAge time.Duration) ([]Result, bool) {
	if path == "" {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, false
	}
	if time.Since(snapshot.Taken) > maxAge {
		return nil, false
	}
	return snapshot.Results, true
}
