package main

import (
	"context"
	"log"
	"time"

	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	"github.com/gin-gonic/gin"
	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/dao"
	"go.orx.me/apps/unifeed/internal/http"
	"go.orx.me/apps/unifeed/internal/service"
)

var router func(*gin.Engine)

func main() {
	Init()
	app := NewApp()
	app.Run()
}

func NewApp() *app.App {
	app := core.New(&app.Config{
		Config:   conf.Conf,
		Service:  "unifeed",
		Router:   router,
		InitFunc: []func() error{},
	})
	return app
}

func Init() {

	s3Client, err := dao.NewS3Client()
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

	// 初始化 AI 服务
	aiService := service.NewAIService(conf.Conf.AI)

	// 初始化 RSS 服务
	rssConfig := service.RssConfig{
		MaxRetries: 3,
		RetryDelay: time.Second * 5,
	}
	rssService := service.NewRssService(aiService, s3Client, rssConfig)

	// 初始化调度器服务
	schedulerConfig := service.SchedulerConfig{
		UpdateInterval: conf.Conf.Scheduler.UpdateInterval,
		MaxRetries:     conf.Conf.Scheduler.MaxRetries,
		RetryDelay:     conf.Conf.Scheduler.RetryDelay,
	}
	schedulerService := service.NewSchedulerService(rssService, schedulerConfig)

	// 初始化 HTTP 处理器
	handler := http.NewHandler(rssService, schedulerService)
	router = handler.Router

	// 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 为每个 RSS feed 启动调度任务
	for _, feed := range conf.Conf.Feeds {
		if feed.RssFeed != "" {
			if err := schedulerService.StartJob(ctx, feed); err != nil {
				log.Printf("Failed to start job for feed %s: %v", feed.Name, err)
			}
		}
	}

}
