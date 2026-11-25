package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
	"github.com/sakshamcozykun/kamui/internal/probe"
	"github.com/sakshamcozykun/kamui/internal/search"
)

var (
	searchSection string
	searchRefresh bool
	searchWorkers int
	searchTimeout time.Duration
	searchLimit   int
	searchMax     int
	searchDebug   bool
	searchJSON    bool
)

var searchCmd = &cobra.Command{
	Use:   "search [flags] <title>",
	Short: "Search every live FMHY site in parallel",
	RunE:  runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().StringVar(&searchSection, "section", "", "only search sections containing this text")
	searchCmd.Flags().BoolVar(&searchRefresh, "refresh", false, "re-probe sites before searching")
	searchCmd.Flags().IntVar(&searchWorkers, "workers", 32, "how many sites to search at once")
	searchCmd.Flags().DurationVar(&searchTimeout, "timeout", 3*time.Second, "per-request timeout")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 25, "maximum results to show")
	searchCmd.Flags().IntVar(&searchMax, "max", 0, "only use the first N sources (0 = all)")
	searchCmd.Flags().BoolVar(&searchDebug, "debug", false, "show what each site did")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "print raw JSON instead of a table")
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		return fmt.Errorf("usage: kamui search [flags] <title>")
	}

	fmt.Fprintf(os.Stderr, "loading FMHY sources for %q...\n", query)
	all, err := loadSources()
	if err != nil {
		return err
	}
	sources := filter(all, searchSection, false)
	if searchMax > 0 && len(sources) > searchMax {
		sources = sources[:searchMax]
	}
	if len(sources) == 0 {
		return fmt.Errorf("no sources matched")
	}

	fmt.Fprintf(os.Stderr, "checking %d sources for availability...\n", len(sources))
	alive, err := aliveSources(sources, searchRefresh)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "searching %d live sites...\n", len(alive))

	cfg := search.DefaultConfig()
	cfg.Workers = searchWorkers
	cfg.Timeout = searchTimeout
	cfg.Limit = searchLimit
	cfg.OnProgress = func(done, total int) {
		if done == total || done%5 == 0 {
			fmt.Fprintf(os.Stderr, "search progress: %d/%d sites checked\n", done, total)
		}
	}

	fmt.Fprintf(os.Stderr, "searching %d live sites for %q...\n", len(alive), query)
	searchCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	results, attempts := search.Run(searchCtx, alive, query, cfg)

	if searchDebug {
		for _, a := range attempts {
			fmt.Fprintf(os.Stderr, "  %-28s code=%-4s anchors=%-4s matched=%-3s %s\n",
				a.Host, itoa(a.Code), itoa(a.Anchors), itoa(a.Matched), a.Err)
		}
	}

	if searchJSON {
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
