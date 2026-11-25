package cmd

import (
	"cmp"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
	"github.com/sakshamcozykun/kamui/internal/probe"
)

const probeTTL = 12 * time.Hour

// Probe implements "kamui probe".
func Probe(args []string) error {
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	section := fs.String("section", "", "only probe sections containing this text")
	refresh := fs.Bool("refresh", false, "ignore the cached results and check again")
	workers := fs.Int("workers", 16, "how many requests to make at once")
	timeout := fs.Duration("timeout", 10*time.Second, "per-site timeout, e.g. 5s")
	limit := fs.Int("max", 0, "stop after N sources (0 = all)")
	asJSON := fs.Bool("json", false, "print raw JSON instead of a table")
	if err := fs.Parse(args); err != nil {
		return err
	}

	all, err := loadSources()
	if err != nil {
		return err
	}
	sources := filter(all, *section, false)
	if *limit > 0 && len(sources) > *limit {
		sources = sources[:*limit]
	}
	if len(sources) == 0 {
		return fmt.Errorf("no sources matched")
	}

	results, fromCache, err := probeSources(sources, *refresh, *workers, *timeout)
	if err != nil {
		return err
	}

	orderResults(results)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	printProbe(results, fromCache)
	return nil
}

func probeSources(sources []fmhy.Source, refresh bool, workers int, timeout time.Duration) ([]probe.Result, bool, error) {
	path := ""
	if dir := cacheDir(); dir != "" {
		path = filepath.Join(dir, "probe.json")
	}

	if !refresh {
		if cached, ok := probe.Load(path, probeTTL); ok {
			return reuse(cached, sources), true, nil
		}
	}

	cfg := probe.DefaultConfig()
	cfg.Workers = workers
	cfg.Timeout = timeout

	results := probe.Run(context.Background(), sources, cfg)
	if err := probe.Save(path, results); err != nil {
		fmt.Fprintf(os.Stderr, "kamui: could not cache probe results: %v\n", err)
	}
	return results, false, nil
}

// reuse reattaches cached results to the current source list by host, so a
// freshly added site is not silently reported as fine.
func reuse(cached []probe.Result, sources []fmhy.Source) []probe.Result {
	byHost := make(map[string]probe.Result, len(cached))
	for _, r := range cached {
		byHost[r.Source.Host()] = r
	}

	out := make([]probe.Result, 0, len(sources))
	for _, s := range sources {
		r, ok := byHost[s.Host()]
		if !ok {
			r = probe.Result{Source: s, Status: probe.Unknown, Tier: probe.TierUnknown}
		}
		r.Source = s
		out = append(out, r)
	}
	return out
}

func orderResults(results []probe.Result) {
	rank := map[probe.Status]int{
		probe.Alive: 0, probe.Blocked: 1, probe.Dead: 2, probe.Unknown: 3,
	}
	slices.SortFunc(results, func(a, b probe.Result) int {
		if c := cmp.Compare(rank[a.Status], rank[b.Status]); c != 0 {
			return c
		}
		if c := cmp.Compare(string(a.Tier), string(b.Tier)); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Latency, b.Latency); c != 0 {
			return c
		}
		return cmp.Compare(a.Source.Order, b.Source.Order)
	})
}

func printProbe(results []probe.Result, fromCache bool) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "STATUS\tTIER\tCODE\tMS\tHOST\n")

	for _, r := range results {
		code := fmt.Sprintf("%d", r.Code)
		if r.Code == 0 {
			code = "-"
		}
		ms := fmt.Sprintf("%d", r.Latency.Milliseconds())
		if r.Err != "" {
			ms = r.Err
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.Status, r.Tier, code, ms, r.Source.Host())
	}
	_ = w.Flush()

	var alive, blocked, dead int
	for _, r := range results {
		switch r.Status {
		case probe.Alive:
			alive++
		case probe.Blocked:
			blocked++
		default:
			dead++
		}
	}
	byTier := map[probe.Tier]int{}
	for _, r := range results {
		if r.Status == probe.Alive {
			byTier[r.Tier]++
		}
	}

	parts := []string{
		fmt.Sprintf("alive %d", alive),
		fmt.Sprintf("blocked %d", blocked),
		fmt.Sprintf("dead/unknown %d", dead),
	}
	keys := make([]string, 0, len(byTier))
	for k := range byTier {
		keys = append(keys, string(k))
	}
	slices.Sort(keys)
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", k, byTier[probe.Tier(k)]))
	}

	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Join(parts, "  |  "))
	if fromCache {
		fmt.Fprintln(os.Stderr, "using cached results (--refresh to recheck)")
	}
}
