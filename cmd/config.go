package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

var configCmd = &cobra.Command{Use: "config", Short: "Show Kamui configuration status", RunE: func(cmd *cobra.Command, args []string) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "kamui", "config.json")
	_, err = os.Stat(path)
	configured := err == nil || os.Getenv("OMDB_API_KEY") != ""
	fmt.Printf("Config file: %s\nOMDb key: ", path)
	if configured {
		fmt.Println("configured")
	} else {
		fmt.Println("not configured; run 'kamui poster'")
	}
	return nil
}}

func init() { rootCmd.AddCommand(configCmd) }
