package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"vasset/proxy-service/internal/config"
	"vasset/proxy-service/internal/models"
)

// YtDLPClient 第三方 yt-dlp API 客户端
type YtDLPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewYtDLPClient 创建客户端
func NewYtDLPClient(cfg *config.YtDLPAPIConfig, logger *zap.Logger) *YtDLPClient {
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	return &YtDLPClient{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// Parse 解析视频信息
func (c *YtDLPClient) Parse(ctx context.Context, url string) (*models.ParseAPIResponse, error) {
	c.logger.Info("Parsing URL", zap.String("url", url))

	reqBody := map[string]string{"url": url}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/parse", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("parse API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result models.ParseAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Parse successful",
		zap.String("title", result.Title),
		zap.Int("formats", len(result.Formats)))

	return &result, nil
}

// StreamDownload 流式下载 - 返回 Reader
func (c *YtDLPClient) StreamDownload(ctx context.Context, streamReq *models.StreamRequest) (io.ReadCloser, string, int64, error) {
	c.logger.Info("Starting stream download",
		zap.String("format_id", streamReq.FormatID),
		zap.String("name", streamReq.Name))

	jsonBody, err := json.Marshal(streamReq)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/stream", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, "", 0, fmt.Errorf("stream API returned status %d: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	contentLength := resp.ContentLength

	c.logger.Info("Stream started",
		zap.String("content_type", contentType),
		zap.Int64("content_length", contentLength))

	return resp.Body, contentType, contentLength, nil
}

// SelectBestFormat 根据质量选择最佳格式
func SelectBestFormat(formats []models.Format, quality string, isVideo bool) *models.Format {
	targetHeight := parseQualityToHeight(quality)

	var bestFormat *models.Format
	var bestScore int

	for i := range formats {
		f := &formats[i]

		if isVideo {
			if !f.IsVideoFormat() {
				continue
			}
		} else {
			if f.IsVideoFormat() {
				continue
			}
			if !f.IsAudioFormat() {
				continue
			}
		}

		score := calculateFormatScore(f, targetHeight, isVideo)
		if score > bestScore {
			bestScore = score
			bestFormat = f
		}
	}

	return bestFormat
}

func parseQualityToHeight(quality string) int {
	switch quality {
	case "2160p", "4K":
		return 2160
	case "1440p":
		return 1440
	case "1080p":
		return 1080
	case "720p":
		return 720
	case "480p":
		return 480
	case "360p":
		return 360
	case "best":
		return 9999
	default:
		return 1080
	}
}

func calculateFormatScore(f *models.Format, targetHeight int, isVideo bool) int {
	score := 0

	if isVideo {
		if f.Height > 0 {
			if f.Height == targetHeight {
				score += 1000
			} else if f.Height < targetHeight {
				score += 500 - (targetHeight - f.Height)
			} else if targetHeight == 9999 {
				score += f.Height // best = 最高分辨率
			} else {
				score += 100
			}
		}

		if f.Ext == "mp4" {
			score += 50
		} else if f.Ext == "webm" {
			score += 30
		}
	} else {
		if f.ABR > 0 {
			score += int(f.ABR)
		}
		if f.Ext == "m4a" {
			score += 50
		}
	}

	return score
}
