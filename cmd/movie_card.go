package cmd

import (
	"fmt"
	"os"

	"github.com/cozykun/goimg"
	"github.com/sakshamcozykun/kamui/internal/imdb"
)

func printMovieCard(movie imdb.Movie) {
	fmt.Fprintf(os.Stderr, "\n%s (%d)\n", movie.Title, movie.Year)
	if movie.Rating > 0 {
		fmt.Fprintf(os.Stderr, "IMDb rating: %.1f/10\n", movie.Rating)
	} else {
		fmt.Fprintln(os.Stderr, "IMDb rating: unavailable")
	}
	if movie.Runtime != "" {
		fmt.Fprintf(os.Stderr, "Runtime: %s\n", movie.Runtime)
	}
	if movie.Genre != "" {
		fmt.Fprintf(os.Stderr, "Genre: %s\n", movie.Genre)
	}
	if movie.Plot != "" {
		fmt.Fprintf(os.Stderr, "Plot: %s\n", movie.Plot)
	}
	if movie.Poster == "" {
		return
	}
	if err := printPoster(movie.Poster); err != nil {
		fmt.Fprintf(os.Stderr, "Could not render poster: %v\n", err)
	}
}

// printPoster renders a poster image using goimg, which auto-detects the
// best terminal protocol (sixel, kitty, iTerm2, or halfblock fallback).
func printPoster(posterURL string) error {
	fmt.Fprintln(os.Stderr, "Poster:")
	player := goimg.New(goimg.Config{})
	return player.ViewImage(posterURL)
}
