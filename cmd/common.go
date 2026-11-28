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

// animeSources keeps only the wiki's anime streaming sections. Other anime
// sections (downloads, torrents, tracking databases) are not playable sites,
// so they are skipped.
func animeSources(sources []fmhy.Source) []fmhy.Source {
	return sourcesMatching(sources, func(path string) bool {
		return strings.Contains(path, "anime streaming")
	})
}

// mangaSources keeps manga and comic reading sites, skipping readers, tools,
// trackers and recommendation lists because those do not host titles.
func mangaSources(sources []fmhy.Source) []fmhy.Source {
	return sourcesMatching(sources, func(path string) bool {
		if !strings.Contains(path, "manga") && !strings.Contains(path, "comics") {
			return false
		}
		for _, skip := range []string{"reader", "tool", "tracking", "recommendation", "database"} {
			if strings.Contains(path, skip) {
				return false
			}
		}
		return true
	})
}

// novelSources keeps light novel and web novel reading sites.
func novelSources(sources []fmhy.Source) []fmhy.Source {
	return sourcesMatching(sources, func(path string) bool {
		return strings.Contains(path, "light novels") || strings.Contains(path, "web novels")
	})
}

// bookSources keeps general ebook sites and public-domain/PDF libraries,
// skipping readers, tools and Kindle utilities.
func bookSources(sources []fmhy.Source) []fmhy.Source {
	return sourcesMatching(sources, func(path string) bool {
		if strings.Contains(path, "audiobook") || strings.Contains(path, "fanfiction") {
			return false
		}
		for _, skip := range []string{"reader", "tool", "kindle", "tracking", "recommendation"} {
			if strings.Contains(path, skip) {
				return false
			}
		}
		return strings.Contains(path, "ebook") ||
			strings.Contains(path, "public domain") ||
			strings.Contains(path, "pdf search") ||
			strings.Contains(path, "calibre")
	})
}

func sourcesMatching(sources []fmhy.Source, match func(path string) bool) []fmhy.Source {
	out := make([]fmhy.Source, 0, len(sources))
	for _, s := range sources {
		if match(strings.ToLower(strings.Join(s.Path, " "))) {
			out = append(out, s)
		}
	}
	return out
}

// loadSources fetches (or reads from cache) the video wiki and returns its
// deduplicated sources.
func loadSources() ([]fmhy.Source, error) {
	return loadDoc(fmhy.DocVideo)
}

// loadReadingSources fetches the books/comics wiki.
func loadReadingSources() ([]fmhy.Source, error) {
	return loadDoc(fmhy.DocReading)
}

func loadDoc(doc string) ([]fmhy.Source, error) {
	client := fmhy.NewClient(cacheDir())
	body, err := client.Fetch(context.Background(), doc)
	if err != nil {
		return nil, err
	}
	return fmhy.Dedupe(fmhy.Parse(string(body))), nil
}
