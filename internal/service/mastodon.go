package service

import (
	"encoding/xml"
	"fmt"
	"time"

	"context"

	"github.com/mattn/go-mastodon"
	"go.orx.me/apps/unifeed/internal/conf"
)

type MastodonService struct {
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

type RSSItem struct {
	Title       string     `xml:"title"`
	Link        string     `xml:"link"`
	Description string     `xml:"description"`
	Content     string     `xml:"content"`
	PubDate     string     `xml:"pubDate"`
	GUID        string     `xml:"guid"`
	Author      string     `xml:"author,omitempty"`
	Categories  []string   `xml:"category,omitempty"`
	Enclosure   *Enclosure `xml:"enclosure,omitempty"`
	Image       string     `xml:"image,omitempty"`
}

type Enclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr,omitempty"`
	Length string `xml:"length,attr,omitempty"`
}

type Channel struct {
	Title string    `xml:"title"`
	Link  string    `xml:"link"`
	Items []RSSItem `xml:"item"`
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
	items := make([]RSSItem, 0, len(statuses))
	for _, st := range statuses {
		status := st
		isReblog := false
		if st.Reblog != nil {
			status = st.Reblog
			isReblog = true
		}

		// 构建 media HTML
		mediaHTML := ""
		for _, m := range status.MediaAttachments {
			switch m.Type {
			case "image":
				mediaHTML += fmt.Sprintf(`<br><img src="%s" alt="%s"/>`, m.URL, m.Description)
			case "video", "gifv":
				mediaHTML += fmt.Sprintf(`<br><video controls src="%s" poster="%s">%s</video>`, m.URL, m.PreviewURL, m.Description)
			case "audio":
				mediaHTML += fmt.Sprintf(`<br><audio controls src="%s">%s</audio>`, m.URL, m.Description)
			}
		}
		author := status.Account.DisplayName
		categories := make([]string, 0, len(status.Tags))
		for _, tag := range status.Tags {
			categories = append(categories, tag.Name)
		}
		var enclosure *Enclosure
		for _, m := range status.MediaAttachments {
			if m.URL != "" {
				enclosure = &Enclosure{
					URL:  m.URL,
					Type: m.Type,
				}
				break
			}
		}
		image := ""
		if len(status.MediaAttachments) > 0 && status.MediaAttachments[0].PreviewURL != "" {
			image = status.MediaAttachments[0].PreviewURL
		}

		// Construct title using author's handle and nickname
		title := fmt.Sprintf("%s (@%s)", status.Account.DisplayName, status.Account.Acct)
		link := status.URL

		description := status.Content + mediaHTML
		if isReblog {
			// 拼接原作者和转嘟者
			origAuthor := status.Account.Acct
			if origAuthor == "" {
				origAuthor = status.Account.DisplayName
			}
			reblogger := status.Reblog.Account.Acct
			if reblogger == "" {
				reblogger = status.Reblog.Account.DisplayName
			}
			description = fmt.Sprintf("@%s 转嘟 @%s: %s%s", reblogger, origAuthor, status.Content, mediaHTML)
		}

		items = append(items, RSSItem{
			Title:       title,
			Link:        link,
			Description: description,
			Content:     description,
			PubDate:     st.CreatedAt.Format(time.RFC1123Z),
			GUID:        string(st.ID),
			Author:      author,
			Categories:  categories,
			Enclosure:   enclosure,
			Image:       image,
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
