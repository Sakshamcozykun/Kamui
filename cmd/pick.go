package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/player"
	"github.com/sakshamcozykun/kamui/internal/search"
)

var (
	pickLang    string
	pickQuality string
)

// checkedResult pairs a search result with stream extraction outcome.
type checkedResult struct {
	r      search.Result
	stream string
	err    error
}

// selectCmd is the "select" subcommand.
var selectCmd = &cobra.Command{
	Use:   "select [title]",
	Short: "Choose a result and print its link to open in a browser",
	Long:  "Search available sources, choose a result, and print its page link for you to open in a browser. Use 'kamui play' to attempt playback in mpv.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSelect(strings.Join(args, " "))
	},
}

func init() {
	rootCmd.AddCommand(selectCmd)
	selectCmd.Flags().StringVar(&pickLang, "lang", "", "preferred language")
	selectCmd.Flags().StringVar(&pickQuality, "quality", "", "preferred quality")
}

// runSelect searches, lets user pick, asks quality/language, returns URL.
func runSelect(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("usage: kamui select <title>")
	}
	fmt.Fprintf(os.Stderr, "searching for %q...\n", title)
	all, err := loadSources()
	if err != nil {
		return err
	}

	// Search enough sources to find multiple providers across different sites.
	searchSources := all
	const maxSearchSources = 200
	if len(searchSources) > maxSearchSources {
		searchSources = searchSources[:maxSearchSources]
	}

	alive, err := aliveSources(searchSources, false)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "searching %d live sites...\n", len(alive))

	cfg := search.DefaultConfig()
	cfg.Workers = 12
	cfg.Timeout = 8 * time.Second
	cfg.Limit = 20
	results, _ := search.Run(context.Background(), alive, title, cfg)
	if len(results) == 0 {
		fmt.Fprintln(os.Stderr, "no matches yet; retrying sources with a 15s timeout...")
		cfg.Timeout = 15 * time.Second
		retryCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		results, _ = search.Run(retryCtx, searchSources, title, cfg)
		cancel()
	}
	if len(results) == 0 {
		return fmt.Errorf("no results for %q", title)
	}

	// Check stream extraction for top results to show confidence.
	// Results are already deduped by domain from search.Run.
	const maxChecks = 5
	checked := make([]checkedResult, len(results))
	var wg sync.WaitGroup
	for i := 0; i < len(results) && i < maxChecks; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := results[i]
			stream, err := player.Extract(context.Background(), r.URL, 15*time.Second)
			checked[i] = checkedResult{r: r, stream: stream, err: err}
		}(i)
	}
	wg.Wait()

	// Fill in remaining results without extraction (they show as "unchecked").
	for i := maxChecks; i < len(results); i++ {
		checked[i] = checkedResult{r: results[i], stream: "", err: fmt.Errorf("not checked")}
	}

	choice := pickResultWithConfidence(checked)
	if choice < 0 || choice >= len(checked) {
		return fmt.Errorf("invalid selection")
	}
	selected := checked[choice]

	fmt.Fprintf(os.Stderr, "\nYou selected: %d. %s — %s from %s\n", choice+1, selected.r.Site, selected.r.Title, selected.r.URL)
	if selected.err != nil && selected.err.Error() != "not checked" {
		fmt.Fprintf(os.Stderr, "  ⚠️  yt-dlp cannot extract (JavaScript site — open in browser)\n")
	} else if selected.stream != "" {
		fmt.Fprintf(os.Stderr, "  ✅ Direct stream URL found\n")
	} else {
		fmt.Fprintf(os.Stderr, "  ❓ Not checked — open in browser to verify\n")
	}

	if pickQuality == "" {
		fmt.Fprint(os.Stderr, "\nPreferred quality? (e.g. 720p, 1080p, 4k, leave blank for any): ")
		q, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		pickQuality = strings.TrimSpace(q)
	}
	if pickLang == "" {
		fmt.Fprint(os.Stderr, "Preferred language? (e.g. en, pt, es, leave blank for any): ")
		l, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		pickLang = strings.TrimSpace(l)
	}
	fmt.Fprintf(os.Stderr, "\n--- Selection Summary ---\n")
	fmt.Fprintf(os.Stderr, "Movie: %s\n", title)
	fmt.Fprintf(os.Stderr, "Selected result: %d. %s — %s from %s\n", choice+1, selected.r.Site, selected.r.Title, selected.r.URL)
	fmt.Fprintf(os.Stderr, "Preferred quality: %s\n", pickQuality)
	fmt.Fprintf(os.Stderr, "Preferred language: %s\n", pickLang)
	fmt.Fprintf(os.Stderr, "Stream URL: %s\n", selected.r.URL)
	if selected.stream != "" {
		fmt.Fprintf(os.Stderr, "Direct stream URL: %s\n", selected.stream)
	}
	fmt.Fprintf(os.Stderr, "\nOpen this link in your browser:\n%s\n", selected.r.URL)
	return nil
}

// pickResultWithConfidence displays results with extraction confidence indicators.
func pickResultWithConfidence(checked []checkedResult) int {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "SELECT  SITE        TITLE                           SCORE  STATUS\n")
	for i, c := range checked {
		status := "❓ unchecked"
		if c.err != nil && c.err.Error() != "not checked" {
			status = "⚠️ JS site"
		} else if c.stream != "" {
			status = "✅ stream"
		}
		fmt.Fprintf(w, "  %d.   %s    %s    %.2f   %s\n", i+1, c.r.Site, c.r.Title, c.r.Score, status)
	}
	_ = w.Flush()
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "\nEnter the number of your choice: ")
	line, _ := reader.ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(checked) {
		fmt.Fprintln(os.Stderr, "Invalid choice, showing first result.")
		return 0
	}
	return n - 1
}
