package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/sakshamcozykun/kamui/internal/imdb"
	"github.com/spf13/cobra"
)

var posterCmd = &cobra.Command{
	Use:   "poster",
	Short: "Set up OMDb metadata for ratings and posters",
	Long:  "Prompt for an OMDb API key and save it to your per-user Kamui config.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprint(os.Stderr, "Get a key at https://www.omdbapi.com/apikey.aspx\nOMDb API key: ")
		key, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return fmt.Errorf("read API key: %w", err)
		}
		key = strings.TrimSpace(key)
		if err := imdb.SetAPIKey(key); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "OMDb key saved to your user config. Run a movie search to see ratings and posters.")
		return nil
	},
}

func init() { rootCmd.AddCommand(posterCmd) }
