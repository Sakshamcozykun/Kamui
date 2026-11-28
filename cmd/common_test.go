package cmd

import (
	"testing"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

func source(path ...string) fmhy.Source {
	return fmhy.Source{Name: path[len(path)-1], URL: "https://example.com/", Path: path}
}

func names(sources []fmhy.Source) []string {
	out := make([]string, len(sources))
	for i, s := range sources {
		out[i] = s.Name
	}
	return out
}

func equalNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCategorySelectors(t *testing.T) {
	all := []fmhy.Source{
		source("Specialty Streaming", "Anime Streaming"),
		source("Anime Downloading"),
		source("Visual Media", "Comics"),
		source("Visual Media", "Manga"),
		source("Visual Media", "Manga Readers / Tools"),
		source("Book Tracking / Databases", "Comic Tracking / Databases"),
		source("Ebooks"),
		source("Ebooks", "Ebook Readers"),
		source("Ebooks", "Calibre Libraries"),
		source("Special Interest", "Light Novels"),
		source("Audiobooks", "Streaming"),
	}

	tests := []struct {
		name string
		got  []fmhy.Source
		want []string
	}{
		{"anime", animeSources(all), []string{"Anime Streaming"}},
		{"manga", mangaSources(all), []string{"Comics", "Manga"}},
		{"novel", novelSources(all), []string{"Light Novels"}},
		{"book", bookSources(all), []string{"Ebooks", "Calibre Libraries"}},
	}
	for _, tc := range tests {
		if got := names(tc.got); !equalNames(got, tc.want) {
			t.Errorf("%s sources = %v, want %v", tc.name, got, tc.want)
		}
	}
}
