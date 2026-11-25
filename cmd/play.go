package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/player"
	"github.com/sakshamcozykun/kamui/internal/search"
)

var (
	playMax     int
	playWorkers int
	playTimeout time.Duration
	playPick    bool
	playDebug   bool
)

var playCmd = &cobra.Command{
	Use:   "play [title]",
	Short: "Search and play a title",
	RunE:  func(cmd *cobra.Command, args []string) error { return playQuery(strings.Join(args, " ")) },
}

func init() {
	rootCmd.AddCommand(playCmd)
	playCmd.Flags().IntVar(&playMax, "max", 0, "only use the first N sources (0 = all)")
	playCmd.Flags().IntVar(&playWorkers, "workers", 12, "how many sites to search at once")
	playCmd.Flags().DurationVar(&playTimeout, "timeout", 8*time.Second, "per-site timeout")
	playCmd.Flags().BoolVar(&playPick, "pick", false, "show a list and ask which one to play")
	playCmd.Flags().BoolVar(&playDebug, "debug", false, "show what each site did")
}

// playQuery is shared by "kamui play ..." and the default "kamui <title>".
func playQuery(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("usage: kamui [title]")
	}

	all, err := loadSources()
	if err != nil {
		return err
	}
	sources := all
	if playMax > 0 && len(sources) > playMax {
		sources = sources[:playMax]
	}

	alive, err := aliveSources(sources, false)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "searching %d live sites...\n", len(alive))

	cfg := search.DefaultConfig()
	cfg.Workers = playWorkers
	cfg.Timeout = playTimeout
	cfg.Limit = 10

	results, attempts := search.Run(context.Background(), alive, title, cfg)
	if playDebug {
		for _, a := range attempts {
			fmt.Fprintf(os.Stderr, "  %-28s code=%-4s anchors=%-4s matched=%-3s %s\n",
				a.Host, itoa(a.Code), itoa(a.Anchors), itoa(a.Matched), a.Err)
		}
	}
	if len(results) == 0 {
		return fmt.Errorf("no results for %q", title)
	}

	choice := 0
	if playPick {
		choice, err = pickResult(results)
		if err != nil {
			return err
		}
	}

	for i := choice; i < len(results); i++ {
		r := results[i]
		fmt.Fprintf(os.Stderr, "trying: %s — %s\n", r.Site, r.Title)

		streamURL, err := player.Extract(context.Background(), r.URL, 30*time.Second)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  could not extract: %v\n", err)
			continue
		}

		fmt.Fprintf(os.Stderr, "playing: %s\n", r.Title)
		if err := player.Play(context.Background(), streamURL, r.Title); err != nil {
			fmt.Fprintf(os.Stderr, "  playback failed: %v\n", err)
			continue
		}
		return nil
	}

	return fmt.Errorf("no playable stream found")
}

func pickResult(results []search.Result) (int, error) {
	fmt.Fprintln(os.Stderr, "\nresults:")
	for i, r := range results {
		fmt.Fprintf(os.Stderr, "  %d. %s — %s (%.2f)\n", i+1, r.Site, r.Title, r.Score)
	}
	fmt.Fprint(os.Stderr, "pick a number: ")

	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(results) {
		return 0, fmt.Errorf("invalid selection")
	}
	return n - 1, nil
}
