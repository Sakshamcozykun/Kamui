package search

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

func TestCandidateURLsHaveNoFormattingArtifacts(t *testing.T) {
	got := candidateURLs("https://site.example", "toy story 4")
	want := []string{
		"https://site.example/search?q=toy+story+4",
		"https://site.example/?s=toy+story+4",
		"https://site.example/search/toy-story-4",
		"https://site.example/search?keyword=toy+story+4",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRunFindsResultOnFakeSite(t *testing.T) {
	page := `<html><body>
		<a href="/home">Home</a>
		<a href="/movie/toy-story-4">Toy Story 4 (2019)</a>
		<a href="https://ads.example/buy">Buy now</a>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, page)
	}))
	defer server.Close()

	sources := []fmhy.Source{{Name: "FakeSite", URL: server.URL + "/", Starred: true}}
	cfg := DefaultConfig()
	cfg.Workers = 1

	results, attempts := Run(context.Background(), sources, "toy story 4", cfg)

	if len(results) != 1 {
		t.Fatalf("got %d results: %+v", len(results), results)
	}
	if results[0].Title != "Toy Story 4 (2019)" {
		t.Errorf("title = %q", results[0].Title)
	}
	if results[0].URL != server.URL+"/movie/toy-story-4" {
		t.Errorf("url = %q", results[0].URL)
	}
	if len(attempts) != 1 || attempts[0].Code != 200 || attempts[0].Matched != 1 {
		t.Errorf("attempt = %+v", attempts)
	}
}

func TestScoreMatchRejectsUnrelated(t *testing.T) {
	tokens := []string{"toy", "story", "4"}
	if _, ok := scoreMatch("Home", "/home", tokens, false); ok {
		t.Error("unrelated anchor was accepted")
	}
	if _, ok := scoreMatch("Toy Story 3 (2010)", "/movie/toy-story-3", tokens, true); ok {
		t.Error("partial match (Toy Story 3) was accepted")
	}
	score, ok := scoreMatch("Toy Story 4", "/movie/toy-story-4", tokens, true)
	if !ok || score < 1.5 {
		t.Errorf("score = %.2f, ok = %v", score, ok)
	}
}

func TestScoreMatchUsesWholeWords(t *testing.T) {
	if _, ok := scoreMatch("Red Sonja – Berserker (2014)", "/other-comics/red-sonja-berserker-2014/", []string{"berserk"}, false); ok {
		t.Error("berserk should not match berserker")
	}
	if _, ok := scoreMatch("Berserk", "/manga/berserk", []string{"berserk"}, false); !ok {
		t.Error("exact title should match")
	}
}

func TestCleanText(t *testing.T) {
	got := cleanText(`Toy <b>Story</b> 4 &amp; Beyond  \n`)
	if got != "Toy Story 4 & Beyond \\n" {
		t.Errorf("got %q", got)
	}
}

func TestCleanTextStripsTagsWithGtInAttributes(t *testing.T) {
	got := cleanText(`<span class="[&>svg]:size-3 group/badge">9.1 PG-13</span> Frieren`)
	if got != "9.1 PG-13 Frieren" {
		t.Errorf("got %q", got)
	}
}
