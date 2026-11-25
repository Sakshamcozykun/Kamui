// Command kamui pulls streams from FMHY-listed sites into your terminal.
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
