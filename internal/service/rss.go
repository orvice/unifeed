package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"butterfly.orx.me/core/log"
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
	Content     string    `json:"content,omitempty"`
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

	// 获取 item keys
	var items []map[string]interface{}
	prefix := fmt.Sprintf("feeds/%s/items/", feedName)

	// 列出所有匹配前缀的对象
	objectInfos, err := s.s3Client.ListObjects(ctx, prefix)
	if err != nil {
		logger.Error("Failed to list feed item keys from S3", err,
			"feed_name", feedName,
			"prefix", prefix,
		)
		metrics.S3OperationTotal.WithLabelValues("list", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("list", "s3_error").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "s3_error").Inc()
		return nil, fmt.Errorf("failed to list feed items: %w", err)
	}

	// 并行获取每个 item
	var wg sync.WaitGroup
	itemChan := make(chan map[string]interface{}, len(objectInfos))
	errChan := make(chan error, len(objectInfos))

	for _, objInfo := range objectInfos {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			var reader io.Reader
			var err error

			err = s.retryWithBackoff(ctx, "get_feed_item", func() error {
				reader, err = s.s3Client.GetObject(ctx, key)
				return err
			})

			if err != nil {
				logger.Error("Failed to get feed item from S3", err,
					"feed_name", feedName,
					"key", key,
				)
				errChan <- err
				return
			}

			// 读取数据
			data, err := io.ReadAll(reader)
			if err != nil {
				logger.Error("Failed to read feed item data", err,
					"feed_name", feedName,
					"key", key,
				)
				errChan <- err
				return
			}

			var item map[string]interface{}
			if err := json.Unmarshal(data, &item); err != nil {
				logger.Error("Failed to unmarshal feed item", err,
					"feed_name", feedName,
					"data_size", len(data),
				)
				errChan <- err
				return
			}

			itemChan <- item
		}(objInfo.Key)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	close(itemChan)
	close(errChan)

	// 处理错误
	if len(errChan) > 0 {
		err := <-errChan
		metrics.S3OperationTotal.WithLabelValues("get", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("get", "read_error").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "read_error").Inc()
		return nil, fmt.Errorf("failed to read some feed items: %w", err)
	}

	// 收集所有 item
	for item := range itemChan {
		// 确保摘要字段存在于结果中
		if summary, ok := item["custom"].(map[string]interface{})["summary"]; ok {
			item["summary"] = summary
		}
		items = append(items, item)
	}

	// 记录项目大小
	for _, item := range items {
		if data, err := json.Marshal(item); err == nil {
			metrics.FeedItemSize.WithLabelValues(feedName).Observe(float64(len(data)))
		}
	}

	// 更新缓存
	s.cache.Store(cacheKey, items)
	metrics.FeedCacheMisses.Inc()

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

	// 存储每个 item 到 S3
	var wg sync.WaitGroup
	errChan := make(chan error, len(items))

	for i, item := range items {
		wg.Add(1)
		go func(idx int, feedItem *gofeed.Item) {
			defer wg.Done()

			// 为 item 生成唯一标识符
			itemID := feedItem.GUID
			if itemID == "" {
				// 如果没有 GUID，使用链接或标题作为备选
				if feedItem.Link != "" {
					itemID = feedItem.Link
				} else {
					itemID = feedItem.Title
				}
			}

			// 创建安全的文件名
			safeID := s.sanitizeID(itemID)

			// 生成存储路径
			objectName := fmt.Sprintf("feeds/%s/items/%s.json", feedName, safeID)

			// 将项目转换为 JSON
			data, err := json.Marshal(feedItem)
			if err != nil {
				logger.Error("Failed to marshal feed item", err,
					"feed_name", feedName,
					"item_index", idx,
				)
				errChan <- fmt.Errorf("failed to marshal item: %w", err)
				return
			}

			// 存储到 S3
			if err := s.s3Client.PutObject(ctx, objectName, data, "application/json"); err != nil {
				logger.Error("Failed to store item in S3", err,
					"feed_name", feedName,
					"object_name", objectName,
				)
				errChan <- fmt.Errorf("failed to store item in S3: %w", err)
				return
			}

			logger.Debug("Successfully stored feed item",
				"feed_name", feedName,
				"object_name", objectName,
				"data_size", len(data),
			)

			metrics.S3OperationTotal.WithLabelValues("store", "success").Inc()
			metrics.S3ObjectSize.WithLabelValues("store").Observe(float64(len(data)))
		}(i, item)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	close(errChan)

	// 处理错误
	if len(errChan) > 0 {
		err := <-errChan
		metrics.S3OperationTotal.WithLabelValues("store", "error").Inc()
		metrics.S3OperationErrors.WithLabelValues("store", "s3_error").Inc()
		metrics.FeedErrors.WithLabelValues(feedName, "s3_error").Inc()
		return fmt.Errorf("failed to store some items in S3: %w", err)
	}

	// 更新缓存
	s.cache.Store(fmt.Sprintf("items:%s", feedName), items)

	logger.Info("Successfully stored all feed items",
		"feed_name", feedName,
		"item_count", len(items),
	)

	metrics.FeedItemsTotal.WithLabelValues(feedName).Set(float64(len(items)))

	return nil
}

