package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

// Sources implements "kamui sources".
func Sources(args []string) error {
	fs := flag.NewFlagSet("sources", flag.ContinueOnError)
	section := fs.String("section", "", "only show sections containing this text (case-insensitive)")
	starred := fs.Bool("starred", false, "only show FMHY-starred picks")
	max := fs.Int("max", 0, "stop after N sources (0 = all)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	all, err := loadSources()
	if err != nil {
		return err
	}

	sources := filter(all, *section, *starred)
	if len(sources) == 0 {
		return fmt.Errorf("no sources matched")
	}
	if *max > 0 && len(sources) > *max {
		sources = sources[:*max]
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
