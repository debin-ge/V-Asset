package handler

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

// ParseHandler 解析处理器
type ParseHandler struct {
	parserClient pb.ParserServiceClient
	timeout      time.Duration
}

// NewParseHandler 创建解析处理器
func NewParseHandler(parserClient pb.ParserServiceClient, timeout time.Duration) *ParseHandler {
	return &ParseHandler{
		parserClient: parserClient,
		timeout:      timeout,
	}
}

// ParseURL 解析视频 URL
func (h *ParseHandler) ParseURL(c *gin.Context) {
	var req models.ParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.parserClient.ParseURL(ctx, &pb.ParseURLRequest{
		Url:       req.URL,
		SkipCache: req.SkipCache,
	})
	if err != nil {
		models.InternalError(c, "failed to parse URL: "+err.Error())
		return
	}

	// 调试日志：从 gRPC 接收的格式
	log.Printf("[DEBUG-ParseHandler] Received from gRPC: %d formats", len(resp.Formats))

	// 转换格式列表
	formats := make([]models.VideoFormat, 0, len(resp.Formats))
	var maxHeight int32
	var videoCount, audioCount int
	for i, f := range resp.Formats {
		// 统计
		if f.Height > maxHeight {
			maxHeight = f.Height
		}
		if f.VideoCodec != "" && f.VideoCodec != "none" {
			videoCount++
		}
		if f.AudioCodec != "" && f.AudioCodec != "none" && (f.VideoCodec == "" || f.VideoCodec == "none") {
			audioCount++
		}

		// 调试前3个格式
		if i < 3 {
			log.Printf("[DEBUG-ParseHandler] Format[%d]: id=%s, height=%d, video_codec=%s, audio_codec=%s, filesize=%d",
				i, f.FormatId, f.Height, f.VideoCodec, f.AudioCodec, f.Filesize)
		}

		formats = append(formats, models.VideoFormat{
			FormatID:   f.FormatId,
			Quality:    f.Quality,
			Extension:  f.Extension,
			Filesize:   f.Filesize,
			Height:     f.Height,
			Width:      f.Width,
			FPS:        f.Fps,
			VideoCodec: f.VideoCodec,
			AudioCodec: f.AudioCodec,
			VBR:        f.Vbr,
			ABR:        f.Abr,
			ASR:        f.Asr,
		})
	}

	log.Printf("[DEBUG-ParseHandler] Sending to frontend: %d formats (video=%d, audio=%d, maxHeight=%d)",
		len(formats), videoCount, audioCount, maxHeight)

	models.Success(c, models.ParseResponse{
		VideoID:     resp.VideoId,
		Platform:    resp.Platform,
		Title:       resp.Title,
		Description: resp.Description,
		Duration:    resp.Duration,
		Thumbnail:   resp.Thumbnail,
		Author:      resp.Author,
		UploadDate:  resp.UploadDate,
		ViewCount:   resp.ViewCount,
		Formats:     formats,
	})
}
