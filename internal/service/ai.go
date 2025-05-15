package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.orx.me/apps/unifeed/internal/conf"
	"go.orx.me/apps/unifeed/internal/logger"
)

type AiConfig struct {
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float32
	Endpoint    string
}

type AiService struct {
	client     *openai.Client
	config     conf.AIConfig
	maxRetries int
	retryDelay time.Duration
}

// NewAIService 创建一个新的 AI 服务实例
func NewAIService(config conf.AIConfig) *AiService {
	var client *openai.Client
	if config.Endpoint != "" {
		cfg := openai.DefaultConfig(config.APIKey)
		cfg.BaseURL = config.Endpoint
		client = openai.NewClientWithConfig(cfg)
	} else {
		client = openai.NewClient(config.APIKey)
	}

	// 设置默认值
	if config.Model == "" {
		config.Model = openai.GPT3Dot5Turbo
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 500
	}
	if config.Temperature == 0 {
		config.Temperature = 0.7
	}

	logger.Info("Initializing AI service",
		"model", config.Model,
		"max_tokens", config.MaxTokens,
		"temperature", config.Temperature,
	)

	return &AiService{
		client:     client,
		config:     config,
		maxRetries: 3,
		retryDelay: time.Second * 2,
	}
}

// SummarizeArticle 使用 OpenAI API 总结文章内容
func (s *AiService) SummarizeArticle(ctx context.Context, content string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("content cannot be empty")
	}

	// 如果内容太长，进行截断
	if len(content) > 4000 {
		content = content[:4000] + "..."
	}

	// 构建提示词
	prompt := fmt.Sprintf("请用中文总结以下文章的主要内容，突出关键点，并保持简洁：\n\n%s", content)

	var result string
	var err error
	for i := 0; i < s.maxRetries; i++ {
		result, err = s.callOpenAI(ctx, prompt)
		if err == nil {
			break
		}
		if i < s.maxRetries-1 {
			time.Sleep(s.retryDelay)
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to summarize after %d retries: %w", s.maxRetries, err)
	}

	return result, nil
}

// callOpenAI 调用 OpenAI API
func (s *AiService) callOpenAI(ctx context.Context, prompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: s.config.Model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
	}

	logger.Debug("Calling OpenAI API",
		"model", s.config.Model,
		"max_tokens", s.config.MaxTokens,
		"temperature", s.config.Temperature,
	)

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		logger.Error("OpenAI API call failed", err)
		return "", fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		err := fmt.Errorf("no choices returned from OpenAI")
		logger.Error("OpenAI API returned no choices", err)
		return "", err
	}

	result := strings.TrimSpace(resp.Choices[0].Message.Content)
	logger.Debug("OpenAI API call successful",
		"response_length", len(result),
		"usage", resp.Usage,
	)

	return result, nil
}

// SetMaxRetries 设置最大重试次数
func (s *AiService) SetMaxRetries(maxRetries int) {
	if maxRetries > 0 {
		s.maxRetries = maxRetries
	}
}

// SetRetryDelay 设置重试延迟时间
func (s *AiService) SetRetryDelay(delay time.Duration) {
	if delay > 0 {
		s.retryDelay = delay
	}
}

// GetModel 获取当前使用的模型
func (s *AiService) GetModel() string {
	return s.config.Model
}

// SetModel 设置使用的模型
func (s *AiService) SetModel(model string) {
	if model != "" {
		s.config.Model = model
	}
}

func (s *AiService) Summarize(ctx context.Context, content string) (string, error) {
	if content == "" {
		err := fmt.Errorf("content cannot be empty")
		logger.Error("Failed to summarize content", err)
		return "", err
	}

	// 如果内容太长，进行截断
	originalLength := len(content)
	if originalLength > 4000 {
		content = content[:4000] + "..."
		logger.Warn("Content truncated for summarization",
			"original_length", originalLength,
			"truncated_length", len(content),
		)
	}

	// 构建提示词
	prompt := fmt.Sprintf("请用中文总结以下文章的主要内容，突出关键点，并保持简洁：\n\n%s", content)

	var result string
	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		logger.Debug("Attempting to summarize content",
			"attempt", i+1,
			"max_retries", s.maxRetries,
		)

		result, lastErr = s.callOpenAI(ctx, prompt)
		if lastErr == nil {
			logger.Info("Successfully summarized content",
				"attempt", i+1,
				"result_length", len(result),
			)
			break
		}

		logger.Warn("Failed to summarize content, retrying",
			"attempt", i+1,
			"error", lastErr,
		)

		if i < s.maxRetries-1 {
			time.Sleep(s.retryDelay)
		}
	}

	if lastErr != nil {
		err := fmt.Errorf("failed to summarize after %d retries: %w", s.maxRetries, lastErr)
		logger.Error("Failed to summarize content after all retries", err)
		return "", err
	}

	return result, nil
}
