package adapter

import (
	"context"
	"fmt"
)

// VidLink adapter.
type VidLink struct{}

func NewVidLink() *VidLink { return &VidLink{} }

func (v *VidLink) Name() string { return "VidLink" }

func (v *VidLink) Extract(ctx context.Context, imdbID string) (string, error) {
	client := newClient()
	url := fmt.Sprintf("https://vidlink.pro/movie/%s", imdbID)
	body, err := fetchPage(ctx, client, url, 512<<10)
	if err != nil {
		return "", err
	}
	if s := extractStream(ctx, client, body, url, 0); s != "" {
		return s, nil
	}
	return "", fmt.Errorf("no stream found")
}
