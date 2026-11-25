package adapter

import (
	"context"
	"fmt"
)

// VidSrc adapter.
type VidSrc struct{}

func NewVidSrc() *VidSrc { return &VidSrc{} }

func (v *VidSrc) Name() string { return "VidSrc" }

func (v *VidSrc) Extract(ctx context.Context, imdbID string) (string, error) {
	client := newClient()
	url := fmt.Sprintf("https://vidsrc.to/embed/movie/%s", imdbID)
	body, err := fetchPage(ctx, client, url, 512<<10)
	if err != nil {
		return "", err
	}
	if s := extractStream(ctx, client, body, url, 0); s != "" {
		return s, nil
	}
	return "", fmt.Errorf("no stream found")
}
