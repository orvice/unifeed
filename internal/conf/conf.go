package conf

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

var (
	Conf = new(Config)
)

type Config struct {
	Feeds     []Feed          `json:"feeds" yaml:"feeds"`
	S3        S3Config        `json:"s3" yaml:"s3"`
	AI        AIConfig        `json:"ai" yaml:"ai"`
	Scheduler SchedulerConfig `json:"scheduler" yaml:"scheduler"`
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
	RssFeed  string   `json:"rss_feed" yaml:"rss_feed"`
}

type S3Config struct {
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
	AccessKeyID     string `json:"access_key_id" yaml:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key" yaml:"secret_access_key"`
	UseSSL          bool   `json:"use_ssl" yaml:"use_ssl"`
	BucketName      string `json:"bucket_name" yaml:"bucket_name"`
}

type AIConfig struct {
	APIKey      string  `json:"api_key" yaml:"api_key"`
	Model       string  `json:"model" yaml:"model"`
	MaxTokens   int     `json:"max_tokens" yaml:"max_tokens"`
	Temperature float32 `json:"temperature" yaml:"temperature"`
}

type SchedulerConfig struct {
	UpdateInterval time.Duration `json:"update_interval" yaml:"update_interval"`
	MaxRetries     int           `json:"max_retries" yaml:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay" yaml:"retry_delay"`
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
		if feed.RssFeed != "" {
			slog.Info("RssFeed", "url", feed.RssFeed)
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

	// 验证 Feed 配置
	for _, feed := range c.Feeds {
		if feed.Name == "" {
			return fmt.Errorf("feed name required")
		}
		if feed.Mastodon.Host == "" && feed.Bluesky.Host == "" && feed.RssFeed == "" {
			return fmt.Errorf("feed %s: at least one source required", feed.Name)
		}
	}

	// 验证 S3 配置
	if c.S3.Endpoint == "" || c.S3.AccessKeyID == "" || c.S3.SecretAccessKey == "" || c.S3.BucketName == "" {
		return fmt.Errorf("S3 configuration incomplete")
	}

	// 验证 AI 配置
	if c.AI.APIKey == "" {
		return fmt.Errorf("AI API key required")
	}

	// 验证调度器配置
	if c.Scheduler.UpdateInterval == 0 {
		c.Scheduler.UpdateInterval = time.Hour
	}
	if c.Scheduler.MaxRetries == 0 {
		c.Scheduler.MaxRetries = 3
	}
	if c.Scheduler.RetryDelay == 0 {
		c.Scheduler.RetryDelay = time.Second * 5
	}

	return nil
}
