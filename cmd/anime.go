package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/search"
)

var (
	animeRefresh bool
	animeWorkers int
	animeTimeout time.Duration
	animeLimit   int
	animeMax     int
	animeDebug   bool
	animeJSON    bool
)

var animeCmd = &cobra.Command{
	Use:   "anime [flags] <title>",
	Short: "Search anime streaming sources",
	Long: `kamui anime searches only anime streaming sources from the catalogue.

It is the same search engine as "kamui search", restricted to the wiki's
anime streaming section (including donghua sites). Results come from
different providers and each link opens the title's page on that site.

Examples:
  kamui anime "frieren"
  kamui anime "one piece" --limit 10
  kamui anime "jujutsu kaisen" --refresh`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAnime(strings.Join(args, " "))
	},
}

func init() {
	rootCmd.AddCommand(animeCmd)
	animeCmd.Flags().BoolVar(&animeRefresh, "refresh", false, "re-probe anime sites before searching")
	animeCmd.Flags().IntVar(&animeWorkers, "workers", 16, "how many anime sites to search at once")
	animeCmd.Flags().DurationVar(&animeTimeout, "timeout", 8*time.Second, "per-request timeout")
	animeCmd.Flags().IntVar(&animeLimit, "limit", 25, "maximum results to show")
	animeCmd.Flags().IntVar(&animeMax, "max", 60, "only use the first N wiki anime sources (other lists are always included; 0 = all)")
	animeCmd.Flags().BoolVar(&animeDebug, "debug", false, "show what each anime site did")
	animeCmd.Flags().BoolVar(&animeJSON, "json", false, "print raw JSON instead of a table")
}

func runAnime(query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("usage: kamui anime [flags] <title>")
	}

	fmt.Fprintf(os.Stderr, "loading anime sources for %q...\n", query)
	all, err := loadSources()
	if err != nil {
		return err
	}
	sources := animeSources(all)
	if len(sources) == 0 {
		return fmt.Errorf("no anime sources found in the catalogue")
	}
	sources = limitSources(sources, animeMax)

	fmt.Fprintf(os.Stderr, "checking %d anime sources for availability...\n", len(sources))
	alive, err := aliveSources(sources, animeRefresh)
	if err != nil {
		return err
	}
	if len(alive) == 0 {
		return fmt.Errorf("no reachable anime sources; try 'kamui anime %q --refresh'", query)
	}

	cfg := search.DefaultConfig()
	cfg.Workers = animeWorkers
	cfg.Timeout = animeTimeout
	cfg.Limit = animeLimit
	cfg.OnProgress = func(done, total int) {
		if done == total || done%5 == 0 {
			fmt.Fprintf(os.Stderr, "search progress: %d/%d anime sites checked\n", done, total)
		}
	}

	fmt.Fprintf(os.Stderr, "searching %d live anime sites for %q...\n", len(alive), query)
	searchCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	results, attempts := search.Run(searchCtx, alive, query, cfg)

	if animeDebug {
		for _, a := range attempts {
			fmt.Fprintf(os.Stderr, "  %-28s code=%-4s anchors=%-4s matched=%-3s %s\n",
				a.Host, itoa(a.Code), itoa(a.Anchors), itoa(a.Matched), a.Err)
		}
	}

	if animeJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	if len(results) == 0 {
		return fmt.Errorf("no anime results for %q; try the full title or '--refresh'", query)
	}

	printSearch(results)
	return nil
}
