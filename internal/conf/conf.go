package conf

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	Conf = new(Config)
)

type Config struct {
	Feeds []Feed
}

type Mastodon struct {
	Host  string
	Token string
}

type Bluesky struct {
	Host      string
	Handle    string
	AppKey    string
	AppSecret string
}

type Feed struct {
	Name     string
	Mastodon Mastodon
	Bluesky  Bluesky
}

func (c *Config) Print() {

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
