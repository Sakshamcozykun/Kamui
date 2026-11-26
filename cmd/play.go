package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/adapter"
	"github.com/sakshamcozykun/kamui/internal/imdb"
	"github.com/sakshamcozykun/kamui/internal/library"
	"github.com/sakshamcozykun/kamui/internal/player"
	"github.com/sakshamcozykun/kamui/internal/search"
)

// playCmd is the "play" subcommand.
var playCmd = &cobra.Command{
	Use:   "play [title]",
	Short: "Search and play a title from available sources",
	Long: `kamui play searches reachable sources and tries to play the result in mpv.

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
	recordHistory(title)

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
	if err := imdbClient.Details(context.Background(), &movie); err != nil {
		fmt.Fprintf(os.Stderr, "IMDb metadata unavailable: %v\n", err)
	}
	printMovieCard(movie)
	if len(movies) > 1 {
		movie, err = chooseMovie(movies)
		if err != nil {
			return err
		}
		if err := imdbClient.Details(context.Background(), &movie); err != nil {
			fmt.Fprintf(os.Stderr, "IMDb metadata unavailable: %v\n", err)
		}
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
			fmt.Fprintf(os.Stderr, "Browser link: %s\n", r.url)
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
	// Search enough sources to find multiple providers. If we only check 20,
	// we might find just one match. Checking 60–100 gives cross-provider
	// diversity: Dune from Reelix, Dune from Chillflix, Dune from ArrowTV.
	const maxPlaySources = 80
	if len(sources) > maxPlaySources {
		sources = sources[:maxPlaySources]
	}

	alive, err := aliveSources(sources, false)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "searching %d live sites...\n", len(alive))

	cfg := search.DefaultConfig()
	cfg.Workers = 32
	cfg.Timeout = 3 * time.Second
	cfg.Limit = 10
	cfg.OnProgress = func(done, total int) {
		if done == total || done%5 == 0 {
			fmt.Fprintf(os.Stderr, "search progress: %d/%d sites checked\n", done, total)
		}
	}

	searchCtx, cancelSearch := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelSearch()
	searchResults, _ := search.Run(searchCtx, alive, title, cfg)
	if len(searchResults) == 0 {
		return fmt.Errorf("no playable stream found")
	}

	for i := 0; i < len(searchResults); i++ {
		r := searchResults[i]
		fmt.Fprintf(os.Stderr, "trying: %s — %s\n", r.Site, r.Title)

		streamURL, err := player.Extract(context.Background(), r.URL, 30*time.Second)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  yt-dlp cannot extract (JavaScript site)\n")
			fmt.Fprintf(os.Stderr, "  Browser link: %s\n", r.URL)
			continue
		}

		fmt.Fprintf(os.Stderr, "playing: %s\n", r.Title)
		if err := player.Play(context.Background(), streamURL, r.Title); err != nil {
			fmt.Fprintf(os.Stderr, "  playback failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "Browser link: %s\n", r.URL)
			continue
		}
		fmt.Fprintf(os.Stderr, "playing: %s (%d)\n", movie.Title, movie.Year)
		return nil
	}

	fmt.Fprintln(os.Stderr, "\nNo direct stream could be extracted. All results are JavaScript-rendered sites.")
	fmt.Fprintln(os.Stderr, "Open one of these browser links instead:")
	for i, r := range searchResults {
		fmt.Fprintf(os.Stderr, "  %d. %s — %s\n     %s\n", i+1, r.Site, r.Title, r.URL)
	}
	return fmt.Errorf("no playable stream found")
}

// yearRe extracts a 4-digit year from a search result title.
var yearRe = regexp.MustCompile(`\b(19|20)\d{2}\b`)

// sequelRe matches sequel indicators in titles.
var sequelRe = regexp.MustCompile(`(?i)\b(part\s*[234]|part\s*(two|three|four|five)|[234]\s*:|\bII\b|\bIII\b|\bIV\b|sequel)\b`)

// excludedHosts are domains that are not media streaming sites.
var excludedHosts = []string{"github.com", "wikipedia.org", "reddit.com", "imdb.com"}

// filterByMovie filters search results to match a specific movie (title + year).
// It excludes sequels when the target is not itself a sequel, and filters by
// the selected year so "Dune (2021)" returns Dune Part 1, not Part 2/3.
func filterByMovie(results []search.Result, movie imdb.Movie) []search.Result {
	out := make([]search.Result, 0, len(results))
	yearStr := strconv.Itoa(movie.Year)
	movieTitleLower := strings.ToLower(movie.Title)
	targetIsSequel := sequelRe.MatchString(movie.Title)

	for _, r := range results {
		t := strings.ToLower(r.Title)

		// Skip non-media domains (GitHub, Wikipedia, etc.).
		if u, err := url.Parse(r.URL); err == nil {
			host := strings.ToLower(u.Hostname())
			excluded := false
			for _, bad := range excludedHosts {
				if host == bad || strings.HasSuffix(host, "."+bad) {
					excluded = true
					break
				}
			}
			if excluded {
				continue
			}
		}

		// Skip sequels when the target is the original film.
		if !targetIsSequel && sequelRe.MatchString(t) {
			continue
		}

		// If the result title contains a year, it must match the selected movie.
		if found := yearRe.FindString(r.Title); found != "" && found != yearStr {
			continue
		}

		// Ensure the title contains the movie name (base words).
		if !strings.Contains(t, movieTitleLower) {
			// Allow partial match on first significant word (e.g. "dune" matches
			// "Dune 7.8 • 2021 • Movie")
			words := strings.Fields(movieTitleLower)
			if len(words) > 0 && !strings.Contains(t, words[0]) {
				continue
			}
		}

		out = append(out, r)
	}
	return out
}

func selectQuery(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("usage: kamui <title>")
	}
	recordHistory(title)
	metaClient := imdb.NewClient()
	movies, metaErr := metaClient.Search(context.Background(), title)
	var selectedMovie *imdb.Movie
	if metaErr != nil {
		fmt.Fprintf(os.Stderr, "Movie metadata unavailable: %v\n", metaErr)
	} else if len(movies) > 0 {
		movie := movies[0]
		if len(movies) > 1 {
			movie, metaErr = chooseMovie(movies)
			if metaErr != nil {
				return metaErr
			}
		}
		if err := metaClient.Details(context.Background(), &movie); err != nil {
			fmt.Fprintf(os.Stderr, "IMDb rating lookup failed: %v\n", err)
		}
		printMovieCard(movie)
		selectedMovie = &movie
	}
	fmt.Fprintf(os.Stderr, "Searching browser links for %q...\n", title)
	fmt.Fprintln(os.Stderr, "Loading source catalogue...")
	all, err := loadSources()
	if err != nil {
		return err
	}
	// Search enough sources to find 5 results matching the exact movie.
	const maxBrowserSearchSources = 150
	if len(all) > maxBrowserSearchSources {
		all = all[:maxBrowserSearchSources]
	}
	fmt.Fprintf(os.Stderr, "Searching %d sites for %q...\n", len(all), title)
	cfg := search.DefaultConfig()
	cfg.Workers = 24
	cfg.Timeout = 3 * time.Second
	cfg.Limit = 50 // get many results so filtering can find the right ones
	cfg.NoDedupe = true // dedupe AFTER filtering by movie/year
	cfg.OnProgress = func(done, total int) {
		if done == total || done%5 == 0 {
			fmt.Fprintf(os.Stderr, "search progress: %d/%d sites checked\n", done, total)
		}
	}
	alive, err := aliveSources(all, false)
	if err != nil {
		return err
	}
	searchCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	results, _ := search.Run(searchCtx, alive, title, cfg)
	if len(results) == 0 {
		return fmt.Errorf("no results for %q", title)
	}

	// Filter results to match the selected movie (year + title), then dedupe
	// by host. This order matters: a site may have both Dune 1 and Dune 2 pages,
	// and we want the Dune 1 page, not just the highest-scored one.
	if selectedMovie != nil {
		filtered := filterByMovie(results, *selectedMovie)
		if len(filtered) > 0 {
			fmt.Fprintf(os.Stderr, "Filtered to %d results matching %s (%d)...\n", len(filtered), selectedMovie.Title, selectedMovie.Year)
			results = search.DedupeByHost(filtered)
		} else {
			fmt.Fprintf(os.Stderr, "No exact matches for %s (%d); showing all results.\n", selectedMovie.Title, selectedMovie.Year)
			results = search.DedupeByHost(results)
		}
	} else {
		results = search.DedupeByHost(results)
	}

	// Show top 5 results from different providers.
	const topResults = 5
	if len(results) > topResults {
		results = results[:topResults]
	}
	for i, r := range results {
		fmt.Fprintf(os.Stdout, "%d. %s — %s\n   %s\n", i+1, r.Site, r.Title, r.URL)
	}
	return nil
}

func recordHistory(title string) {
	if s, err := library.New(); err == nil {
		_ = s.Add("history", title)
	}
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
