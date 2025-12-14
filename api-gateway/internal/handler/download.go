package handler

import (
	"context"
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
	var req models.DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	// 1. 检查配额
	quotaResp, err := h.assetClient.CheckQuota(ctx, &pb.CheckQuotaRequest{
		UserId: userID,
	})
	if err != nil {
		models.InternalError(c, "failed to check quota: "+err.Error())
		return
	}
	if quotaResp.Remaining <= 0 {
		models.Forbidden(c, "quota exceeded, please try again tomorrow")
		return
	}

	// 2. 验证 URL
	validateResp, err := h.parserClient.ValidateURL(ctx, &pb.ValidateURLRequest{
		Url: req.URL,
	})
	if err != nil {
		models.InternalError(c, "failed to validate URL: "+err.Error())
		return
	}
	if !validateResp.Valid {
		models.BadRequest(c, "invalid URL: "+validateResp.Message)
		return
	}

	// 3. 解析获取标题
	parseResp, err := h.parserClient.ParseURL(ctx, &pb.ParseURLRequest{
		Url: req.URL,
	})
	if err != nil {
		models.InternalError(c, "failed to parse URL: "+err.Error())
		return
	}

	// 4. 生成任务 ID
	taskID := uuid.New().String()

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
		models.InternalError(c, "failed to create history: "+err.Error())
		return
	}

	// 6. 消费配额
	_, err = h.assetClient.ConsumeQuota(ctx, &pb.ConsumeQuotaRequest{
		UserId: userID,
	})
	if err != nil {
		models.InternalError(c, "failed to consume quota: "+err.Error())
		return
	}

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
	}

	if err := h.publisher.Publish(ctx, task); err != nil {
		models.InternalError(c, "failed to submit task: "+err.Error())
		return
	}

	// 8. 返回响应
	models.Accepted(c, models.DownloadResponse{
		TaskID:        taskID,
		HistoryID:     historyResp.HistoryId,
		EstimatedTime: estimateDownloadTime(parseResp.Duration, req.Quality),
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
