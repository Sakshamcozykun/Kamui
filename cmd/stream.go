package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sakshamcozykun/kamui/internal/player"
	"github.com/sakshamcozykun/kamui/internal/stream"
)

var streamCmd = &cobra.Command{
	Use:   "stream <url>",
	Short: "Resolve a supported video URL and play it in mpv",
	Long: `Resolve and play a direct media URL or a supported YouTube, Vimeo, or Twitch URL.

Site extraction uses the locally installed yt-dlp. Playback is handled by mpv.
Only URLs you are authorized to access can be played; DRM-protected streams are
not decrypted or bypassed.

Examples:
  kamui stream https://www.youtube.com/watch?v=...
  kamui stream https://vimeo.com/123456789
  kamui stream https://www.twitch.tv/example
  kamui stream https://cdn.example.org/video.mp4`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := strings.TrimSpace(args[0])
		fmt.Fprintln(os.Stderr, "Resolving stream...")
		ytDLPPath, _ := player.LookPathTool("yt-dlp")
		registry := stream.NewRegistry(ytDLPPath)
		resolved, provider, err := registry.Resolve(cmd.Context(), input)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Resolved with %s; starting mpv...\n", provider)
		if err := player.Play(cmd.Context(), resolved, "Kamui stream"); err != nil {
			return fmt.Errorf("mpv playback failed: %w", err)
		}
		return nil
	},
}

func init() { rootCmd.AddCommand(streamCmd) }
