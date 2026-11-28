package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
	"github.com/sakshamcozykun/kamui/internal/search"
)

var (
	readingRefresh bool
	readingWorkers int
	readingTimeout time.Duration
	readingLimit   int
	readingMax     int
	readingDebug   bool
	readingJSON    bool
)

var mangaCmd = &cobra.Command{
	Use:     "manga [flags] <title>",
	Aliases: []string{"mangas"},
	Short:   "Search manga and comic sources",
	Long: `kamui manga searches only manga and comic reading sources.

It uses the same search engine as "kamui search", restricted to the wiki's
Manga and Comics sections. Results are deduplicated per provider.

Examples:
  kamui manga "berserk"
  kamui manga "one punch man" --limit 10
  kamui manga "vinland saga" --refresh`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReadingSearch("manga", mangaSources, strings.Join(args, " "))
	},
}

var bookCmd = &cobra.Command{
	Use:     "book [flags] <title>",
	Aliases: []string{"books"},
	Short:   "Search ebook sources",
	Long: `kamui book searches ebook libraries and public-domain/PDF book sources.

It is restricted to the wiki's Ebooks sections and skips readers, tools and
tracking databases. Use "kamui novel" for light novels and web novels.

Examples:
  kamui book "dune"
  kamui book "the hobbit" --limit 10
  kamui book "1984" --refresh`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReadingSearch("book", bookSources, strings.Join(args, " "))
	},
}

var novelCmd = &cobra.Command{
	Use:     "novel [flags] <title>",
	Aliases: []string{"novels", "lightnovel", "lightnovels"},
	Short:   "Search light novel and web novel sources",
	Long: `kamui novel searches only light novel and web novel reading sources.

It uses the wiki's Light Novels section, which covers Japanese light novels,
Chinese web novels and Korean web novels.

Examples:
  kamui novel "overlord"
  kamui novel "rezero" --limit 10
  kamui novel "shadow slave" --refresh`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReadingSearch("novel", novelSources, strings.Join(args, " "))
	},
}

func init() {
	for _, cmd := range []*cobra.Command{mangaCmd, bookCmd, novelCmd} {
		rootCmd.AddCommand(cmd)
		cmd.Flags().BoolVar(&readingRefresh, "refresh", false, "re-probe sites before searching")
		cmd.Flags().IntVar(&readingWorkers, "workers", 16, "how many sites to search at once")
		cmd.Flags().DurationVar(&readingTimeout, "timeout", 8*time.Second, "per-request timeout")
		cmd.Flags().IntVar(&readingLimit, "limit", 25, "maximum results to show")
		cmd.Flags().IntVar(&readingMax, "max", 80, "only use the first N wiki sources (other lists are always included; 0 = all)")
		cmd.Flags().BoolVar(&readingDebug, "debug", false, "show what each site did")
		cmd.Flags().BoolVar(&readingJSON, "json", false, "print raw JSON instead of a table")
	}
}

// runReadingSearch is shared by the manga, book and novel commands.
func runReadingSearch(label string, selectSources func([]fmhy.Source) []fmhy.Source, query string) error {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("usage: kamui %s [flags] <title>", label)
	}

	fmt.Fprintf(os.Stderr, "loading %s sources for %q...\n", label, query)
	all, err := loadReadingSources()
	if err != nil {
		return err
	}
	sources := selectSources(all)
	if len(sources) == 0 {
		return fmt.Errorf("no %s sources found in the catalogue", label)
	}
	sources = limitSources(sources, readingMax)

	fmt.Fprintf(os.Stderr, "checking %d %s sources for availability...\n", len(sources), label)
	alive, err := aliveSources(sources, readingRefresh)
	if err != nil {
		return err
	}
	if len(alive) == 0 {
		return fmt.Errorf("no reachable %s sources; try 'kamui %s %q --refresh'", label, label, query)
	}

	cfg := search.DefaultConfig()
	cfg.Workers = readingWorkers
	cfg.Timeout = readingTimeout
	cfg.Limit = readingLimit
	cfg.OnProgress = func(done, total int) {
		if done == total || done%5 == 0 {
			fmt.Fprintf(os.Stderr, "search progress: %d/%d %s sites checked\n", done, total, label)
		}
	}

	fmt.Fprintf(os.Stderr, "searching %d live %s sites for %q...\n", len(alive), label, query)
	searchCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	results, attempts := search.Run(searchCtx, alive, query, cfg)

	if readingDebug {
		for _, a := range attempts {
			fmt.Fprintf(os.Stderr, "  %-28s code=%-4s anchors=%-4s matched=%-3s %s\n",
				a.Host, itoa(a.Code), itoa(a.Anchors), itoa(a.Matched), a.Err)
		}
	}

	results = filterReadingResults(results)

	if readingJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}
	if len(results) == 0 {
		return fmt.Errorf("no %s results for %q; try the full title or '--refresh'", label, query)
	}

	printSearch(results)
	return nil
}

// nonContentHosts are platforms that host discussions, code or social posts,
// not readable books, manga or novels.
var nonContentHosts = []string{
	"github.com", "gitlab.com", "reddit.com", "wikipedia.org",
	"discord.com", "discord.gg", "t.me", "telegram.me",
	"youtube.com", "youtu.be", "x.com", "twitter.com",
	"facebook.com", "instagram.com",
}

func filterReadingResults(results []search.Result) []search.Result {
	out := results[:0]
	for _, r := range results {
		u, err := url.Parse(r.URL)
		if err != nil {
			continue
		}
		host := strings.ToLower(u.Hostname())
		skip := false
		for _, bad := range nonContentHosts {
			if host == bad || strings.HasSuffix(host, "."+bad) {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, r)
		}
	}
	return out
}
