package probe

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

func TestDetectTier(t *testing.T) {
	cases := map[string]Tier{
		`<html><script src="movie-web.js"></script></html>`: TierMovieWeb,
		`<html>P-Stream player</html>`:                      TierPStream,
		`<html><body>Just a moment...</body></html>`:        TierCloudflare,
		`<html><body>hello</body></html>`:                   TierPlain,
		`garbage`:                                           TierUnknown,
	}
	for body, want := range cases {
		if got := detectTier([]byte(body)); got != want {
			t.Errorf("detectTier(%q) = %q, want %q", body, got, want)
		}
	}
}

func TestCheckClassifiesResponses(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html>movie-web homepage</html>`)
	}))
	defer ok.Close()

	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `<html>Just a moment...</html>`)
	}))
	defer blocked.Close()

	cfg := DefaultConfig()
	cfg.Timeout = 2 * time.Second
	client := newClient(cfg)

	alive := check(context.Background(), client, fmhy.Source{URL: ok.URL}, cfg)
	if alive.Status != Alive || alive.Tier != TierMovieWeb || alive.Code != 200 {
		t.Errorf("alive: %+v", alive)
	}

	cf := check(context.Background(), client, fmhy.Source{URL: blocked.URL}, cfg)
	if cf.Status != Blocked || cf.Tier != TierCloudflare {
		t.Errorf("blocked: %+v", cf)
	}
}

func TestRunPreservesOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html>ok</html>`)
	}))
	defer server.Close()

	sources := []fmhy.Source{
		{Name: "first", URL: server.URL + "/one"},
		{Name: "second", URL: server.URL + "/two"},
		{Name: "third", URL: server.URL + "/three"},
	}

	cfg := DefaultConfig()
	cfg.Workers = 3
	results := Run(context.Background(), sources, cfg)

	if len(results) != 3 {
		t.Fatalf("got %d results", len(results))
	}
	for i, name := range []string{"first", "second", "third"} {
		if results[i].Source.Name != name {
			t.Errorf("[%d] = %s, want %s", i, results[i].Source.Name, name)
		}
	}
}

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "probe.json")
	results := []Result{{Source: fmhy.Source{Name: "x", URL: "https://x.example/"}, Status: Alive, Tier: TierPlain}}

	if err := Save(path, results); err != nil {
		t.Fatal(err)
	}
	got, ok := Load(path, time.Hour)
	if !ok {
		t.Fatal("Load reported no cache")
	}
	if len(got) != 1 || got[0].Source.Name != "x" {
		t.Errorf("got %+v", got)
	}

	if _, ok := Load(path, -time.Hour); ok {
		t.Error("stale cache was accepted")
	}
}
