package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/middleware"
	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// FileHandler 文件下载处理器
type FileHandler struct {
	assetClient pb.AssetServiceClient
	timeout     time.Duration
	bufferSize  int
}

// NewFileHandler 创建文件下载处理器
func NewFileHandler(assetClient pb.AssetServiceClient, timeout time.Duration, bufferSize int) *FileHandler {
	return &FileHandler{
		assetClient: assetClient,
		timeout:     timeout,
		bufferSize:  bufferSize,
	}
}

// DownloadFile 下载文件
func (h *FileHandler) DownloadFile(c *gin.Context) {
	// 获取历史 ID
	historyIDStr := c.Query("history_id")
	if historyIDStr == "" {
		models.BadRequest(c, "history_id is required")
		return
	}

	var historyID int64
	if _, err := fmt.Sscanf(historyIDStr, "%d", &historyID); err != nil {
		models.BadRequest(c, "invalid history_id")
		return
	}

	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	// 1. 获取文件信息
	resp, err := h.assetClient.GetFileInfo(ctx, &pb.GetFileInfoRequest{
		HistoryId: historyID,
		UserId:    userID,
	})
	if err != nil {
		models.Forbidden(c, "permission denied or file not found")
		return
	}

	// 2. 验证文件存在
	if resp.FilePath == "" {
		models.NotFound(c, "file not found")
		return
	}

	// 3. 打开文件
	file, err := os.Open(resp.FilePath)
	if err != nil {
		models.NotFound(c, "file not found on disk")
		return
	}
	defer file.Close()

	// 4. 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		models.InternalError(c, "failed to get file info")
		return
	}

	// 5. 设置响应头
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", resp.FileName))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	// 6. 支持断点续传
	c.Header("Accept-Ranges", "bytes")

	// 7. 流式传输文件
	c.Status(http.StatusOK)

	// 使用缓冲区流式传输
	buffer := make([]byte, h.bufferSize)
	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return
		}
		if _, err := c.Writer.Write(buffer[:n]); err != nil {
			return
		}
		c.Writer.Flush()
	}
}
