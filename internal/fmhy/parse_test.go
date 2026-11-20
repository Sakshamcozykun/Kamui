package fmhy

import (
	"os"
	"testing"
)

// loadFixture reads the miniature wiki used by the tests.
func loadFixture(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("testdata/video.md")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(b)
}

func TestParse(t *testing.T) {
	got := Parse(loadFixture(t))

	type want struct {
		name    string
		url     string
		section string
		starred bool
	}

	wants := []want{
		{"AggOne", "https://aggone.example/", "Stream Aggregators", true},
		{"AggTwo", "https://aggtwo.example/search", "Stream Aggregators", false},
		{"MultiOne", "http://multi1.example", "Multi-Server", true},
		{"MultiTwo", "https://multi2.example/", "Multi-Server", false},
		{"BoldSite", "https://bold.example/", "Multi-Server", false},
		{"JumpTarget", "https://external.example/", "External Section", false},
		{"TorrentOne", "https://torrent1.example/", "Anime Torrenting", false},
	}

	if len(got) != len(wants) {
		t.Fatalf("got %d sources, want %d\n%+v", len(got), len(wants), got)
	}

	for i, w := range wants {
		g := got[i]
		if g.Name != w.name {
			t.Errorf("[%d] Name = %q, want %q", i, g.Name, w.name)
		}
		if g.URL != w.url {
			t.Errorf("[%d] URL = %q, want %q", i, g.URL, w.url)
		}
		if g.Section() != w.section {
			t.Errorf("[%d] Section = %q, want %q", i, g.Section(), w.section)
		}
		if g.Starred != w.starred {
			t.Errorf("[%d] Starred = %v, want %v", i, g.Starred, w.starred)
		}
		if g.Order != i {
			t.Errorf("[%d] Order = %d, want %d", i, g.Order, i)
		}
	}
}

// The single most important parser property: a section must stop at the next
// heading of the same or a higher level. If this breaks, sources appear under
// the wrong section and the ranking goes wrong.
func TestParseSectionBoundaries(t *testing.T) {
	got := Parse(loadFixture(t))

	for _, s := range got {
		if s.Name == "TorrentOne" {
			if len(s.Path) != 2 || s.Path[0] != "Torrent Sites" {
				t.Fatalf("TorrentOne path = %v, want [Torrent Sites Anime Torrenting]", s.Path)
			}
			return
		}
	}
	t.Fatal("TorrentOne not found")
}

// Mirror links, numbered labels and metadata labels must never become sources.
func TestParseSkipsNoiseLinks(t *testing.T) {
	got := Parse(loadFixture(t))

	banned := map[string]bool{
		"https://aggmirror.example/":     true,
		"https://multi-alt.example/":     true,
		"https://multi3.example/":        true,
		"https://status.example/":        true,
		"https://multi2-mirror.example/": true,
		"https://discord.gg/example":     true,
	}
	for _, s := range got {
		if banned[s.URL] {
			t.Errorf("noise link became a source: %s (%s)", s.Name, s.URL)
		}
	}
}

func TestHost(t *testing.T) {
	cases := map[string]string{
		"https://www.example.com/path": "example.com",
		"http://example.com":           "example.com",
		"https://EXAMPLE.com":          "example.com",
		"https://sub.example.co.uk/x":  "sub.example.co.uk",
	}
	for in, want := range cases {
		if got := (Source{URL: in}).Host(); got != want {
			t.Errorf("Host(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDedupe(t *testing.T) {
	in := []Source{
		{Name: "A", URL: "https://a.example/"},
		{Name: "B", URL: "https://b.example/"},
		{Name: "A again", URL: "https://www.a.example/other"},
	}
	got := Dedupe(in)

	if len(got) != 2 {
		t.Fatalf("got %d sources, want 2: %+v", len(got), got)
	}
	if got[0].Name != "A" {
		t.Errorf("kept %q, want the first entry %q", got[0].Name, "A")
	}
}
