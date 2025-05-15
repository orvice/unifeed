package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/service"
)

type Handler struct {
	rssService       *service.RssService
	schedulerService *service.SchedulerService
}

func NewHandler(rssService *service.RssService, schedulerService *service.SchedulerService) *Handler {
	return &Handler{
		rssService:       rssService,
		schedulerService: schedulerService,
	}
}

func (h *Handler) Router(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, World!",
		})
	})

	// 获取 Feed 内容
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

		// 处理不同类型的 Feed
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

		if feed.RssFeed != "" {
			// 获取存储的 Feed 项目
			items, err := h.rssService.GetStoredFeedItems(c.Request.Context(), feed.Name)
			if err != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, items)
			return
		}

		c.JSON(http.StatusNotImplemented, gin.H{"error": "unsupported feed type"})
	})

	// 手动触发 Feed 更新
	r.POST("/feeds/:name/update", func(c *gin.Context) {
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

		if feed.RssFeed == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "feed does not support RSS updates"})
			return
		}

		// 启动更新任务
		if err := h.schedulerService.StartJob(c.Request.Context(), *feed); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "update started"})
	})

	// 获取 Feed 更新状态
	r.GET("/feeds/:name/status", func(c *gin.Context) {
		name := c.Param("name")
		job, err := h.schedulerService.GetJobStatus(name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		status := gin.H{
			"name":      job.Feed.Name,
			"last_run":  job.LastRun.Format(time.RFC3339),
			"is_active": true,
		}
		if job.Error != nil {
			status["error"] = job.Error.Error()
		}

		c.JSON(http.StatusOK, status)
	})

	// 停止 Feed 更新
	r.POST("/feeds/:name/stop", func(c *gin.Context) {
		name := c.Param("name")
		if err := h.schedulerService.StopJob(name); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "update stopped"})
	})
}
