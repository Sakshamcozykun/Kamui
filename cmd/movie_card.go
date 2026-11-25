package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	termimg "github.com/blacktop/go-termimg"
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

func printPoster(posterURL string) error {
	resp, err := http.Get(posterURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("poster HTTP %d", resp.StatusCode)
	}
	temp, err := os.CreateTemp("", "kamui-poster-*.img")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, err := temp.ReadFrom(resp.Body); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "Poster:")
	img, err := termimg.Open(filepath.Clean(name))
	if err != nil {
		return err
	}
	return img.Width(32).Height(40).Print()
}
