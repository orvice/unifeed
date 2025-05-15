package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/dao"
	"go.orx.me/apps/unifeed/internal/http"
	"go.orx.me/apps/unifeed/internal/service"
)

func main() {
	// 加载配置
	config, err := conf.LoadConfigFromFile("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	config.Print()

	// 初始化 S3 客户端
	s3Config := dao.S3Config{
		Endpoint:        config.S3.Endpoint,
		AccessKeyID:     config.S3.AccessKeyID,
		SecretAccessKey: config.S3.SecretAccessKey,
		UseSSL:          config.S3.UseSSL,
		BucketName:      config.S3.BucketName,
	}
	s3Client, err := dao.NewS3Client(s3Config)
	if err != nil {
		log.Fatalf("Failed to initialize S3 client: %v", err)
	}

	// 初始化 AI 服务
	aiService := service.NewAIService(config.AI)

	// 初始化 RSS 服务
	rssConfig := service.RssConfig{
		MaxRetries: 3,
		RetryDelay: time.Second * 5,
	}
	rssService := service.NewRssService(aiService, s3Client, rssConfig)

	// 初始化调度器服务
	schedulerConfig := service.SchedulerConfig{
		UpdateInterval: config.Scheduler.UpdateInterval,
		MaxRetries:     config.Scheduler.MaxRetries,
		RetryDelay:     config.Scheduler.RetryDelay,
	}
	schedulerService := service.NewSchedulerService(rssService, schedulerConfig)

	// 初始化 HTTP 处理器
	handler := http.NewHandler(rssService, schedulerService)

	// 设置路由
	r := gin.Default()
	handler.Router(r)

	// 启动调度器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 为每个 RSS feed 启动调度任务
	for _, feed := range config.Feeds {
		if feed.RssFeed != "" {
			if err := schedulerService.StartJob(ctx, feed); err != nil {
				log.Printf("Failed to start job for feed %s: %v", feed.Name, err)
			}
		}
	}

	// 启动 HTTP 服务器
	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// 优雅关闭
	log.Println("Shutting down...")
	cancel()
}
