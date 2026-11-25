package adapter

import (
	"context"
	"fmt"
)

// TwoEmbed adapter.
type TwoEmbed struct{}

func NewTwoEmbed() *TwoEmbed { return &TwoEmbed{} }

func (t *TwoEmbed) Name() string { return "2Embed" }

func (t *TwoEmbed) Extract(ctx context.Context, imdbID string) (string, error) {
	client := newClient()
	url := fmt.Sprintf("https://www.2embed.cc/embed/%s", imdbID)
	body, err := fetchPage(ctx, client, url, 512<<10)
	if err != nil {
		return "", err
	}
	if s := extractStream(ctx, client, body, url, 0); s != "" {
		return s, nil
	}
	return "", fmt.Errorf("no stream found")
}
