package cmd

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"strings"

	"github.com/sakshamcozykun/kamui/internal/imdb"
)

func printMovieCard(movie imdb.Movie) {
	fmt.Fprintf(os.Stderr, "\n%s (%d)\n", movie.Title, movie.Year)
	if movie.Rating > 0 {
		fmt.Fprintf(os.Stderr, "IMDb rating: %.1f/10\n", movie.Rating)
	} else {
		fmt.Fprintln(os.Stderr, "IMDb rating: unavailable")
	}
	if movie.Poster == "" {
		return
	}
	if os.Getenv("NO_COLOR") != "" {
		fmt.Fprintf(os.Stderr, "Poster: %s\n", movie.Poster)
		return
	}
	if err := printPoster(movie.Poster); err != nil {
		fmt.Fprintf(os.Stderr, "Poster: %s\n", movie.Poster)
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
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return err
	}
	img = fitPoster(img, 24, 16)
	fmt.Fprintln(os.Stderr, "Poster:")
	for y := 0; y < img.Bounds().Dy(); y += 2 {
		var line strings.Builder
		for x := 0; x < img.Bounds().Dx(); x++ {
			top := rgb(img.At(x, y))
			bottom := top
			if y+1 < img.Bounds().Dy() {
				bottom = rgb(img.At(x, y+1))
			}
			fmt.Fprintf(&line, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm\u2580", top[0], top[1], top[2], bottom[0], bottom[1], bottom[2])
		}
		fmt.Fprintf(os.Stderr, "%s\x1b[0m\n", line.String())
	}
	return nil
}

func fitPoster(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	sb := src.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sx := sb.Min.X + x*sb.Dx()/width
			sy := sb.Min.Y + y*sb.Dy()/height
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

func rgb(c color.Color) [3]uint8 {
	r, g, b, _ := c.RGBA()
	return [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}
}
