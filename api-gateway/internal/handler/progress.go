package handler

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/middleware"
	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// ProgressHandler 进度查询处理器
type ProgressHandler struct {
	proxyClient pb.ProxyServiceClient
	timeout     time.Duration
}

// NewProgressHandler 创建进度查询处理器
func NewProgressHandler(proxyClient pb.ProxyServiceClient, timeout time.Duration) *ProgressHandler {
	return &ProgressHandler{
		proxyClient: proxyClient,
		timeout:     timeout,
	}
}

// GetProgress 获取下载进度
func (h *ProgressHandler) GetProgress(c *gin.Context) {
	log.Printf("[Progress] Received progress request from %s", c.ClientIP())

	// 1. 验证用户
	userID := middleware.GetUserID(c)
	if userID == "" {
		log.Printf("[Progress] ❌ User not authenticated")
		models.Unauthorized(c, "user not authenticated")
		return
	}

	// 2. 获取 task_id
	taskID := c.Param("task_id")
	if taskID == "" {
		models.BadRequest(c, "task_id is required")
		return
	}

	log.Printf("[Progress] ✓ Request - TaskID: %s, UserID: %s", taskID, userID)

	// 3. 调用 proxy-service 获取进度
	req := &pb.GetProgressRequest{
		TaskId: taskID,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.proxyClient.GetProgress(ctx, req)
	if err != nil {
		log.Printf("[Progress] ❌ Failed to get progress: %v", err)
		models.InternalError(c, "failed to get progress: "+err.Error())
		return
	}

	log.Printf("[Progress] ✓ Progress: %s - %.1f%%", resp.Status, resp.Progress)

	// 4. 返回响应
	models.Success(c, gin.H{
		"task_id":          resp.TaskId,
		"status":           resp.Status,
		"progress":         resp.Progress,
		"speed":            resp.Speed,
		"eta":              resp.Eta,
		"error":            resp.Error,
		"filename":         resp.Filename,
		"total_bytes":      resp.TotalBytes,
		"downloaded_bytes": resp.DownloadedBytes,
	})
}
