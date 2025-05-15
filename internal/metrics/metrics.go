package metrics

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Feed 相关指标
	FeedUpdateTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "feed_update_total",
			Help: "Total number of feed updates",
		},
		[]string{"feed_name", "status"},
	)

	FeedUpdateDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "feed_update_duration_seconds",
			Help:    "Duration of feed updates in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"feed_name"},
	)

	FeedItemsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "feed_items_total",
			Help: "Total number of items in each feed",
		},
		[]string{"feed_name"},
	)

	// 缓存相关指标
	FeedCacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "feed_cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	FeedCacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "feed_cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	FeedCacheSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "feed_cache_size",
			Help: "Current number of items in the cache",
		},
	)

	FeedCacheHitRatio = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "feed_cache_hit_ratio",
			Help: "Cache hit ratio (hits / (hits + misses))",
		},
	)

	FeedCacheEvictions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "feed_cache_evictions_total",
			Help: "Total number of cache evictions",
		},
		[]string{"reason"},
	)

	// 错误相关指标
	FeedErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "feed_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"feed_name", "error_type"},
	)

	FeedRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "feed_retries_total",
			Help: "Total number of retries by operation",
		},
		[]string{"operation"},
	)

	// 性能相关指标
	FeedOperationLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "feed_operation_latency_seconds",
			Help:    "Latency of feed operations in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"operation"},
	)

	FeedItemSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "feed_item_size_bytes",
			Help:    "Size of feed items in bytes",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"feed_name"},
	)

	// AI 服务相关指标
	AISummaryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_summary_total",
			Help: "Total number of AI summaries generated",
		},
		[]string{"status"},
	)

	AISummaryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_summary_duration_seconds",
			Help:    "Duration of AI summary generation in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"model"},
	)

	AISummaryTokens = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_summary_tokens",
			Help:    "Number of tokens used in AI summaries",
			Buckets: []float64{100, 200, 500, 1000, 2000, 4000},
		},
		[]string{"model"},
	)

	AISummaryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_summary_errors_total",
			Help: "Total number of AI summary errors",
		},
		[]string{"error_type"},
	)

	// S3 存储相关指标
	S3OperationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_operation_total",
			Help: "Total number of S3 operations",
		},
		[]string{"operation", "status"},
	)

	S3OperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "s3_operation_duration_seconds",
			Help:    "Duration of S3 operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	S3ObjectSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "s3_object_size_bytes",
			Help:    "Size of S3 objects in bytes",
			Buckets: []float64{1024, 10240, 102400, 1024000, 10240000},
		},
		[]string{"operation"},
	)

	S3OperationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "s3_operation_errors_total",
			Help: "Total number of S3 operation errors",
		},
		[]string{"operation", "error_type"},
	)

	// HTTP 服务相关指标
	HTTPRequestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	HTTPRequestErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_request_errors_total",
			Help: "Total number of HTTP request errors",
		},
		[]string{"method", "path", "error_type"},
	)

	// 系统资源相关指标
	MemoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
	)

	GoroutineCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutine_count",
			Help: "Current number of goroutines",
		},
	)
)

// 使用原子计数器来跟踪缓存命中率
var (
	cacheHits   atomic.Int64
	cacheMisses atomic.Int64
)

// UpdateCacheStats 更新缓存统计信息
func UpdateCacheStats(hit bool) {
	if hit {
		cacheHits.Add(1)
		FeedCacheHits.Inc()
	} else {
		cacheMisses.Add(1)
		FeedCacheMisses.Inc()
	}
	updateCacheHitRatio()
}

// updateCacheHitRatio 更新缓存命中率
func updateCacheHitRatio() {
	total := float64(cacheHits.Load() + cacheMisses.Load())
	if total > 0 {
		FeedCacheHitRatio.Set(float64(cacheHits.Load()) / total)
	}
}
