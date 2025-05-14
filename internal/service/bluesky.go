package service

import (
	"fmt"
	"net/http"
	"time"

	"go.orx.me/apps/unifeed/internal/conf"
)

type BlueskyService struct {
	client *http.Client
}

func NewBlueskyService() *BlueskyService {
	return &BlueskyService{client: &http.Client{Timeout: 10 * time.Second}}
}

// 拉取 Bluesky timeline 并生成 RSS XML（占位实现，需补充 API 细节）
func (s *BlueskyService) TimelineToRSS(feed conf.Feed) (string, error) {
	if feed.Bluesky.Host == "" || feed.Bluesky.Handle == "" {
		return "", fmt.Errorf("bluesky config required")
	}
	// TODO: 调用 Bluesky API 拉取 timeline，组装 RSS
	return "<rss version=\"2.0\"><channel><title>Bluesky feed (TODO)</title></channel></rss>", nil
}
