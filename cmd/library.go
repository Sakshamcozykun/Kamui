package cmd

import (
	"fmt"
	"github.com/sakshamcozykun/kamui/internal/library"
	"github.com/spf13/cobra"
	"strings"
)

var libraryCmd = &cobra.Command{Use: "library", Short: "Manage your watchlist and search history"}

func init() {
	rootCmd.AddCommand(libraryCmd)
	for _, name := range []string{"watchlist", "history"} {
		listName := name
		listCmd := &cobra.Command{Use: listName, Short: "Show " + listName, RunE: func(cmd *cobra.Command, args []string) error {
			s, err := library.New()
			if err != nil {
				return err
			}
			entries, err := s.List(listName)
			if err != nil {
				return err
			}
			for i, e := range entries {
				fmt.Printf("%d. %s\n", i+1, e.Title)
			}
			return nil
		}}
		libraryCmd.AddCommand(listCmd)
	}
	add := &cobra.Command{Use: "add <title>", Args: cobra.MinimumNArgs(1), Short: "Add a title to your watchlist", RunE: func(cmd *cobra.Command, args []string) error {
		s, e := library.New()
		if e != nil {
			return e
		}
		return s.Add("watchlist", strings.Join(args, " "))
	}}
	remove := &cobra.Command{Use: "remove <title>", Args: cobra.MinimumNArgs(1), Short: "Remove a title from your watchlist", RunE: func(cmd *cobra.Command, args []string) error {
		s, e := library.New()
		if e != nil {
			return e
		}
		return s.Remove("watchlist", strings.Join(args, " "))
	}}
	libraryCmd.AddCommand(add, remove)
}
