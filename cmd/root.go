// Package cmd holds kamui's subcommands.
package cmd

import (
	"fmt"
	"os"
)

// Execute runs the command named by args[0] and returns any error.
//
// Errors travel up to main(), which prints them and sets the exit code. No
// command calls os.Exit itself.
func Execute(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}

	switch args[0] {
	case "sources":
		return Sources(args[1:])
	case "probe":
		return Probe(args[1:])
	case "search":
		return Search(args[1:])
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
  kamui sources [flags]      list the sites FMHY recommends
  kamui probe   [flags]      check which of those sites are actually alive
  kamui search  [flags] <title>
                             search every live site in parallel
                             (flags go before the title)
  kamui help                 show this message

Run "kamui <command> -h" for the flags of a command.
`)
}
