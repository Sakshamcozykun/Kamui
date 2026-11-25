package imdb

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type userConfig struct {
	OMDbAPIKey string `json:"omdb_api_key"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find user config directory: %w", err)
	}
	return filepath.Join(dir, "kamui", "config.json"), nil
}

// SetAPIKey saves the user's OMDb key in Kamui's per-user config file.
func SetAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(userConfig{OMDbAPIKey: key}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0600); err != nil {
		return fmt.Errorf("save Kamui config: %w", err)
	}
	return nil
}

func apiKey() (string, error) {
	if key := strings.TrimSpace(os.Getenv("OMDB_API_KEY")); key != "" {
		return key, nil
	}
	path, err := configPath()
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("OMDb API key is not configured; run 'kamui poster'")
		}
		return "", err
	}
	var cfg userConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return "", fmt.Errorf("read Kamui config: %w", err)
	}
	if cfg.OMDbAPIKey == "" {
		return "", fmt.Errorf("OMDb API key is not configured; run 'kamui poster'")
	}
	return strings.TrimSpace(cfg.OMDbAPIKey), nil
}
