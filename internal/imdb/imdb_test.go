package imdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchParsesResults(t *testing.T) {
	sample := map[string]any{
		"d": []map[string]any{
			{"id": "tt1979376", "l": "Toy Story 4", "y": 2019, "q": "feature",
				"i": map[string]string{"imageUrl": "https://example.com/poster.jpg"}},
			{"id": "nm0000158", "l": "Tom Hanks", "q": "name"}, // should be filtered out
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(sample)
	}))
	defer server.Close()

	endpoint = server.URL
	defer func() { endpoint = "https://v2.sg.media-imdb.com/suggestion" }()

	client := NewClient()
	movies, err := client.Search(context.Background(), "toy story 4")
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 1 {
		t.Fatalf("got %d movies, want 1: %+v", len(movies), movies)
	}
	m := movies[0]
	if m.ID != "tt1979376" || m.Title != "Toy Story 4" || m.Year != 2019 {
		t.Errorf("got %+v", m)
	}
	if m.Poster == "" {
		t.Error("poster not parsed")
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	client := NewClient()
	if _, err := client.Search(context.Background(), "   "); err == nil {
		t.Error("expected error for empty query")
	}
}
