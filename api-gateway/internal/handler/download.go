package handler

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vasset/api-gateway/internal/middleware"
	"vasset/api-gateway/internal/models"
	"vasset/api-gateway/internal/mq"
	pb "vasset/api-gateway/proto"
)

// DownloadHandler 下载处理器
type DownloadHandler struct {
	assetClient  pb.AssetServiceClient
	parserClient pb.ParserServiceClient
	publisher    *mq.Publisher
	timeout      time.Duration
}

// NewDownloadHandler 创建下载处理器
func NewDownloadHandler(
	assetClient pb.AssetServiceClient,
	parserClient pb.ParserServiceClient,
	publisher *mq.Publisher,
	timeout time.Duration,
) *DownloadHandler {
	return &DownloadHandler{
		assetClient:  assetClient,
		parserClient: parserClient,
		publisher:    publisher,
		timeout:      timeout,
	}
}

// SubmitDownload 提交下载任务
func (h *DownloadHandler) SubmitDownload(c *gin.Context) {
	log.Printf("[Download] Received download request from %s", c.ClientIP())
	var req models.DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Download] ❌ Failed to parse request: %v", err)
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	log.Printf("[Download] ✓ Request parsed - URL: %s, Mode: %s, Quality: %s, Format: %s",
		req.URL, req.Mode, req.Quality, req.Format)

	userID := middleware.GetUserID(c)
	if userID == "" {
		log.Printf("[Download] ❌ User not authenticated")
		models.Unauthorized(c, "user not authenticated")
		return
	}
	log.Printf("[Download] ✓ User authenticated: %s", userID)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	log.Printf("[Download] Step 1/7: Checking quota for user %s...", userID)
	// 1. 检查配额
	quotaResp, err := h.assetClient.CheckQuota(ctx, &pb.CheckQuotaRequest{
		UserId: userID,
	})
	if err != nil {
		log.Printf("[Download] ❌ Failed to check quota: %v", err)
		models.InternalError(c, "failed to check quota: "+err.Error())
		return
	}
	log.Printf("[Download] ✓ Quota check passed - Remaining: %d", quotaResp.Remaining)
	if quotaResp.Remaining <= 0 {
		log.Printf("[Download] ❌ Quota exceeded for user %s", userID)
		models.Forbidden(c, "quota exceeded, please try again tomorrow")
		return
	}

	log.Printf("[Download] Step 2/7: Validating URL: %s...", req.URL)
	// 2. 验证 URL
	validateResp, err := h.parserClient.ValidateURL(ctx, &pb.ValidateURLRequest{
		Url: req.URL,
	})
	if err != nil {
		log.Printf("[Download] ❌ Failed to validate URL: %v", err)
		models.InternalError(c, "failed to validate URL: "+err.Error())
		return
	}
	if !validateResp.Valid {
		log.Printf("[Download] ❌ Invalid URL: %s", validateResp.Message)
		models.BadRequest(c, "invalid URL: "+validateResp.Message)
		return
	}
	log.Printf("[Download] ✓ URL validated - Platform: %s", validateResp.Platform)

	log.Printf("[Download] Step 3/7: Parsing URL to get metadata...")
	// 3. 解析获取标题
	parseResp, err := h.parserClient.ParseURL(ctx, &pb.ParseURLRequest{
		Url: req.URL,
	})
	if err != nil {
		log.Printf("[Download] ❌ Failed to parse URL: %v", err)
		models.InternalError(c, "failed to parse URL: "+err.Error())
		return
	}
	log.Printf("[Download] ✓ URL parsed - Title: %s, Duration: %ds", parseResp.Title, parseResp.Duration)

	log.Printf("[Download] Step 4/7: Generating task ID...")
	// 4. 生成任务 ID
	taskID := uuid.New().String()
	log.Printf("[Download] ✓ Task ID generated: %s", taskID)

	log.Printf("[Download] Step 5/7: Creating download history for task %s...", taskID)
	// 5. 创建下载历史
	historyResp, err := h.assetClient.CreateHistory(ctx, &pb.CreateHistoryRequest{
		UserId:   userID,
		TaskId:   taskID,
		Url:      req.URL,
		Platform: validateResp.Platform,
		Title:    parseResp.Title,
		Mode:     req.Mode,
		Quality:  req.Quality,
	})
	if err != nil {
		log.Printf("[Download] ❌ Failed to create history: %v", err)
		models.InternalError(c, "failed to create history: "+err.Error())
		return
	}
	log.Printf("[Download] ✓ History created - HistoryID: %d", historyResp.HistoryId)

	log.Printf("[Download] Step 6/7: Consuming quota for user %s...", userID)
	// 6. 消费配额
	_, err = h.assetClient.ConsumeQuota(ctx, &pb.ConsumeQuotaRequest{
		UserId: userID,
	})
	if err != nil {
		log.Printf("[Download] ❌ Failed to consume quota: %v", err)
		models.InternalError(c, "failed to consume quota: "+err.Error())
		return
	}
	log.Printf("[Download] ✓ Quota consumed")

	log.Printf("[Download] Step 7/7: Publishing task %s to RabbitMQ...", taskID)
	// 7. 发布下载任务到 RabbitMQ
	task := &mq.DownloadTask{
		TaskID:    taskID,
		UserID:    userID,
		HistoryID: historyResp.HistoryId,
		URL:       req.URL,
		Mode:      req.Mode,
		Quality:   req.Quality,
		Format:    req.Format,
		Platform:  validateResp.Platform,
		Title:     parseResp.Title,
		CookieID:  parseResp.CookieId, // 传递 parser 使用的 cookie ID
		ProxyURL:  parseResp.ProxyUrl, // 传递 parser 使用的 proxy URL
	}

	if err := h.publisher.Publish(ctx, task); err != nil {
		log.Printf("[Download] ❌ Failed to publish task to RabbitMQ: %v", err)
		models.InternalError(c, "failed to submit task: "+err.Error())
		return
	}
	log.Printf("[Download] ✓ Task %s published to RabbitMQ", taskID)

	// 8. 返回响应
	estimatedTime := estimateDownloadTime(parseResp.Duration, req.Quality)
	log.Printf("[Download] ✅ Download request completed successfully - TaskID: %s, EstimatedTime: %ds", taskID, estimatedTime)
	models.Accepted(c, models.DownloadResponse{
		TaskID:        taskID,
		HistoryID:     historyResp.HistoryId,
		EstimatedTime: estimatedTime,
	})
}

// estimateDownloadTime 估算下载时间
func estimateDownloadTime(duration int64, quality string) int {
	// 简单估算：视频时长 / 10 + 基础时间，根据质量调整
	base := int(duration / 10)
	if base < 30 {
		base = 30
	}

	switch quality {
	case "1080p":
		return base + 60
	case "720p":
		return base + 30
	default:
		return base + 15
	}
}
