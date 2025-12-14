package handler

import (
	"context"
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

	// 转换格式列表
	formats := make([]models.VideoFormat, 0, len(resp.Formats))
	for _, f := range resp.Formats {
		formats = append(formats, models.VideoFormat{
			FormatID:   f.FormatId,
			Quality:    f.Quality,
			Extension:  f.Extension,
			Filesize:   f.Filesize,
			Height:     f.Height,
			FPS:        f.Fps,
			VideoCodec: f.VideoCodec,
			AudioCodec: f.AudioCodec,
		})
	}

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
