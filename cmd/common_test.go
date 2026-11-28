package cmd

import (
	"testing"

	"github.com/sakshamcozykun/kamui/internal/extras"
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

func TestOtherListsAreMergedWithoutDuplicateHosts(t *testing.T) {
	// MangaDex is in the extra catalogue. A wiki entry for the same host
	// should win, so it is not listed twice.
	base := []fmhy.Source{{"WikiManga", "https://mangadex.org/", []string{"Visual Media", "Manga"}, false, 1}, source("Manga")}
	merged := appendOtherLists(base)

	count := 0
	for _, s := range merged {
		if s.Host() == "mangadex.org" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("mangadex.org appears %d times, want 1", count)
	}
	if merged[0].Name != "WikiManga" {
		t.Errorf("wiki entry was not kept first: %q", merged[0].Name)
	}
}

func TestLimitSourcesKeepsOtherLists(t *testing.T) {
	sources := []fmhy.Source{
		source("Anime Streaming"),
		source("Anime Streaming"),
		source("Anime Streaming"),
	}
	sources = append(sources, animeSources(extras.Sources())[:2]...)

	got := limitSources(sources, 1)
	if len(got) != 3 {
		t.Fatalf("got %d sources, want 1 wiki + 2 extras", len(got))
	}
	if got[0].Name != sources[0].Name {
		t.Errorf("first source = %q, want wiki source", got[0].Name)
	}
	for _, s := range got[1:] {
		if !extras.IsExtra(s) {
			t.Errorf("source %q should be from another list", s.Name)
		}
	}
}

func TestCategorySelectorsIncludeOtherLists(t *testing.T) {
	all := extras.Sources()
	tests := []struct {
		name string
		got  []fmhy.Source
	}{
		{"anime", animeSources(all)},
		{"manga", mangaSources(all)},
		{"novel", novelSources(all)},
		{"book", bookSources(all)},
	}
	for _, tc := range tests {
		if len(tc.got) == 0 {
			t.Errorf("%s selector found no sources from other lists", tc.name)
		}
		for _, s := range tc.got {
			if !extras.IsExtra(s) {
				t.Errorf("%s selector returned a wiki source: %q", tc.name, s.Name)
			}
		}
	}
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