// sanitizeID 清理标识符以便安全用作文件名
func (s *RssService) sanitizeID(id string) string {
	// 简单替换不安全的字符
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)

	// 如果 ID 过长，截断并添加哈希后缀
	safeID := replacer.Replace(id)
	if len(safeID) > 200 {
		h := sha256.New()
		h.Write([]byte(id))
		hash := fmt.Sprintf("%x", h.Sum(nil))[:8]
		safeID = safeID[:192] + "_" + hash
	}

	return safeID
}

// UpdateFeed 更新 Feed
func (s *RssService) UpdateFeed(ctx context.Context, feed conf.Feed) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.FeedUpdateDuration.WithLabelValues(feed.Name).Observe(duration)
	}()

	logger := log.FromContext(ctx).With("feed_name", feed.Name,
		"feed_url", feed.RssFeed)

	if feed.RssFeed == "" {
		err := fmt.Errorf("feed URL is required")
		logger.Error("Failed to update feed",
			"error", err,
		)
		metrics.FeedUpdateTotal.WithLabelValues(feed.Name, "error").Inc()
		return err
	}

	logger.Info("Starting feed update")

	// 解析 Feed
	parsedFeed, err := s.ParseFeed(ctx, feed.RssFeed)
	if err != nil {
		logger.Error("Failed to parse feed during update",
			"error", err,
		)
		metrics.FeedUpdateTotal.WithLabelValues(feed.Name, "error").Inc()
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	logger.Info("Parsed feed",

		"item_count", len(parsedFeed.Items),
	)

	// 为每个条目生成摘要
	items := parsedFeed.Items
	for i, item := range items {

		var content = item.Content
		if content == "" {
			content = item.Description
		}
		// 生成摘要
		summary, err := s.aiService.Summarize(ctx, content)
		if err != nil {
			logger.Error("Failed to generate summary",
				"error", err,
				"content", content,
				"item_index", i,
			)
			metrics.AISummaryErrors.WithLabelValues("summarize_error").Inc()
			continue // 继续处理其他条目
		}

		logger.Info("Summary",
			"summary", summary)

		// 创建自定义字段存储摘要，而不是覆盖内容
		if item.Custom == nil {
			items[i].Custom = make(map[string]string)
		}
		items[i].Custom["summary"] = summary
	}

	// 存储到 S3
	if err := s.StoreFeedItems(ctx, feed.Name, items); err != nil {
		logger.Error("Failed to store feed items during update",
			"error", err,
			"item_count", len(items),
		)
		metrics.FeedUpdateTotal.WithLabelValues(feed.Name, "error").Inc()
		return fmt.Errorf("failed to store feed items: %w", err)
	}

	logger.Info("Successfully updated feed",
		"item_count", len(items),
	)

	metrics.FeedUpdateTotal.WithLabelValues(feed.Name, "success").Inc()
	return nil
}

// FormatFeedItems 格式化 Feed 项目，确保内容包含摘要
func (s *RssService) FormatFeedItems(ctx context.Context, feedName string) ([]map[string]interface{}, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.FeedOperationLatency.WithLabelValues("format_feed_items").Observe(duration)
	}()

	// 获取存储的 Feed 项目
	items, err := s.GetStoredFeedItems(ctx, feedName)
	if err != nil {
		logger.Error("Failed to get feed items for formatting", err, "feed_name", feedName)
		metrics.FeedErrors.WithLabelValues(feedName, "format_error").Inc()
		return nil, fmt.Errorf("failed to get feed items: %w", err)
	}

	// 处理每个项目
	for i, item := range items {
		var content string
		var summary string

		// 获取内容
		if c, ok := item["content"]; ok && c != nil {
			content = fmt.Sprintf("%v", c)
		}

		// 获取摘要
		if item["custom"] != nil {
			if customMap, ok := item["custom"].(map[string]interface{}); ok {
				if s, ok := customMap["summary"]; ok && s != nil {
					summary = fmt.Sprintf("%v", s)
				}
			}
		}

		// 如果有摘要，将摘要添加到内容中
		if summary != "" {
			// 将摘要添加到内容开头，并用格式清晰地分隔
			formattedContent := fmt.Sprintf("**摘要**: %s\n\n---\n\n%s", summary, content)
			item["content"] = formattedContent

			// 保留单独的摘要字段
			item["summary"] = summary
		}

		// 确保返回的数据不包含任何可能导致序列化问题的类型
		items[i] = cleanupItemFields(item)
	}

	return items, nil
}

// cleanupItemFields 清理项目字段，确保数据类型适合JSON序列化
func cleanupItemFields(item map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 复制并清理必要的字段
	for k, v := range item {
		// 跳过custom字段，因为我们已经提取了摘要
		if k == "custom" {
			continue
		}

		// 确保字符串类型的正确性
		switch val := v.(type) {
		case string:
			result[k] = val
		case []byte:
			result[k] = string(val)
		case nil:
			// 跳过nil值
		default:
			result[k] = v
		}
	}

	return result
}
