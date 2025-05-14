package service

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

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
	return &MastodonService{client: &http.Client{Timeout: 10 * time.Second}}
}

// 拉取 Mastodon timeline 并生成 RSS XML
func (s *MastodonService) TimelineToRSS(feed conf.Feed) (string, error) {
	if feed.Mastodon.Host == "" || feed.Mastodon.Token == "" {
		return "", fmt.Errorf("mastodon config required")
	}
	url := fmt.Sprintf("%s/api/v1/timelines/home", feed.Mastodon.Host)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+feed.Mastodon.Token)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("mastodon api error: %s", resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// 这里只解析部分字段，实际可根据需要扩展
	var statuses []MastodonStatus
	if err := json.Unmarshal(body, &statuses); err != nil {
		return "", err
	}
	items := make([]Item, 0, len(statuses))
	for _, st := range statuses {
		items = append(items, Item{
			Title:       st.Content,
			Link:        st.URL,
			Description: st.Content,
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
