package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/dao"
	"go.orx.me/apps/unifeed/internal/logger"
	"go.orx.me/apps/unifeed/internal/metrics"
)

type RssConfig struct {
	MaxRetries    int
	RetryDelay    time.Duration
	CacheDuration time.Duration
	MaxCacheSize  int
}

type cacheEntry struct {
	items      []map[string]interface{}
	expiresAt  time.Time
	lastAccess time.Time
}

type RssService struct {
	parser    *gofeed.Parser
	aiService *AiService
	s3Client  *dao.S3Client
	config    RssConfig
	cache     sync.Map
}

type FeedItem struct {
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description"`
	Published   time.Time `json:"published"`
	Summary     string    `json:"summary,omitempty"`
}

// NewRssService 创建一个新的 RSS 服务实例
func NewRssService(aiService *AiService, s3Client *dao.S3Client, config RssConfig) *RssService {
	logger.Info("Initializing RSS service",
		"max_retries", config.MaxRetries,
		"retry_delay", config.RetryDelay,
		"cache_duration", config.CacheDuration,
		"max_cache_size", config.MaxCacheSize,
	)

	// 设置默认值
	if config.CacheDuration == 0 {
		config.CacheDuration = 5 * time.Minute
	}
	if config.MaxCacheSize == 0 {
		config.MaxCacheSize = 100
	}

	return &RssService{
		parser:    gofeed.NewParser(),
		aiService: aiService,
		s3Client:  s3Client,
		config:    config,
	}
}

// retryWithBackoff 执行带退避的重试逻辑
func (s *RssService) retryWithBackoff(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	for i := 0; i <= s.config.MaxRetries; i++ {
		if i > 0 {
			logger.Info("Retrying operation",
				"operation", operation,
				"attempt", i,
				"max_retries", s.config.MaxRetries,
				"delay", s.config.RetryDelay,
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.config.RetryDelay):
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			logger.Warn("Operation failed",
				"operation", operation,
				"attempt", i+1,
				"error", err,
			)
			continue
		}

		return nil
	}

	return fmt.Errorf("operation %s failed after %d retries: %w", operation, s.config.MaxRetries, lastErr)
}

// ParseFeed 解析 RSS Feed
func (s *RssService) ParseFeed(ctx context.Context, url string) (*gofeed.Feed, error) {
	logger.Info("Parsing feed", "url", url)
	start := time.Now()
	defer func() {
		metrics.FeedOperationLatency.WithLabelValues("parse_feed").Observe(time.Since(start).Seconds())
	}()

	// 检查缓存
	if cached, ok := s.cache.Load(url); ok {
		logger.Debug("Cache hit for feed", "url", url)
		metrics.UpdateCacheStats(true)
		if feed, ok := cached.(*gofeed.Feed); ok {
			return feed, nil
		}
	}
	metrics.UpdateCacheStats(false)

	// 获取 Feed 内容
	resp, err := http.Get(url)
	if err != nil {
		metrics.FeedErrors.WithLabelValues(url, "http_error").Inc()
		return nil, fmt.Errorf("failed to fetch feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		metrics.FeedErrors.WithLabelValues(url, "http_status_error").Inc()
		return nil, fmt.Errorf("failed to fetch feed: status code %d", resp.StatusCode)
	}

	// 解析 Feed
	parser := gofeed.NewParser()
	feed, err := parser.Parse(resp.Body)
	if err != nil {
		metrics.FeedErrors.WithLabelValues(url, "parse_error").Inc()
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	// 更新缓存
	s.cache.Store(url, feed)
	metrics.FeedCacheSize.Inc()

	return feed, nil
}

// GetStoredFeedItems 从缓存或 S3 获取存储的 Feed 项目
func (s *RssService) GetStoredFeedItems(ctx context.Context, feedName string) ([]map[string]interface{}, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.FeedUpdateDuration.WithLabelValues("get").Observe(duration)
		metrics.FeedOperationLatency.WithLabelValues("get_feed_items").Observe(duration)
	}()

	// 检查缓存
	cacheKey := fmt.Sprintf("items:%s", feedName)
	if cached, ok := s.cache.Load(cacheKey); ok {
		logger.Debug("Cache hit for stored items", "feed_name", feedName)
		metrics.UpdateCacheStats(true)
		items := cached.([]map[string]interface{})
		return items, nil
	}
	metrics.UpdateCacheStats(false)

	logger.Info("Retrieving stored feed items", "feed_name", feedName)

	// 从 S3 获取存储的 Feed 项目
	key := fmt.Sprintf("feeds/%s/items.json", feedName)
	var reader io.Reader
	var err error

	err = s.retryWithBackoff(ctx, "get_feed_items", func() error {
		reader, err = s.s3Client.GetObject(ctx, key)
		return err
	})

	if err != nil {
		logger.Error("Failed to get feed items from S3", err,
			"feed_name", feedName,
			"key", key,
		)
		metrics.S3OperationTotal.WithLabelValues("get", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("get", "s3_error").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "s3_error").Inc()
		return nil, fmt.Errorf("failed to get feed items: %w", err)
	}

	// 读取数据
	data, err := io.ReadAll(reader)
	if err != nil {
		logger.Error("Failed to read feed items data", err,
			"feed_name", feedName,
			"key", key,
		)
		metrics.S3OperationTotal.WithLabelValues("get", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("get", "read_error").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "read_error").Inc()
		return nil, fmt.Errorf("failed to read feed items: %w", err)
	}

	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		logger.Error("Failed to unmarshal feed items", err,
			"feed_name", feedName,
			"data_size", len(data),
		)
		metrics.S3OperationTotal.WithLabelValues("get", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("get", "unmarshal_error").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "unmarshal_error").Inc()
		return nil, fmt.Errorf("failed to unmarshal feed items: %w", err)
	}

	// 更新缓存
	s.cache.Store(cacheKey, items)
	metrics.FeedCacheMisses.Inc()

	// 记录项目大小
	for _, item := range items {
		if data, err := json.Marshal(item); err == nil {
			metrics.FeedItemSize.WithLabelValues(feedName).Observe(float64(len(data)))
		}
	}

	return items, nil
}

