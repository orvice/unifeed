package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.orx.me/apps/unifeed/internal/conf"
)

type SchedulerConfig struct {
	UpdateInterval time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
}

type SchedulerService struct {
	rssService *RssService
	config     SchedulerConfig
	jobs       map[string]*Job
	mu         sync.RWMutex
}

type Job struct {
	Feed     conf.Feed
	StopChan chan struct{}
	LastRun  time.Time
	Error    error
}

// NewSchedulerService 创建一个新的调度器服务实例
func NewSchedulerService(rssService *RssService, cfg SchedulerConfig) *SchedulerService {
	if cfg.UpdateInterval == 0 {
		cfg.UpdateInterval = time.Hour
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = time.Second * 5
	}

	return &SchedulerService{
		rssService: rssService,
		config:     cfg,
		jobs:       make(map[string]*Job),
	}
}

// StartJob 启动一个 Feed 更新任务
func (s *SchedulerService) StartJob(ctx context.Context, feed conf.Feed) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[feed.Name]; exists {
		return fmt.Errorf("job already exists for feed: %s", feed.Name)
	}

	stopChan := make(chan struct{})
	job := &Job{
		Feed:     feed,
		StopChan: stopChan,
	}

	s.jobs[feed.Name] = job

	// 启动更新循环
	go s.runUpdateLoop(ctx, job)

	return nil
}

// StopJob 停止一个 Feed 更新任务
func (s *SchedulerService) StopJob(feedName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[feedName]
	if !exists {
		return fmt.Errorf("job not found for feed: %s", feedName)
	}

	close(job.StopChan)
	delete(s.jobs, feedName)

	return nil
}

// GetJobStatus 获取任务状态
func (s *SchedulerService) GetJobStatus(feedName string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[feedName]
	if !exists {
		return nil, fmt.Errorf("job not found for feed: %s", feedName)
	}

	return job, nil
}

// runUpdateLoop 运行更新循环
func (s *SchedulerService) runUpdateLoop(ctx context.Context, job *Job) {
	ticker := time.NewTicker(s.config.UpdateInterval)
	defer ticker.Stop()

	// 立即执行一次更新
	if err := s.updateFeed(ctx, job); err != nil {
		job.Error = err
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-job.StopChan:
			return
		case <-ticker.C:
			if err := s.updateFeed(ctx, job); err != nil {
				job.Error = err
			}
		}
	}
}

// updateFeed 更新单个 Feed
func (s *SchedulerService) updateFeed(ctx context.Context, job *Job) error {
	var lastErr error
	for i := 0; i < s.config.MaxRetries; i++ {
		// 解析 Feed
		items, err := s.rssService.ParseFeed(ctx, job.Feed.RssFeed)
		if err != nil {
			lastErr = fmt.Errorf("failed to parse feed: %w", err)
			time.Sleep(s.config.RetryDelay)
			continue
		}
		// 存储到 S3
		if err := s.rssService.StoreFeedItems(ctx, job.Feed.Name, items.Items); err != nil {
			lastErr = fmt.Errorf("failed to store feed items: %w", err)
			time.Sleep(s.config.RetryDelay)
			continue
		}

		// 更新成功
		job.LastRun = time.Now()
		job.Error = nil
		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", s.config.MaxRetries, lastErr)
}

// StartAllJobs 启动所有配置的 Feed 更新任务
func (s *SchedulerService) StartAllJobs(ctx context.Context, feeds []conf.Feed) error {
	for _, feed := range feeds {
		if feed.RssFeed == "" {
			continue
		}
		if err := s.StartJob(ctx, feed); err != nil {
			return fmt.Errorf("failed to start job for feed %s: %w", feed.Name, err)
		}
	}
	return nil
}

// StopAllJobs 停止所有 Feed 更新任务
func (s *SchedulerService) StopAllJobs() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for feedName, job := range s.jobs {
		close(job.StopChan)
		delete(s.jobs, feedName)
	}
}
