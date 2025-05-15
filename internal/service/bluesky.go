package service

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/ipfs/go-cid"
	"go.orx.me/apps/unifeed/internal/conf"
)

type BlueskyService struct {
	client *http.Client
}

func NewBlueskyService() *BlueskyService {
	return &BlueskyService{client: &http.Client{Timeout: 10 * time.Second}}
}

// 拉取 Bluesky timeline 并生成 RSS XML
func (s *BlueskyService) TimelineToRSS(feed conf.Feed) (string, error) {
	if feed.Bluesky.Host == "" || feed.Bluesky.Handle == "" {
		return "", fmt.Errorf("bluesky config required")
	}

	// 创建 XRPC 客户端
	client := &xrpc.Client{
		Host: feed.Bluesky.Host,
		Auth: &xrpc.AuthInfo{
			Handle: feed.Bluesky.Handle,
			Did:    feed.Bluesky.AppKey,
		},
	}

	// 获取用户 timeline
	ctx := context.Background()
	timeline, err := bsky.FeedGetTimeline(ctx, client, "", "", 50)
	if err != nil {
		return "", fmt.Errorf("get timeline: %w", err)
	}

	// 构建 RSS 内容
	items := make([]RSSItem, 0, len(timeline.Feed))
	for _, item := range timeline.Feed {
		if item.Post == nil {
			continue
		}

		// 解析帖子内容
		var postValue struct {
			Text  string `json:"text"`
			Embed *struct {
				Record *struct{ Text string } `json:"record"`
				Images []struct {
					Fullsize string `json:"fullsize"`
					Alt      string `json:"alt"`
				} `json:"images"`
			} `json:"embed"`
			Labels []struct{ Val string } `json:"labels"`
		}
		recordJSON, err := item.Post.Record.MarshalJSON()
		if err != nil {
			continue
		}
		if err := json.Unmarshal(recordJSON, &postValue); err != nil {
			continue
		}

		// 构建媒体内容
		mediaHTML := ""
		if postValue.Embed != nil {
			if postValue.Embed.Record != nil {
				// 处理引用帖子
				mediaHTML += fmt.Sprintf(`<br><blockquote>%s</blockquote>`, postValue.Embed.Record.Text)
			}
			if postValue.Embed.Images != nil {
				// 处理图片
				for _, img := range postValue.Embed.Images {
					mediaHTML += fmt.Sprintf(`<br><img src="%s" alt="%s"/>`, img.Fullsize, img.Alt)
				}
			}
		}

		// 构建标签
		categories := make([]string, 0, len(postValue.Labels))
		for _, label := range postValue.Labels {
			categories = append(categories, label.Val)
		}

		// 构建 RSS 条目
		authorName := ""
		if item.Post.Author.DisplayName != nil {
			authorName = *item.Post.Author.DisplayName
		}

		// 解析记录时间
		createdAt, err := time.Parse(time.RFC3339, item.Post.IndexedAt)
		if err != nil {
			continue
		}

		// 获取帖子 ID
		postID, err := cid.Parse(item.Post.Uri)
		if err != nil {
			continue
		}

		items = append(items, RSSItem{
			Title:       postValue.Text,
			Link:        fmt.Sprintf("https://bsky.app/profile/%s/post/%s", item.Post.Author.Handle, postID.String()),
			Description: postValue.Text + mediaHTML,
			PubDate:     createdAt.Format(time.RFC1123Z),
			GUID:        item.Post.Uri,
			Author:      authorName,
			Categories:  categories,
		})
	}

	// 构建 RSS feed
	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title: feed.Name,
			Link:  feed.Bluesky.Host,
			Items: items,
		},
	}

	// 转换为 XML
	out, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal rss: %w", err)
	}

	return string(out), nil
}