// StoreFeedItems 将 Feed 项目存储到 S3 并更新缓存
func (s *RssService) StoreFeedItems(ctx context.Context, feedName string, items []*gofeed.Item) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.FeedUpdateDuration.WithLabelValues("store").Observe(duration)
		metrics.FeedOperationLatency.WithLabelValues("store_feed_items").Observe(duration)
	}()

	if s.s3Client == nil {
		err := fmt.Errorf("S3 client not configured")
		logger.Error("Failed to store feed items", err)
		metrics.S3OperationTotal.WithLabelValues("store", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("store", "client_not_configured").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "client_not_configured").Inc()
		return err
	}

	logger.Info("Storing feed items",
		"feed_name", feedName,
		"item_count", len(items),
	)

	// 将项目转换为 JSON
	data, err := json.Marshal(items)
	if err != nil {
		logger.Error("Failed to marshal feed items", err)
		metrics.S3OperationTotal.WithLabelValues("store", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("store", "marshal_error").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "marshal_error").Inc()
		return fmt.Errorf("failed to marshal items: %w", err)
	}

	// 生成存储路径
	objectName := fmt.Sprintf("feeds/%s/items.json", feedName)

	// 存储到 S3
	if err := s.s3Client.PutObject(ctx, objectName, data, "application/json"); err != nil {
		logger.Error("Failed to store items in S3", err,
			"feed_name", feedName,
			"object_name", objectName,
		)
		metrics.S3OperationTotal.WithLabelValues("store", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("store", "s3_error").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "s3_error").Inc()
		return fmt.Errorf("failed to store items in S3: %w", err)
	}

	// 更新缓存
	s.cache.Store(fmt.Sprintf("items:%s", feedName), items)

	logger.Info("Successfully stored feed items",
		"feed_name", feedName,
		"object_name", objectName,
		"data_size", len(data),
	)

	metrics.S3OperationTotal.WithLabelValues("store", "success").Inc()
	metrics.S3ObjectSize.WithLabelValues("store").Observe(float64(len(data)))
	metrics.FeedItemsTotal.WithLabelValues(feedName).Set(float64(len(items)))

	return nil
}

// UpdateFeed 更新 Feed
func (s *RssService) UpdateFeed(ctx context.Context, feed conf.Feed) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.FeedUpdateDuration.WithLabelValues(feed.Name).Observe(duration)
	}()

	if feed.RssFeed == "" {
		err := fmt.Errorf("feed URL is required")
		logger.Error("Failed to update feed", err,
			"feed_name", feed.Name,
		)
		metrics.FeedUpdateTotal.WithLabelValues(feed.Name, "error").Inc()
		return err
	}

	logger.Info("Starting feed update",
		"feed_name", feed.Name,
		"feed_url", feed.RssFeed,
	)

	// 解析 Feed
	parsedFeed, err := s.ParseFeed(ctx, feed.RssFeed)
	if err != nil {
		logger.Error("Failed to parse feed during update", err,
			"feed_name", feed.Name,
			"feed_url", feed.RssFeed,
		)
		metrics.FeedUpdateTotal.WithLabelValues(feed.Name, "error").Inc()
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	// 为每个条目生成摘要
	items := parsedFeed.Items
	for i, item := range items {
		// 生成摘要
		summary, err := s.aiService.Summarize(ctx, item.Content)
		if err != nil {
			logger.Error("Failed to generate summary", fmt.Errorf("summary generation failed: %w", err), "item_index", i)
			metrics.AISummaryErrors.WithLabelValues("summarize_error").Inc()
			continue // 继续处理其他条目
		}
		items[i].Content = summary
	}

	// 存储到 S3
	if err := s.StoreFeedItems(ctx, feed.Name, items); err != nil {
		logger.Error("Failed to store feed items during update", err,
			"feed_name", feed.Name,
			"item_count", len(items),
		)
		metrics.FeedUpdateTotal.WithLabelValues(feed.Name, "error").Inc()
		return fmt.Errorf("failed to store feed items: %w", err)
	}

	logger.Info("Successfully updated feed",
		"feed_name", feed.Name,
		"item_count", len(items),
	)

	metrics.FeedUpdateTotal.WithLabelValues(feed.Name, "success").Inc()
	return nil
}
