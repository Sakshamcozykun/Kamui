// Package imdb provides movie metadata via IMDB's public suggestion endpoint.
// No API key required.
package imdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// endpoint is a package variable so tests can point it at a local server.
var endpoint = "https://v2.sg.media-imdb.com/suggestion"
var omdbEndpoint = "https://www.omdbapi.com/"

// Movie is a search result from IMDB.
type Movie struct {
	ID      string // IMDB ID, e.g. "tt1979376"
	Title   string
	Year    int
	Poster  string
	Rating  float64
	Runtime string
	Genre   string
	Plot    string
}

// Details fills movie metadata from OMDb, including IMDb's own rating field.
func (c *Client) Details(ctx context.Context, movie *Movie) error {
	if movie == nil || movie.ID == "" {
		return fmt.Errorf("missing movie id")
	}
	key, err := apiKey()
	if err != nil {
		return err
	}
	u, err := url.Parse(omdbEndpoint)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("i", movie.ID)
	q.Set("apikey", key)
	q.Set("r", "json")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Kamui/1.0")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("imdb details: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("imdb details: HTTP %d", resp.StatusCode)
	}
	var data struct {
		Title      string `json:"Title"`
		Year       string `json:"Year"`
		Poster     string `json:"Poster"`
		IMDbRating string `json:"imdbRating"`
		Runtime    string `json:"Runtime"`
		Genre      string `json:"Genre"`
		Plot       string `json:"Plot"`
		Response   string `json:"Response"`
		Error      string `json:"Error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("OMDb response: %w", err)
	}
	if data.Response == "False" {
		return fmt.Errorf("OMDb: %s", data.Error)
	}
	if data.Title != "" {
		movie.Title = data.Title
	}
	movie.Runtime, movie.Genre, movie.Plot = data.Runtime, data.Genre, data.Plot
	if year, err := strconv.Atoi(strings.Split(data.Year, "–")[0]); err == nil {
		movie.Year = year
	}
	if data.Poster != "" && data.Poster != "N/A" {
		movie.Poster = data.Poster
	}
	if data.IMDbRating != "" && data.IMDbRating != "N/A" {
		movie.Rating, err = strconv.ParseFloat(data.IMDbRating, 64)
		if err != nil {
			return fmt.Errorf("invalid IMDb rating from OMDb: %w", err)
		}
	}
	if movie.Rating == 0 {
		return fmt.Errorf("OMDb did not return an IMDb rating for %s", movie.ID)
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
	queryLower := strings.ToLower(strings.TrimSpace(query))
	slices.SortStableFunc(movies, func(a, b Movie) int {
		aExact, bExact := strings.EqualFold(a.Title, query), strings.EqualFold(b.Title, query)
		if aExact != bExact {
			if aExact {
				return -1
			}
			return 1
		}
		aStarts, bStarts := strings.HasPrefix(strings.ToLower(a.Title), queryLower), strings.HasPrefix(strings.ToLower(b.Title), queryLower)
		if aStarts != bStarts {
			if aStarts {
				return -1
			}
			return 1
		}
		if a.Year != b.Year {
			if a.Year > b.Year {
				return -1
			}
			return 1
		}
		return 0
	})
	return movies, nil
}
