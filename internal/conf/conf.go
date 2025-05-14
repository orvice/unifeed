package conf

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

var (
	Conf = new(Config)
)

type Config struct {
	Feeds []Feed `json:"feeds" yaml:"feeds"`
}

type Mastodon struct {
	Host  string `json:"host" yaml:"host"`
	Token string `json:"token" yaml:"token"`
}

type Bluesky struct {
	Host      string `json:"host" yaml:"host"`
	Handle    string `json:"handle" yaml:"handle"`
	AppKey    string `json:"app_key" yaml:"app_key"`
	AppSecret string `json:"app_secret" yaml:"app_secret"`
}

type Feed struct {
	Name     string   `json:"name" yaml:"name"`
	Mastodon Mastodon `json:"mastodon" yaml:"mastodon"`
	Bluesky  Bluesky  `json:"bluesky" yaml:"bluesky"`
}

func (c *Config) Print() {
	for _, feed := range c.Feeds {
		slog.Info("Feed", "name", feed.Name)
		if feed.Mastodon.Host != "" {
			slog.Info("Mastodon", "host", feed.Mastodon.Host)
		}
		if feed.Bluesky.Host != "" {
			slog.Info("Bluesky", "host", feed.Bluesky.Host)
		}
	}
}

func LoadConfigFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()
	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Feeds) == 0 {
		return fmt.Errorf("no feeds configured")
	}
	for _, feed := range c.Feeds {
		if feed.Name == "" {
			return fmt.Errorf("feed name required")
		}
		if feed.Mastodon.Host == "" && feed.Bluesky.Host == "" {
			return fmt.Errorf("feed %s: at least one source required", feed.Name)
		}
	}
	return nil
}
