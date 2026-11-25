package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/adapter"
	"github.com/sakshamcozykun/kamui/internal/imdb"
	"github.com/sakshamcozykun/kamui/internal/player"
	"github.com/sakshamcozykun/kamui/internal/search"
)

// playCmd is the "play" subcommand.
var playCmd = &cobra.Command{
	Use:   "play [title]",
	Short: "Search and play a title from FMHY-listed sites",
	Long: `kamui play searches live FMHY sites and tries to play the result in mpv.

If a TMDB/API key is configured, the title is first resolved to an IMDB ID,
then embed providers (VidSrc, 2Embed, VidLink) are tried in parallel.

If no key is configured, the IMDB suggestion API (no key required) is used.

Examples:
  kamui play "toy story 4"
  kamui play --pick "toy story 4"   # show list and choose
  kamui --max 10 play "dune part two"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return playQuery(strings.Join(args, " "))
	},
}

func init() {
	rootCmd.AddCommand(playCmd)
	playCmd.Flags().IntVar(&playMax, "max", playMax, "only use the first N sources (0 = all)")
	playCmd.Flags().IntVar(&playWorkers, "workers", playWorkers, "how many sites to search at once")
	playCmd.Flags().DurationVar(&playTimeout, "timeout", playTimeout, "per-site timeout")
	playCmd.Flags().BoolVar(&playPick, "pick", playPick, "show a list and ask which one to play")
	playCmd.Flags().BoolVar(&playDebug, "debug", playDebug, "show what each site did")
}

// playQuery is shared by "kamui play ..." and the default "kamui <title>".
func playQuery(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("usage: kamui [title]")
	}

	// Step 1: resolve title to an IMDB ID via IMDB's public suggestion API.
	fmt.Fprintf(os.Stderr, "looking up %q...\n", title)
	imdbClient := imdb.NewClient()
	movies, err := imdbClient.Search(context.Background(), title)
	if err != nil {
		return fmt.Errorf("imdb lookup failed: %w", err)
	}
	if len(movies) == 0 {
		return fmt.Errorf("no movie found for %q", title)
	}

	movie := movies[0]
	_ = imdbClient.Details(context.Background(), &movie)
	printMovieCard(movie)
	if len(movies) > 1 {
		movie, err = chooseMovie(movies)
		if err != nil {
			return err
		}
		_ = imdbClient.Details(context.Background(), &movie)
		printMovieCard(movie)
	}

	imdbID := movie.ID

	// Step 2: try embed providers in parallel.
	// Adapters implement the Adapter interface; each takes an IMDB ID and
	// returns a stream URL (or an error if the site can't provide one).
	adapters := []adapter.Adapter{
		adapter.NewVidSrc(),
		adapter.NewTwoEmbed(),
		adapter.NewVidLink(),
	}

	fmt.Fprintf(os.Stderr, "trying %d providers for %s...\n", len(adapters), imdbID)

	type result struct {
		name string
		url  string
		err  error
	}
	resCh := make(chan result, len(adapters))

	var wg sync.WaitGroup
	for _, a := range adapters {
		wg.Add(1)
		go func(a adapter.Adapter) {
			defer wg.Done()
			providerCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			u, err := a.Extract(providerCtx, imdbID)
			cancel()
			resCh <- result{name: a.Name(), url: u, err: err}
		}(a)
	}

	// Collect results as they arrive; play the first success.
	go func() {
		wg.Wait()
		close(resCh)
	}()

	for r := range resCh {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "  %s: %v\n", r.name, r.err)
			continue
		}
		fmt.Fprintf(os.Stderr, "  %s: found stream\n", r.name)

		// Try to play the stream with mpv.
		if err := player.Play(context.Background(), r.url, movie.Title); err != nil {
			fmt.Fprintf(os.Stderr, "  playback failed: %v\n", err)
			// If this adapter failed, continue to the next one.
			continue
		}
		fmt.Fprintf(os.Stderr, "playing: %s (%d)\n", movie.Title, movie.Year)
		return nil
	}

	// Step 3: all adapters failed – fall back to generic search.
	fmt.Fprintln(os.Stderr, "\nall providers failed; falling back to generic search...")

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

	searchResults, _ := search.Run(context.Background(), alive, title, cfg)
	if len(searchResults) == 0 {
		return fmt.Errorf("no playable stream found")
	}

	for i := 0; i < len(searchResults); i++ {
		r := searchResults[i]
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

func chooseMovie(movies []imdb.Movie) (imdb.Movie, error) {
	fmt.Fprintf(os.Stderr, "Found %d matches; choose the movie (0 to cancel):\n", len(movies))
	for i, m := range movies {
		fmt.Fprintf(os.Stderr, "  %d. %s (%d) %s\n", i+1, m.Title, m.Year, m.ID)
	}
	fmt.Fprint(os.Stderr, "Choice: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return imdb.Movie{}, fmt.Errorf("read movie choice: %w", err)
	}
	choice, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || choice < 1 || choice > len(movies) {
		return imdb.Movie{}, fmt.Errorf("movie selection cancelled or invalid")
	}
	return movies[choice-1], nil
}
