package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var musicCmd = &cobra.Command{
	Use:   "music <search terms or URL>",
	Short: "Find and play music with yt-dlp and mpv",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.TrimSpace(strings.Join(args, " "))
		if _, err := exec.LookPath("mpv"); err != nil {
			return fmt.Errorf("mpv is required for music playback; install it and ensure it is on PATH (run 'kamui doctor')")
		}
		if _, err := exec.LookPath("yt-dlp"); err != nil {
			return fmt.Errorf("yt-dlp is required for music search; install it and ensure it is on PATH (run 'kamui doctor')")
		}

		playURL := query
		title := query
		radio := false
		if !isWebURL(query) {
			tracks, err := searchTracks(query)
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
					tracks, err = searchTracks(artist + " " + query)
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
				playURL += "&list=RD" + url.QueryEscape(track.ID)
			}
		}
		fmt.Fprintf(os.Stderr, "Starting %s...\n", map[bool]string{true: "radio", false: "audio playback"}[radio])
		player := exec.Command("mpv", "--no-video", "--ytdl=yes", "--force-media-title="+title, playURL)
		player.Stdin, player.Stdout, player.Stderr = os.Stdin, os.Stdout, os.Stderr
		return player.Run()
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

func searchTracks(query string) ([]musicTrack, error) {
	fmt.Fprintf(os.Stderr, "Searching YouTube for %q...\n", query)
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "yt-dlp", "--no-warnings", "--flat-playlist", "--dump-single-json", "ytsearch5:"+query).Output()
	if err != nil {
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
	for _, t := range tracks {
		if t.Artist != "" {
			return true
		}
	}
	return false
}

func isWebURL(value string) bool {
	u, err := url.ParseRequestURI(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func init() { rootCmd.AddCommand(musicCmd) }
