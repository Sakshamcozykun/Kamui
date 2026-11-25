package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

const probeTTL = 12 * time.Hour

// cacheDir returns ~/.cache/kamui on Linux, or "" if the OS will not say,
// which disables caching.
func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "kamui")
}

// filter keeps the sources the user asked for. Kept as its own function so it
// can be tested and reused by the probe and search commands.
func filter(sources []fmhy.Source, section string, starredOnly bool) []fmhy.Source {
	section = strings.ToLower(section)

	out := make([]fmhy.Source, 0, len(sources))
	for _, s := range sources {
		if starredOnly && !s.Starred {
			continue
		}
		if section != "" && !strings.Contains(strings.ToLower(strings.Join(s.Path, " ")), section) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// loadSources fetches (or reads from cache) the video wiki and returns its
// deduplicated sources.
func loadSources() ([]fmhy.Source, error) {
	client := fmhy.NewClient(cacheDir())
	body, err := client.Fetch(context.Background(), fmhy.DocVideo)
	if err != nil {
		return nil, err
	}
	return fmhy.Dedupe(fmhy.Parse(string(body))), nil
}
