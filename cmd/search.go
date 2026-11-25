package cmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
	"github.com/sakshamcozykun/kamui/internal/probe"
	"github.com/sakshamcozykun/kamui/internal/search"
)

// Search implements "kamui search".
func Search(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	section := fs.String("section", "", "only search sections containing this text")
	refresh := fs.Bool("refresh", false, "re-probe sites before searching")
	workers := fs.Int("workers", 12, "how many sites to search at once")
	timeout := fs.Duration("timeout", 8*time.Second, "per-site timeout")
	limit := fs.Int("limit", 25, "maximum results to show")
	maxSources := fs.Int("max", 0, "only use the first N sources (0 = all)")
	debug := fs.Bool("debug", false, "show what each site did")
	asJSON := fs.Bool("json", false, "print raw JSON instead of a table")
	if err := fs.Parse(args); err != nil {
		return err
	}

	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return fmt.Errorf("usage: kamui search [flags] <title>")
	}

	all, err := loadSources()
	if err != nil {
		return err
	}
	sources := filter(all, *section, false)
	if *maxSources > 0 && len(sources) > *maxSources {
		sources = sources[:*maxSources]
	}
	if len(sources) == 0 {
		return fmt.Errorf("no sources matched")
	}

	alive, err := aliveSources(sources, *refresh)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "searching %d live sites...\n", len(alive))

	cfg := search.DefaultConfig()
	cfg.Workers = *workers
	cfg.Timeout = *timeout
	cfg.Limit = *limit

	results, attempts := search.Run(context.Background(), alive, query, cfg)

	if *debug {
		for _, a := range attempts {
			fmt.Fprintf(os.Stderr, "  %-28s code=%-4s anchors=%-4s matched=%-3s %s\n",
				a.Host, itoa(a.Code), itoa(a.Anchors), itoa(a.Matched), a.Err)
		}
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	if len(results) == 0 {
		return fmt.Errorf("no results for %q", query)
	}

	printSearch(results)
	return nil
}

// aliveSources reuses the probe cache, or runs a probe when there is none.
func aliveSources(sources []fmhy.Source, refresh bool) ([]fmhy.Source, error) {
	path := filepath.Join(cacheDir(), "probe.json")

	results, ok := probe.Load(path, probeTTL)
	if !ok || refresh {
		cfg := probe.DefaultConfig()
		results = probe.Run(context.Background(), sources, cfg)
		if err := probe.Save(path, results); err != nil {
			fmt.Fprintf(os.Stderr, "kamui: could not cache probe results: %v\n", err)
		}
	}

	live := make(map[string]bool, len(results))
	for _, r := range results {
		if r.Status == probe.Alive {
			live[r.Source.Host()] = true
		}
	}

	out := make([]fmhy.Source, 0, len(live))
	for _, s := range sources {
		if live[s.Host()] {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no live sources found; run 'kamui probe --refresh' first")
	}
	return out, nil
}

func printSearch(results []search.Result) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "SCORE\tSITE\tTITLE\tURL\n")
	for _, r := range results {
		fmt.Fprintf(w, "%.2f\t%s\t%s\t%s\n", r.Score, r.Site, r.Title, r.URL)
	}
	_ = w.Flush()
}

func itoa(n int) string {
	if n == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", n)
}
