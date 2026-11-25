package cmd

import (
	"flag"
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
	Short: "Pull streams from FMHY-listed sites into your terminal",
	Long: `kamui pulls streams from FMHY-listed free media sites into your terminal.

With no subcommand, a title is searched and played:
  kamui "toy story 4"

Subcommands:
  sources   list the sites FMHY recommends
  probe     check which of those sites are actually alive
  search    search every live site in parallel (no playback)
  play      search and play a title
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
func Execute() error {
	args := os.Args[1:]
	if len(args) == 0 {
		return rootCmd.Execute()
	}
	if isSubcommand(args[0]) {
		return rootCmd.Execute()
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
	return playQuery(strings.Join(positional, " "))
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
