// Package player wraps external media tools: yt-dlp for stream extraction and
// mpv for playback.
package player

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LookPathTool locates a media executable, including common Windows install
// directories for packages that do not register their binaries on PATH.
func LookPathTool(name string) (string, error) {
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	if runtime.GOOS == "windows" && name == "mpv" {
		for _, dir := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
			if dir == "" {
				continue
			}
			for _, subdir := range []string{"MPV Player", "mpv"} {
				candidate := filepath.Join(dir, subdir, "mpv.exe")
				if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
					return candidate, nil
				}
			}
		}
	}
	return "", fmt.Errorf("%s not found on PATH or in standard install locations", name)
}

// Extract asks yt-dlp for a direct stream URL behind a page. It returns the
// first URL yt-dlp prints, or an error if extraction fails.
func Extract(ctx context.Context, pageURL string, timeout time.Duration) (string, error) {
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		return "", fmt.Errorf("yt-dlp not found; install it with: sudo apt install yt-dlp")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// --no-update silences the "your version is older than 90 days" nag that
	// apt-installed yt-dlp prints on every run. --no-warnings drops the rest.
	cmd := exec.CommandContext(ctx, "yt-dlp", "--no-update", "--no-warnings", "--no-playlist", "-g", pageURL)
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
	mpvPath, err := LookPathTool("mpv")
	if err != nil {
		return fmt.Errorf("mpv not found; install it with: sudo apt install mpv")
	}

	cmd := exec.CommandContext(ctx, mpvPath, "--force-media-title="+title, streamURL)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PlayPage asks mpv's yt-dlp hook to resolve a supported site page. Leaving
// format selection to the hook lets mpv consume separate audio/video formats
// and avoids requiring a pre-merged format to exist.
func PlayPage(ctx context.Context, pageURL, title, ytDLPPath string) error {
	mpvPath, err := LookPathTool("mpv")
	if err != nil {
		return fmt.Errorf("mpv not found; install it and ensure it is on PATH (run 'kamui doctor')")
	}
	if ytDLPPath == "" {
		return fmt.Errorf("yt-dlp not found; install it and ensure it is on PATH (run 'kamui doctor')")
	}

	cmd := exec.CommandContext(ctx, mpvPath,
		"--ytdl=yes",
		"--script-opts=ytdl_hook-ytdl_path="+ytDLPPath,
		"--force-media-title="+title,
		pageURL,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
