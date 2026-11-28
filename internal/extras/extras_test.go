package extras

import (
	"strings"
	"testing"
)

func TestSourcesHaveUniqueHosts(t *testing.T) {
	seen := make(map[string]string)
	for _, s := range Sources() {
		host := s.Host()
		if host == "" {
			t.Errorf("%q has an invalid URL %q", s.Name, s.URL)
			continue
		}
		if prev, ok := seen[host]; ok {
			t.Errorf("%q and %q share host %s", prev, s.Name, host)
		}
		seen[host] = s.Name
	}
}

func TestSourcesUseCataloguePaths(t *testing.T) {
	want := map[string]bool{
		"streaming sites": false,
		"anime streaming": false,
		"manga":           false,
		"light novels":    false,
		"ebooks":          false,
	}
	for _, s := range Sources() {
		if !IsExtra(s) {
			t.Errorf("%q is missing the %q path prefix", s.Name, Root)
		}
		want[strings.ToLower(s.Section())] = true
	}
	for section, found := range want {
		if !found {
			t.Errorf("no sources for section %q", section)
		}
	}
}
