package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	playMax     int
	playWorkers int
	playTimeout time.Duration
	playPick    bool
	playDebug   bool
)

var rootCmd = &cobra.Command{
	Use:   "kamui [title]",
	Short: "Search media sources and play results from your terminal",
	Long: `kamui searches listed media sources and provides playback from your terminal.

With no subcommand, a title is searched and played:
  kamui "toy story 4"

Subcommands:
  sources   list available sources
  probe     check which sources are reachable
  search    search every live site in parallel (no playback)
  anime     search anime streaming sites only
  play      search and play a title
  stream    resolve a supported media URL and play it in mpv
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return playQuery(strings.Join(args, " "))
	},
}

// Execute is the entry point called by main.
//
// cobra only knows about registered subcommands, so a bare title like
// "kamui toy story 4" would otherwise be rejected as an unknown command.
// We route anything that is not a known subcommand to the play path.
//
// Exception: if there is exactly one positional argument and it does not
// contain a space, we treat it as a potential subcommand that doesn't exist
// rather than a movie title, and show an error.
func Execute() error {
	args := os.Args[1:]
	if len(args) == 0 {
		printWelcome()
		return nil
	}
	if isSubcommand(args[0]) {
		return rootCmd.Execute()
	}
	if command, ok := closeCommand(args[0]); ok {
		fmt.Fprintf(os.Stderr, "Rin caught that typo! Did you mean `kamui %s`?\n", command)
		if len(args) > 1 {
			fmt.Fprintf(os.Stderr, "Try: kamui %s %s\n", command, strings.Join(args[1:], " "))
		} else {
			fmt.Fprintf(os.Stderr, "Run `kamui %s --help` to see how it works.\n", command)
		}
		return nil
	}

	// Separate flags from the title so "kamui --max 10 toy story 4" works
	// even though the title contains spaces.
	flags, positional := splitFlags(args)

	if len(flags) > 0 {
		fs := flag.NewFlagSet("kamui", flag.ContinueOnError)
		fs.IntVar(&playMax, "max", playMax, "")
		fs.IntVar(&playWorkers, "workers", playWorkers, "")
		fs.DurationVar(&playTimeout, "timeout", playTimeout, "")
		fs.BoolVar(&playPick, "pick", playPick, "")
		fs.BoolVar(&playDebug, "debug", playDebug, "")
		if err := fs.Parse(flags); err != nil {
			return err
		}
	}

	if len(positional) == 0 {
		return rootCmd.Execute()
	}
	return selectQuery(strings.Join(positional, " "))
}

func closeCommand(input string) (string, bool) {
	commands := []string{"play", "select", "search", "anime", "sources", "probe", "stream", "music", "doctor", "config", "library", "poster"}
	input = strings.ToLower(strings.TrimSpace(input))
	best, bestDistance := "", 3
	for _, candidate := range commands {
		d := editDistance(input, candidate)
		if d < bestDistance {
			best, bestDistance = candidate, d
		}
	}
	return best, bestDistance <= 2
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i, ra := range a {
		current := make([]int, len(b)+1)
		current[0] = i + 1
		for j, rb := range b {
			cost := 0
			if ra != rb {
				cost = 1
			}
			current[j+1] = min(current[j]+1, prev[j+1]+1, prev[j]+cost)
		}
		prev = current
	}
	return prev[len(b)]
}

func printWelcome() {
	fmt.Fprint(os.Stdout, `
  ██╗  ██╗ █████╗ ███╗   ███╗██╗   ██╗██╗
  ██║ ██╔╝██╔══██╗████╗ ████║██║   ██║██║
  █████╔╝ ███████║██╔████╔██║██║   ██║██║
  ██╔═██╗ ██╔══██║██║╚██╔╝██║██║   ██║██║
  ██║  ██╗██║  ██║██║ ╚═╝ ██║╚██████╔╝██║
  ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝ ╚═════╝ ╚═╝

  Search media sources and play results from your terminal.

  COMMANDS
  ──────────────────────────────────────────────────────────
  kamui "movie title"              Find a title and print browser links
  kamui play "movie title"         Search and play a title
  kamui stream <url>               Resolve a supported URL and play in mpv
  kamui music "artist - track"     Search and play music
  kamui select "movie title"       Search and choose a result
  kamui search "movie title"       Search live sites, no playback
  kamui anime "anime title"        Search anime streaming sites only
  kamui sources                    List recommended sites
  kamui probe                      Check site health
  kamui doctor                     Check playback dependencies
  kamui library add "movie title"   Add a title to your watchlist
  kamui library watchlist          Show your watchlist
  kamui library history            Show search history
  kamui config                     Show configuration status

  Run 'kamui help' for all flags and examples.
`)
}

func splitFlags(args []string) (flags, positional []string) {
	boolFlags := map[string]bool{
		"--pick": true, "--debug": true, "--refresh": true, "--starred": true,
		"-pick": true, "-debug": true, "-refresh": true, "-starred": true,
	}
	i := 0
	for i < len(args) {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			name := a
			if idx := strings.Index(a, "="); idx >= 0 {
				name = a[:idx]
			}
			if !boolFlags[name] && !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, a)
		}
		i++
	}
	return
}

func isSubcommand(name string) bool {
	for _, c := range rootCmd.Commands() {
		if c.Name() == name || c.HasAlias(name) {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	// Mirror play's flags on the root command so "kamui --max 20 toy story 4"
	// works as well as "kamui play --max 20 toy story 4".
	rootCmd.Flags().IntVar(&playMax, "max", 0, "only use the first N sources (0 = all)")
	rootCmd.Flags().IntVar(&playWorkers, "workers", 12, "how many sites to search at once")
	rootCmd.Flags().DurationVar(&playTimeout, "timeout", 8*time.Second, "per-site timeout")
	rootCmd.Flags().BoolVar(&playPick, "pick", false, "show a list and ask which one to play")
	rootCmd.Flags().BoolVar(&playDebug, "debug", false, "show what each site did")
}
