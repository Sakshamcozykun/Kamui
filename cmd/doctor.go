package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sakshamcozykun/kamui/internal/player"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check and optionally install playback dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		missing := make([]string, 0, 3)
		for _, name := range []string{"mpv", "yt-dlp", "ffmpeg"} {
			if path, err := player.LookPathTool(name); err == nil {
				fmt.Printf("[ok]      %s: %s\n", name, path)
			} else {
				fmt.Printf("[missing] %s\n", name)
				missing = append(missing, name)
			}
		}
		if len(missing) == 0 {
			fmt.Println("All playback tools are installed.")
			return nil
		}
		if !installSupported() {
			fmt.Println("Install these tools with your system package manager.")
			return nil
		}
		fmt.Fprint(os.Stderr, "Install all missing playback tools now? [y/N]: ")
		answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && len(answer) == 0 {
			return fmt.Errorf("read response: %w", err)
		}
		if !strings.EqualFold(strings.TrimSpace(answer), "y") && !strings.EqualFold(strings.TrimSpace(answer), "yes") {
			fmt.Println("No changes made.")
			return nil
		}
		return installPlaybackTools(missing)
	},
}

func installSupported() bool {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("where", "winget").Run() == nil
	case "darwin":
		return exec.Command("which", "brew").Run() == nil
	case "linux":
		return exec.Command("sh", "-c", "command -v apt-get || command -v dnf || command -v pacman").Run() == nil
	default:
		return false
	}
}

func installPlaybackTools(missing []string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		ids := map[string]string{"mpv": "shinchiro.mpv", "yt-dlp": "yt-dlp.yt-dlp", "ffmpeg": "Gyan.FFmpeg"}
		for _, name := range missing {
			args := []string{"install", "--force", "--exact", "--id", ids[name], "--source", "winget", "--accept-package-agreements", "--accept-source-agreements"}
			if err := runInstaller("winget", args...); err != nil {
				return err
			}
		}
	case "darwin":
		command = exec.Command("brew", append([]string{"install"}, missing...)...)
	case "linux":
		if exec.Command("sh", "-c", "command -v apt-get").Run() == nil {
			command = exec.Command("sudo", append([]string{"apt-get", "install", "-y"}, missing...)...)
		} else if exec.Command("sh", "-c", "command -v dnf").Run() == nil {
			command = exec.Command("sudo", append([]string{"dnf", "install", "-y"}, missing...)...)
		} else {
			command = exec.Command("sudo", append([]string{"pacman", "-Sy", "--needed"}, missing...)...)
		}
	default:
		return fmt.Errorf("automatic dependency installation is not supported on %s", runtime.GOOS)
	}
	if command != nil {
		command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("dependency installation failed: %w", err)
		}
	}
	fmt.Println("Install command completed. Reopen your terminal, then run 'kamui doctor' again to verify PATH.")
	return nil
}

func runInstaller(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

func init() { rootCmd.AddCommand(doctorCmd) }
