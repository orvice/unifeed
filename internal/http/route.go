package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/service"
)

func Router(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	r.GET("/feeds/:name", func(c *gin.Context) {
		name := c.Param("name")
		var feed *conf.Feed
		for _, f := range conf.Conf.Feeds {
			if f.Name == name {
				feed = &f
				break
			}
		}
		if feed == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "feed not found"})
			return
		}
		if feed.Mastodon.Host != "" {
			svc := service.NewMastodonService()
			rss, err := svc.TimelineToRSS(*feed)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
			c.Header("Content-Type", "application/xml; charset=utf-8")
			c.String(http.StatusOK, rss)
			return
		}
		if feed.Bluesky.Host != "" {
			svc := service.NewBlueskyService()
			rss, err := svc.TimelineToRSS(*feed)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
			c.Header("Content-Type", "application/xml; charset=utf-8")
			c.String(http.StatusOK, rss)
			return
		}
		c.JSON(http.StatusNotImplemented, gin.H{"error": "unsupported feed type"})
	})
}
