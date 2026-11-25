package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractStreamFromDirectURL(t *testing.T) {
	page := `<html><body><script>var src = "https://cdn.example.com/movie.m3u8?token=abc";</script></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(page))
	}))
	defer server.Close()

	// VidSrc builds its own URL, so we test extractStream directly.
	client := newClient()
	got := extractStream(context.Background(), client, page, server.URL, 0)
	if got != "https://cdn.example.com/movie.m3u8?token=abc" {
		t.Errorf("got %q", got)
	}
}

func TestExtractStreamFollowsIframe(t *testing.T) {
	inner := `<html><body><video src="https://cdn.example.com/stream.mp4"></video></body></html>`
	outer := `<html><body><iframe src="/player"></iframe></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/player" {
			w.Write([]byte(inner))
		} else {
			w.Write([]byte(outer))
		}
	}))
	defer server.Close()

	client := newClient()
	body, _ := fetchPage(context.Background(), client, server.URL+"/", 512<<10)
	got := extractStream(context.Background(), client, body, server.URL, 0)
	if got != "https://cdn.example.com/stream.mp4" {
		t.Errorf("got %q", got)
	}
}

func TestAdapterInterface(t *testing.T) {
	// compile-time check that all adapters implement Adapter
	var _ Adapter = NewVidSrc()
	var _ Adapter = NewTwoEmbed()
	var _ Adapter = NewVidLink()
}
