package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"vasset/proxy-service/internal/client"
	"vasset/proxy-service/internal/models"
	pb "vasset/proxy-service/proto"
)

// GRPCHandler gRPC 服务实现
type GRPCHandler struct {
	pb.UnimplementedProxyServiceServer
	ytdlpClient *client.YtDLPClient
	redisClient *redis.Client
	cacheTTL    time.Duration
	logger      *zap.Logger
}

// NewGRPCHandler 创建 Handler
func NewGRPCHandler(
	ytdlpClient *client.YtDLPClient,
	redisClient *redis.Client,
	cacheTTL time.Duration,
	logger *zap.Logger,
) *GRPCHandler {
	return &GRPCHandler{
		ytdlpClient: ytdlpClient,
		redisClient: redisClient,
		cacheTTL:    cacheTTL,
		logger:      logger,
	}
}

// Parse 解析视频信息
func (h *GRPCHandler) Parse(ctx context.Context, req *pb.ParseRequest) (*pb.ParseResponse, error) {
	h.logger.Info("Parse request", zap.String("url", req.Url))

	// 调用第三方 API
	result, err := h.ytdlpClient.Parse(ctx, req.Url)
	if err != nil {
		h.logger.Error("Parse failed", zap.Error(err))
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// 转换为 proto 格式
	formats := make([]*pb.ProxyVideoFormat, 0, len(result.Formats))
	for _, f := range result.Formats {
		// 过滤掉非视频/音频格式
		if !f.IsVideoFormat() && !f.IsAudioFormat() {
			continue
		}

		formats = append(formats, &pb.ProxyVideoFormat{
			FormatId:   f.FormatID,
			Quality:    f.GetQuality(),
			Extension:  f.Ext,
			Filesize:   f.GetFilesize(),
			Height:     int32(f.Height),
			Width:      int32(f.Width),
			Fps:        f.FPS,
			VideoCodec: f.VCodec,
			AudioCodec: f.ACodec,
			Vbr:        f.VBR,
			Abr:        f.ABR,
			Asr:        int32(f.ASR),
			FormatNote: f.FormatNote,
		})
	}

	return &pb.ParseResponse{
		VideoId:     result.ID,
		Platform:    detectPlatform(req.Url),
		Title:       result.Title,
		Description: "",
		Duration:    int64(result.Duration),
		Thumbnail:   result.Thumbnail,
		Author:      result.Uploader,
		UploadDate:  "",
		ViewCount:   result.ViewCount,
		Formats:     formats,
	}, nil
}

// Download 提交下载任务
func (h *GRPCHandler) Download(ctx context.Context, req *pb.DownloadRequest) (*pb.DownloadResponse, error) {
	h.logger.Info("Download request",
		zap.String("url", req.Url),
		zap.String("mode", req.Mode),
		zap.String("quality", req.Quality))

	// 先解析获取格式列表
	parseResult, err := h.ytdlpClient.Parse(ctx, req.Url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse for download: %w", err)
	}

	// 选择最佳格式
	isVideo := req.Mode != "audio_only"
	bestFormat := client.SelectBestFormat(parseResult.Formats, req.Quality, isVideo)
	if bestFormat == nil {
		return nil, fmt.Errorf("no suitable format found")
	}

	// 生成任务 ID
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	h.logger.Info("Download task created",
		zap.String("task_id", taskID),
		zap.String("format_id", bestFormat.FormatID))

	return &pb.DownloadResponse{
		TaskId:        taskID,
		HistoryId:     0,
		EstimatedTime: 60,
	}, nil
}

// StreamDownload 流式下载
func (h *GRPCHandler) StreamDownload(req *pb.StreamDownloadRequest, stream pb.ProxyService_StreamDownloadServer) error {
	h.logger.Info("StreamDownload request",
		zap.String("url", req.Url),
		zap.String("format_id", req.FormatId))

	ctx := stream.Context()

	// 构建请求
	streamReq := &models.StreamRequest{
		URL:      req.Url,
		FormatID: req.FormatId,
		Name:     req.Name,
		Ext:      req.Ext,
		IsVideo:  req.IsVideo,
	}

	// 获取流
	reader, contentType, contentLength, err := h.ytdlpClient.StreamDownload(ctx, streamReq)
	if err != nil {
		return fmt.Errorf("failed to start stream: %w", err)
	}
	defer reader.Close()

	// 发送头信息
	if err := stream.Send(&pb.StreamChunk{
		ContentType:   contentType,
		ContentLength: contentLength,
		IsHeader:      true,
	}); err != nil {
		return err
	}

	// 流式传输数据
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&pb.StreamChunk{
				Data: buf[:n],
			}); sendErr != nil {
				return sendErr
			}
		}
		if err != nil {
			break
		}
	}

	h.logger.Info("StreamDownload completed")
	return nil
}

// detectPlatform 检测平台
func detectPlatform(url string) string {
	url = strings.ToLower(url)
	switch {
	case strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be"):
		return "youtube"
	case strings.Contains(url, "bilibili.com"):
		return "bilibili"
	case strings.Contains(url, "tiktok.com"):
		return "tiktok"
	case strings.Contains(url, "twitter.com") || strings.Contains(url, "x.com"):
		return "twitter"
	default:
		return "unknown"
	}
}

// GetProgress 获取下载进度
func (h *GRPCHandler) GetProgress(ctx context.Context, req *pb.GetProgressRequest) (*pb.GetProgressResponse, error) {
	h.logger.Info("GetProgress request", zap.String("task_id", req.TaskId))

	// 调用第三方 API 获取进度
	result, err := h.ytdlpClient.GetProgress(ctx, req.TaskId)
	if err != nil {
		h.logger.Error("GetProgress failed", zap.Error(err))
		return nil, fmt.Errorf("failed to get progress: %w", err)
	}

	return &pb.GetProgressResponse{
		TaskId:          result.TaskID,
		Status:          result.Status,
		Progress:        result.Progress,
		Speed:           result.Speed,
		Eta:             int32(result.ETA),
		Error:           result.Error,
		Filename:        result.Filename,
		TotalBytes:      result.TotalBytes,
		DownloadedBytes: result.DownloadedBytes,
	}, nil
}
