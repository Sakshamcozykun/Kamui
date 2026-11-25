package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/search"
	"github.com/sakshamcozykun/kamui/internal/player"
)

var (
	pickLang    string
	pickQuality string
)

// selectCmd is the "select" subcommand.
var selectCmd = &cobra.Command{
	Use:   "select [title]",
	Short: "Search and choose a movie result",
	Long:  "Search FMHY sites and pick a result by number",
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
	cfg := search.DefaultConfig()
	cfg.Workers = 12
	cfg.Timeout = 8 * time.Second
	cfg.Limit = 20
	results, _ := search.Run(context.Background(), all, title, cfg)
	if len(results) == 0 {
		return fmt.Errorf("no results for %q", title)
	}
	choice := pickResult(results)
	if choice < 0 || choice >= len(results) {
		return fmt.Errorf("invalid selection")
	}
	selected := results[choice]
	fmt.Fprintf(os.Stderr, "\nYou selected: %d. %s — %s from %s\n", choice+1, selected.Site, selected.Title, selected.URL)
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
	fmt.Fprintf(os.Stderr, "Selected result: %d. %s — %s from %s\n", choice+1, selected.Site, selected.Title, selected.URL)
	fmt.Fprintf(os.Stderr, "Preferred quality: %s\n", pickQuality)
	fmt.Fprintf(os.Stderr, "Preferred language: %s\n", pickLang)
	fmt.Fprintf(os.Stderr, "Stream URL: %s\n", selected.URL)
	streamURL, err := player.Extract(context.Background(), selected.URL, 30*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not extract stream: %v\n", err)
		fmt.Fprintf(os.Stderr, "URL: %s\n", selected.URL)
	} else {
		fmt.Fprintf(os.Stderr, "Extracted stream URL: %s\n", streamURL)
	}
	return nil
}

func pickResult(results []search.Result) int {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "SELECT  SITE        TITLE                           SCORE\n")
	for i, r := range results {
		fmt.Fprintf(w, "  %d.   %s    %s    %.2f\n", i+1, r.Site, r.Title, r.Score)
	}
	_ = w.Flush()
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "\nEnter the number of your choice: ")
	line, _ := reader.ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(results) {
		fmt.Fprintln(os.Stderr, "Invalid choice, showing first result.")
		return 0
	}
	return n - 1
}