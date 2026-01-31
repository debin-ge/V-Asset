package handler

import (
	"io"
	"log"
	"net/url"
	"strconv"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/middleware"
	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// StreamHandler 流式下载处理器
type StreamHandler struct {
	proxyClient pb.ProxyServiceClient
}

// NewStreamHandler 创建流式下载处理器
func NewStreamHandler(proxyClient pb.ProxyServiceClient) *StreamHandler {
	return &StreamHandler{
		proxyClient: proxyClient,
	}
}

// StreamDownload 流式下载 - 直接转发第三方流到客户端
func (h *StreamHandler) StreamDownload(c *gin.Context) {
	log.Printf("[Stream] Received stream download request from %s", c.ClientIP())

	// 1. 验证用户
	userID := middleware.GetUserID(c)
	if userID == "" {
		log.Printf("[Stream] ❌ User not authenticated")
		models.Unauthorized(c, "user not authenticated")
		return
	}

	// 2. 获取请求参数
	videoURL := c.Query("url")
	formatID := c.Query("format_id")
	name := c.Query("name")
	ext := c.Query("ext")
	isVideoStr := c.Query("is_video")

	if videoURL == "" {
		models.BadRequest(c, "url is required")
		return
	}

	isVideo := isVideoStr != "false"
	if ext == "" {
		if isVideo {
			ext = "mp4"
		} else {
			ext = "m4a"
		}
	}

	log.Printf("[Stream] ✓ Request - URL: %s, FormatID: %s, Name: %s, Ext: %s, IsVideo: %v",
		videoURL, formatID, name, ext, isVideo)

	// 3. 调用 proxy-service 的 StreamDownload
	req := &pb.StreamDownloadRequest{
		Url:      videoURL,
		FormatId: formatID,
		Name:     name,
		Ext:      ext,
		IsVideo:  isVideo,
	}

	stream, err := h.proxyClient.StreamDownload(c.Request.Context(), req)
	if err != nil {
		log.Printf("[Stream] ❌ Failed to start stream: %v", err)
		models.InternalError(c, "failed to start download: "+err.Error())
		return
	}

	// 4. 接收第一个 chunk（包含头信息）
	firstChunk, err := stream.Recv()
	if err != nil {
		log.Printf("[Stream] ❌ Failed to receive header: %v", err)
		models.InternalError(c, "failed to receive stream header: "+err.Error())
		return
	}

	// 5. 设置响应头
	contentType := firstChunk.ContentType
	if contentType == "" {
		if isVideo {
			contentType = "video/mp4"
		} else {
			contentType = "audio/mpeg"
		}
	}

	filename := name
	if filename == "" {
		filename = "download"
	}
	filename = filename + "." + ext

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))
	if firstChunk.ContentLength > 0 {
		c.Header("Content-Length", strconv.FormatInt(firstChunk.ContentLength, 10))
	}
	c.Header("Cache-Control", "no-cache")

	log.Printf("[Stream] ✓ Streaming started - ContentType: %s, ContentLength: %d, Filename: %s",
		contentType, firstChunk.ContentLength, filename)

	// 6. 开始写入响应（如果第一个 chunk 有数据）
	c.Status(200)

	if len(firstChunk.Data) > 0 {
		if _, err := c.Writer.Write(firstChunk.Data); err != nil {
			log.Printf("[Stream] ❌ Failed to write first chunk: %v", err)
			return
		}
	}

	// 7. 持续接收并写入数据
	totalBytes := int64(len(firstChunk.Data))
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("[Stream] ❌ Stream error: %v", err)
			return
		}

		if len(chunk.Data) > 0 {
			if _, err := c.Writer.Write(chunk.Data); err != nil {
				log.Printf("[Stream] ❌ Failed to write chunk: %v", err)
				return
			}
			totalBytes += int64(len(chunk.Data))
			c.Writer.Flush()
		}
	}

	log.Printf("[Stream] ✅ Stream completed - Total bytes: %d", totalBytes)
}
