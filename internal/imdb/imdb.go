// Package imdb provides movie metadata via IMDB's public suggestion endpoint.
// No API key required.
package imdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// endpoint is a package variable so tests can point it at a local server.
var endpoint = "https://v2.sg.media-imdb.com/suggestion"

// Movie is a search result from IMDB.
type Movie struct {
	ID     string // IMDB ID, e.g. "tt1979376"
	Title  string
	Year   int
	Poster string
	Rating float64
}

// Details fills in metadata that the suggestion endpoint does not provide.
func (c *Client) Details(ctx context.Context, movie *Movie) error {
	if movie == nil || movie.ID == "" {
		return fmt.Errorf("missing movie id")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.imdb.com/title/"+movie.ID+"/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("imdb details: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("imdb details: HTTP %d", resp.StatusCode)
	}
	body := make([]byte, 0, 256*1024)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		body = append(body, buf[:n]...)
		if readErr != nil {
			break
		}
	}
	match := regexp.MustCompile(`"ratingValue"\s*:\s*"?([0-9]+(?:\.[0-9]+)?)`).FindSubmatch(body)
	if len(match) > 1 {
		movie.Rating, _ = strconv.ParseFloat(string(match[1]), 64)
	}
	return nil
}

// Client searches IMDB without an API key.
type Client struct {
	HTTP *http.Client
}

func NewClient() *Client {
	return &Client{HTTP: &http.Client{Timeout: 10 * time.Second}}
}

// Search returns movies matching the query.
func (c *Client) Search(ctx context.Context, query string) ([]Movie, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	// IMDB's suggestion endpoint uses the first character of the query
	// as a path segment. Using "x" works as a universal fallback.
	first := "x"
	if len(query) > 0 {
		first = strings.ToLower(string(query[0]))
	}

	u := fmt.Sprintf("%s/%s/%s.json", endpoint, first, url.PathEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("imdb search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("imdb search: HTTP %d", resp.StatusCode)
	}

	var result struct {
		D []struct {
			ID string `json:"id"`
			L  string `json:"l"`
			Y  int    `json:"y"`
			Q  string `json:"q"`
			I  struct {
				ImageURL string `json:"imageUrl"`
			} `json:"i"`
		} `json:"d"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("imdb search: %w", err)
	}

	var movies []Movie
	for _, d := range result.D {
		// only movies/TV, skip people
		if !strings.HasPrefix(d.ID, "tt") {
			continue
		}
		movies = append(movies, Movie{
			ID:     d.ID,
			Title:  d.L,
			Year:   d.Y,
			Poster: d.I.ImageURL,
		})
	}
	return movies, nil
}
