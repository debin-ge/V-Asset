package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/middleware"
	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// FileHandler 文件下载处理器
type FileHandler struct {
	assetClient    pb.AssetServiceClient
	timeout        time.Duration
	bufferSize     int
	billingEnabled bool
}

// NewFileHandler 创建文件下载处理器
func NewFileHandler(assetClient pb.AssetServiceClient, timeout time.Duration, bufferSize int, billingEnabled bool) *FileHandler {
	return &FileHandler{
		assetClient:    assetClient,
		timeout:        timeout,
		bufferSize:     bufferSize,
		billingEnabled: billingEnabled,
	}
}

// DownloadFile 下载文件
func (h *FileHandler) DownloadFile(c *gin.Context) {
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

	resp, err := h.assetClient.GetFileInfo(ctx, &pb.GetFileInfoRequest{
		HistoryId: historyID,
		UserId:    userID,
	})
	if err != nil {
		models.Forbidden(c, "permission denied or file not found")
		return
	}

	if resp.FilePath == "" {
		models.NotFound(c, "file not found")
		return
	}

	file, err := os.Open(resp.FilePath)
	if err != nil {
		models.NotFound(c, "file not found on disk")
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		models.InternalError(c, "failed to get file info")
		return
	}

	transferID := ""
	if h.billingEnabled {
		billingResp, err := h.assetClient.PrepareFileTransferBilling(ctx, &pb.PrepareFileTransferBillingRequest{
			UserId:        userID,
			HistoryId:     historyID,
			FileSizeBytes: fileInfo.Size(),
		})
		if err != nil {
			writeGRPCError(c, err)
			return
		}
		transferID = billingResp.GetTransferId()
	}

	dispositionFilename := buildContentDispositionFilename(resp.FileName)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", dispositionFilename)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Header("Accept-Ranges", "bytes")

	rc := http.NewResponseController(c.Writer)
	_ = rc.SetWriteDeadline(time.Time{})

	c.Status(http.StatusOK)

	buffer := make([]byte, h.bufferSize)
	var bytesSent int64
	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err == io.EOF {
				break
			}
			h.abortTransferBilling(transferID, "read file failed")
			return
		}
		written, writeErr := c.Writer.Write(buffer[:n])
		bytesSent += int64(written)
		if writeErr != nil {
			h.abortTransferBilling(transferID, "client disconnected")
			return
		}
		c.Writer.Flush()
	}

	if h.billingEnabled && transferID != "" {
		if bytesSent == fileInfo.Size() {
			completeCtx, cancel := context.WithTimeout(context.Background(), h.timeout)
			defer cancel()
			if _, err := h.assetClient.CompleteFileTransferBilling(completeCtx, &pb.CompleteFileTransferBillingRequest{
				TransferId:        transferID,
				ActualEgressBytes: bytesSent,
			}); err != nil {
				// 下载已经发给用户，账务补记失败只能记录日志
				fmt.Printf("[File] failed to complete transfer billing %s: %v\n", transferID, err)
			}
		} else {
			h.abortTransferBilling(transferID, "incomplete transfer")
		}
	}
}

func (h *FileHandler) abortTransferBilling(transferID, reason string) {
	if !h.billingEnabled || transferID == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()
	if _, err := h.assetClient.AbortFileTransferBilling(ctx, &pb.AbortFileTransferBillingRequest{
		TransferId: transferID,
		Reason:     reason,
	}); err != nil {
		fmt.Printf("[File] failed to abort transfer billing %s: %v\n", transferID, err)
	}
}

func buildContentDispositionFilename(filename string) string {
	if filename == "" {
		filename = "download"
	}

	timestamp := time.Now().Format("20060102_150405")
	lastDot := strings.LastIndex(filename, ".")
	if lastDot == -1 || lastDot == 0 {
		filename = fmt.Sprintf("%s_%s", filename, timestamp)
	} else {
		filename = fmt.Sprintf("%s_%s%s", filename[:lastDot], timestamp, filename[lastDot:])
	}

	fallback := buildASCIIFilename(filename)
	encoded := url.PathEscape(filename)

	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fallback, encoded)
}

func buildASCIIFilename(filename string) string {
	var builder strings.Builder
	builder.Grow(len(filename))

	for _, r := range filename {
		switch {
		case r == '"' || r == '\\':
			builder.WriteByte('_')
		case r < utf8.RuneSelf && r >= 32 && r != ';':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}

	cleaned := strings.TrimSpace(builder.String())
	if cleaned == "" {
		return "download"
	}
	return cleaned
}
