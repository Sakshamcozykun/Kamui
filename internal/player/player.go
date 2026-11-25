// Package player wraps external media tools: yt-dlp for stream extraction and
// mpv for playback.
package player

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Extract asks yt-dlp for a direct stream URL behind a page. It returns the
// first URL yt-dlp prints, or an error if extraction fails.
func Extract(ctx context.Context, pageURL string, timeout time.Duration) (string, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return "", fmt.Errorf("yt-dlp not found; install it with: sudo apt install yt-dlp")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "yt-dlp", "--no-playlist", "-g", pageURL)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp could not extract a stream: %w", err)
	}

	url := strings.TrimSpace(string(out))
	if i := strings.IndexByte(url, '\n'); i >= 0 {
		url = url[:i]
	}
	if url == "" {
		return "", fmt.Errorf("yt-dlp returned no stream URL")
	}
	return url, nil
}

// Play runs mpv on a direct stream URL. It blocks until playback ends.
func Play(ctx context.Context, streamURL, title string) error {
	if _, err := exec.LookPath("mpv"); err != nil {
		return fmt.Errorf("mpv not found; install it with: sudo apt install mpv")
	}

	cmd := exec.CommandContext(ctx, "mpv", "--force-media-title="+title, streamURL)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
