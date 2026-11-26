// Package stream resolves supported, user-supplied media URLs for mpv.
package stream

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"
)

// Resolver recognizes and resolves one class of media URL.
type Resolver interface {
	Name() string
	Match(*url.URL) bool
	Resolve(context.Context, string) (string, error)
}

// Registry tries registered resolvers in order. Register additional resolvers
// here to add support for another platform without changing the CLI command.
type Registry struct {
	resolvers []Resolver
}

func NewRegistry(ytDLPPath string) *Registry {
	r := &Registry{}
	r.Register(DirectMedia{})
	r.Register(YtDlpPlatform{Label: "YouTube", Hosts: []string{"youtube.com", "youtu.be", "youtube-nocookie.com"}, YtDLPPath: ytDLPPath})
	r.Register(YtDlpPlatform{Label: "Vimeo", Hosts: []string{"vimeo.com", "player.vimeo.com"}, YtDLPPath: ytDLPPath})
	r.Register(YtDlpPlatform{Label: "Twitch", Hosts: []string{"twitch.tv", "clips.twitch.tv"}, YtDLPPath: ytDLPPath})
	return r
}

func (r *Registry) Register(resolver Resolver) {
	if resolver != nil {
		r.resolvers = append(r.resolvers, resolver)
	}
}

func (r *Registry) Resolve(ctx context.Context, rawURL string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return "", "", fmt.Errorf("enter a valid http or https media URL")
	}
	for _, resolver := range r.resolvers {
		if !resolver.Match(u) {
			continue
		}
		resolved, err := resolver.Resolve(ctx, u.String())
		if err != nil {
			return "", resolver.Name(), err
		}
		return resolved, resolver.Name(), nil
	}
	return "", "", fmt.Errorf("unsupported URL host %q; supported platforms are YouTube, Vimeo, Twitch, and direct media files", u.Hostname())
}

// DirectMedia passes through common direct media and manifest URLs. mpv handles
// the media protocol itself, so no site extraction is needed.
type DirectMedia struct{}

func (DirectMedia) Name() string { return "direct media" }

func (DirectMedia) Match(u *url.URL) bool {
	switch strings.ToLower(path.Ext(u.Path)) {
	case ".mp4", ".m4v", ".mkv", ".mov", ".webm", ".m3u8", ".mpd", ".ts":
		return true
	default:
		return false
	}
}

func (DirectMedia) Resolve(_ context.Context, rawURL string) (string, error) { return rawURL, nil }

// YtDlpPlatform recognizes platforms handled by mpv's yt-dlp hook. mpv owns
// format selection so it can pair separate audio/video tracks when needed.
type YtDlpPlatform struct {
	Label     string
	Hosts     []string
	YtDLPPath string
}

func (p YtDlpPlatform) Name() string { return p.Label }

func (p YtDlpPlatform) Match(u *url.URL) bool {
	host := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	for _, allowed := range p.Hosts {
		allowed = strings.ToLower(allowed)
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return true
		}
	}
	return false
}

func (p YtDlpPlatform) Resolve(ctx context.Context, rawURL string) (string, error) {
	if p.YtDLPPath == "" {
		return "", fmt.Errorf("yt-dlp is required; install it and ensure it is on PATH (run 'kamui doctor')")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return rawURL, nil
}
