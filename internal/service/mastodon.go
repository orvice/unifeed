package service

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"time"

	"context"

	"github.com/mattn/go-mastodon"
	"go.orx.me/apps/unifeed/internal/conf"
)

type MastodonService struct {
	client *http.Client
}

type MastodonStatus struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
	Items []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func NewMastodonService() *MastodonService {
	return &MastodonService{}
}

// 拉取 Mastodon timeline 并生成 RSS XML
func (s *MastodonService) TimelineToRSS(feed conf.Feed) (string, error) {
	if feed.Mastodon.Host == "" || feed.Mastodon.Token == "" {
		return "", fmt.Errorf("mastodon config required")
	}
	client := mastodon.NewClient(&mastodon.Config{
		Server:      feed.Mastodon.Host,
		AccessToken: feed.Mastodon.Token,
	})
	ctx := context.Background()
	statuses, err := client.GetTimelineHome(ctx, nil)
	if err != nil {
		return "", err
	}
	items := make([]Item, 0, len(statuses))
	for _, st := range statuses {
		// 构建 media HTML
		mediaHTML := ""
		for _, m := range st.MediaAttachments {
			switch m.Type {
			case "image":
				mediaHTML += fmt.Sprintf(`<br><img src="%s" alt="%s"/>`, m.URL, m.Description)
			case "video", "gifv":
				mediaHTML += fmt.Sprintf(`<br><video controls src="%s" poster="%s">%s</video>`, m.URL, m.PreviewURL, m.Description)
			case "audio":
				mediaHTML += fmt.Sprintf(`<br><audio controls src="%s">%s</audio>`, m.URL, m.Description)
			}
		}
		items = append(items, Item{
			Title:       st.Content,
			Link:        st.URL,
			Description: st.Content + mediaHTML,
			PubDate:     st.CreatedAt.Format(time.RFC1123Z),
		})
	}
	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title: feed.Name,
			Link:  feed.Mastodon.Host,
			Items: items,
		},
	}
	out, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}
