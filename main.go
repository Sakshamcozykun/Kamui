// Command kamui pulls streams from FMHY-listed sites into your terminal.
//
// Day 1: read the wiki. Run "kamui sources".
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/sakshamcozykun/kamui/internal/fmhy"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "kamui:", err)
		os.Exit(1)
	}
}

// run is main's body, split out so it can return an error instead of calling
// os.Exit from twenty places. main() stays a four-line wrapper.
func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}

	switch args[0] {
	case "sources":
		return cmdSources(args[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q (try: kamui help)", args[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `kamui - pull streams from FMHY-listed sites

Usage:
  kamui sources [flags]   list the sites FMHY recommends
  kamui help              show this message
`)
}

func cmdSources(args []string) error {
	fs := flag.NewFlagSet("sources", flag.ContinueOnError)
	section := fs.String("section", "", "only show sections containing this text (case-insensitive)")
	starred := fs.Bool("starred", false, "only show FMHY-starred picks")
	max := fs.Int("max", 0, "stop after N sources (0 = all)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()

	client := fmhy.NewClient(cacheDir())
	body, err := client.Fetch(ctx, fmhy.DocVideo)
	if err != nil {
		return err
	}

	sources := fmhy.Dedupe(fmhy.Parse(string(body)))
	sources = filter(sources, *section, *starred)

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

// filter keeps the sources the user asked for. Kept as its own function so it
// can be tested and reused by the search command later.
func filter(sources []fmhy.Source, section string, starredOnly bool) []fmhy.Source {
	section = strings.ToLower(section)

	out := make([]fmhy.Source, 0, len(sources))
	for _, s := range sources {
		if starredOnly && !s.Starred {
			continue
		}
		if section != "" && !strings.Contains(strings.ToLower(strings.Join(s.Path, " ")), section) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// printSources writes a grouped table: one block per section, in the order the
// wiki lists them.
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

// cacheDir returns ~/.cache/kamui on Linux, or "" if the OS will not say,
// which disables caching.
func cacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "kamui")
}
