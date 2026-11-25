package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

var (
	sourcesSection string
	sourcesStarred bool
	sourcesMax     int
)

var sourcesCmd = &cobra.Command{
	Use:   "sources",
	Short: "List the sites FMHY recommends",
	RunE:  runSources,
}

func init() {
	rootCmd.AddCommand(sourcesCmd)
	sourcesCmd.Flags().StringVar(&sourcesSection, "section", "", "only show sections containing this text (case-insensitive)")
	sourcesCmd.Flags().BoolVar(&sourcesStarred, "starred", false, "only show FMHY-starred picks")
	sourcesCmd.Flags().IntVar(&sourcesMax, "max", 0, "stop after N sources (0 = all)")
}

func runSources(cmd *cobra.Command, args []string) error {
	all, err := loadSources()
	if err != nil {
		return err
	}

	sources := filter(all, sourcesSection, sourcesStarred)
	if len(sources) == 0 {
		return fmt.Errorf("no sources matched")
	}
	if sourcesMax > 0 && len(sources) > sourcesMax {
		sources = sources[:sourcesMax]
	}

	printSources(sources)
	fmt.Fprintf(os.Stderr, "\n%d sources\n", len(sources))
	return nil
}

func printSources(sources []fmhy.Source) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	last := ""
	for _, s := range sources {
		group := strings.Join(s.Path, " / ")
		if group != last {
			if last != "" {
				fmt.Fprintln(w)
			}
			fmt.Fprintf(w, "%s\n", strings.ToUpper(group))
			last = group
		}

		star := " "
		if s.Starred {
			star = "★"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\n", star, s.Name, s.URL)
	}

	_ = w.Flush()
}
