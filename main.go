// Command kamui searches media sources and provides terminal playback tools.
package main

import (
	"fmt"
	"os"

	"github.com/sakshamcozykun/kamui/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "kamui:", err)
		os.Exit(1)
	}
}
