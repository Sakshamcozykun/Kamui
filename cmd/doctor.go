package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os/exec"
	"runtime"
)

var doctorCmd = &cobra.Command{Use: "doctor", Short: "Check playback dependencies", Run: func(cmd *cobra.Command, args []string) {
	for _, name := range []string{"mpv", "yt-dlp", "ffmpeg"} {
		if path, err := exec.LookPath(name); err == nil {
			fmt.Printf("[ok] %s: %s\n", name, path)
			continue
		}
		install := "install it and add it to PATH"
		switch runtime.GOOS {
		case "windows":
			install = "install it, then add it to PATH"
		case "linux":
			install = "sudo apt install mpv yt-dlp ffmpeg (or use your distro package manager)"
		case "darwin":
			install = "brew install mpv yt-dlp ffmpeg"
		}
		fmt.Printf("[missing] %s: %s\n", name, install)
	}
}}

func init() { rootCmd.AddCommand(doctorCmd) }
