package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sakshamcozykun/kamui/internal/player"
	"github.com/spf13/cobra"
)

var musicCmd = &cobra.Command{
	Use:   "music <search terms or URL>",
	Short: "Find and play music with yt-dlp and mpv",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.TrimSpace(strings.Join(args, " "))
		mpvPath, err := player.LookPathTool("mpv")
		if err != nil {
			return fmt.Errorf("mpv is required for music playback; install it and ensure it is on PATH (run 'kamui doctor')")
		}
		ytDLPPath, err := player.LookPathTool("yt-dlp")
		if err != nil {
			return fmt.Errorf("yt-dlp is required for music search; install it and ensure it is on PATH (run 'kamui doctor')")
		}

		playURL := query
		title := query
		radio := false
		var radioURLs []string
		if !isWebURL(query) {
			tracks, err := searchTracks(query, ytDLPPath)
			if err != nil {
				return err
			}
			if len(tracks) == 0 {
				return fmt.Errorf("no music results for %q", query)
			}
			if !tracksHaveArtist(tracks) {
				fmt.Fprint(os.Stderr, "Artist name (leave blank to keep searching): ")
				artist, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				artist = strings.TrimSpace(artist)
				if artist != "" {
					tracks, err = searchTracks(artist+" "+query, ytDLPPath)
					if err != nil {
						return err
					}
				}
			}
			for i, t := range tracks {
				artist := t.Artist
				if artist == "" {
					artist = "Artist not identified"
				}
				fmt.Fprintf(os.Stderr, "%d. %s — %s\n", i+1, t.Title, artist)
			}
			fmt.Fprint(os.Stderr, "Choose track (1-5, or 0 to cancel): ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			choice, err := strconv.Atoi(strings.TrimSpace(line))
			if err != nil || choice < 1 || choice > len(tracks) {
				return fmt.Errorf("music selection cancelled or invalid")
			}
			track := tracks[choice-1]
			if track.ID == "" {
				playURL = track.URL
			} else {
				playURL = "https://www.youtube.com/watch?v=" + url.QueryEscape(track.ID)
			}
			title = track.Title
			fmt.Fprint(os.Stderr, "Play track [p] or start similar-song radio [r]? ")
			mode, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			radio = strings.EqualFold(strings.TrimSpace(mode), "r")
			if radio {
				if track.ID == "" {
					return fmt.Errorf("radio needs a YouTube track ID")
				}
				radioURLs, err = getRadioURLs(cmd.Context(), ytDLPPath, track)
				if err != nil {
					return err
				}
			}
		}
		label := "audio playback"
		mpvArgs := []string{"--no-video", "--ytdl=yes", "--script-opts=ytdl_hook-ytdl_path=" + ytDLPPath}
		if radio {
			label = fmt.Sprintf("radio (%d tracks queued)", len(radioURLs))
			title = "Kamui Radio"
			mpvArgs = append(mpvArgs, "--force-media-title="+title)
			mpvArgs = append(mpvArgs, radioURLs...)
		} else {
			mpvArgs = append(mpvArgs, "--force-media-title="+title, playURL)
		}
		fmt.Fprintf(os.Stderr, "Starting %s...\n", label)
		mpv := exec.CommandContext(cmd.Context(), mpvPath, mpvArgs...)
		mpv.Stdin, mpv.Stdout, mpv.Stderr = os.Stdin, os.Stdout, os.Stderr
		return mpv.Run()
	},
}

type musicTrack struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Artist   string `json:"artist"`
	Uploader string `json:"uploader"`
	Channel  string `json:"channel"`
}

func searchTracks(query, ytDLPPath string) ([]musicTrack, error) {
	fmt.Fprintf(os.Stderr, "Searching YouTube for %q...\n", query)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, ytDLPPath, "--no-warnings", "--flat-playlist", "--dump-single-json", "ytsearch5:"+query)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			return nil, fmt.Errorf("music search failed: %s", detail)
		}
		return nil, fmt.Errorf("music search failed: %w", err)
	}
	var found struct {
		Entries []musicTrack `json:"entries"`
	}
	if err := json.Unmarshal(out, &found); err != nil {
		return nil, fmt.Errorf("parse yt-dlp music results: %w", err)
	}
	if len(found.Entries) > 5 {
		found.Entries = found.Entries[:5]
	}
	for i := range found.Entries {
		if found.Entries[i].Artist == "" {
			found.Entries[i].Artist = found.Entries[i].Uploader
		}
		if found.Entries[i].Artist == "" {
			found.Entries[i].Artist = found.Entries[i].Channel
		}
	}
	return found.Entries, nil
}

func tracksHaveArtist(tracks []musicTrack) bool {
	if len(tracks) == 0 {
		return false
	}
	for _, t := range tracks {
		if strings.TrimSpace(t.Artist) == "" {
			return false
		}
	}
	return true
}

// getRadioURLs expands YouTube's song radio mix into a finite queue so mpv
// plays the recommended tracks one after another instead of treating the mix
// page as a single video.
func getRadioURLs(parent context.Context, ytDLPPath string, seed musicTrack) ([]string, error) {
	fmt.Fprintf(os.Stderr, "Loading similar-song radio for %q...\n", seed.Title)
	values := url.Values{}
	values.Set("v", seed.ID)
	values.Set("list", "RD"+seed.ID)
	mixURL := "https://www.youtube.com/watch?" + values.Encode()
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, ytDLPPath,
		"--no-warnings", "--no-update", "--flat-playlist", "--playlist-end", "20", "--dump-single-json", "--", mixURL,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	var parsed struct {
		Entries []musicTrack `json:"entries"`
	}
	if err == nil {
		err = json.Unmarshal(out, &parsed)
	}
	mixErr := err
	queued := make([]string, 0, 15)
	seen := make(map[string]struct{})
	if err == nil {
		for _, entry := range parsed.Entries {
			id := strings.TrimSpace(entry.ID)
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			queued = append(queued, "https://www.youtube.com/watch?v="+url.QueryEscape(id))
			if len(queued) == cap(queued) {
				break
			}
		}
	}
	if len(queued) >= 2 {
		return queued, nil
	}

	// A generated mix is occasionally unavailable. Use a similar-song search
	// as a fallback and omit the selected seed so radio continues with others.
	query := "similar songs to " + seed.Title
	if seed.Artist != "" {
		query += " by " + seed.Artist
	}
	fallback, searchErr := searchTracks(query, ytDLPPath)
	for _, track := range fallback {
		if track.ID == "" || track.ID == seed.ID {
			continue
		}
		if _, exists := seen[track.ID]; exists {
			continue
		}
		seen[track.ID] = struct{}{}
		queued = append(queued, "https://www.youtube.com/watch?v="+url.QueryEscape(track.ID))
		if len(queued) == 10 {
			break
		}
	}
	if len(queued) > 0 {
		return queued, nil
	}
	if searchErr != nil {
		if mixErr != nil {
			return nil, fmt.Errorf("could not load YouTube radio mix (%v), and similar-song search failed: %w", mixErr, searchErr)
		}
		return nil, fmt.Errorf("YouTube radio returned too few tracks, and similar-song search failed: %w", searchErr)
	}
	if detail := strings.TrimSpace(stderr.String()); detail != "" {
		return nil, fmt.Errorf("could not load a radio queue: %s", detail)
	}
	return nil, fmt.Errorf("YouTube returned no similar tracks for %q", seed.Title)
}

func isWebURL(value string) bool {
	u, err := url.ParseRequestURI(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func init() { rootCmd.AddCommand(musicCmd) }
