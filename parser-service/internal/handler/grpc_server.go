package handler

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/parser-service/internal/service"
	"vasset/parser-service/internal/utils"
	pb "vasset/parser-service/proto"
)

// GRPCServer gRPC服务器
type GRPCServer struct {
	pb.UnimplementedParserServiceServer
	parserService *service.ParserService
	logger        *zap.Logger
}

// NewGRPCServer 创建gRPC服务器
func NewGRPCServer(parserService *service.ParserService, logger *zap.Logger) *GRPCServer {
	return &GRPCServer{
		parserService: parserService,
		logger:        logger,
	}
}

// ParseURL 解析视频URL
func (s *GRPCServer) ParseURL(ctx context.Context, req *pb.ParseURLRequest) (*pb.ParseURLResponse, error) {
	s.logger.Info("ParseURL request", zap.String("url", req.Url))

	// 调用解析服务
	result, err := s.parserService.ParseURL(ctx, req.Url, req.SkipCache)
	if err != nil {
		s.logger.Error("ParseURL failed", zap.String("url", req.Url), zap.Error(err))
		return nil, mapErrorToGRPCStatus(err)
	}

	// 转换格式列表
	formats := make([]*pb.VideoFormat, len(result.Formats))
	for i, f := range result.Formats {
		formats[i] = &pb.VideoFormat{
			FormatId:   f.FormatID,
			Quality:    f.Quality,
			Extension:  f.Extension,
			Filesize:   f.Filesize,
			Height:     int32(f.Height),
			Width:      int32(f.Width),
			Fps:        f.FPS,
			VideoCodec: f.VideoCodec,
			AudioCodec: f.AudioCodec,
			Vbr:        f.VBR,
			Abr:        f.ABR,
			Asr:        int32(f.ASR),
		}
	}

	return &pb.ParseURLResponse{
		VideoId:     result.VideoID,
		Platform:    result.Platform,
		Title:       result.Title,
		Description: result.Description,
		Duration:    result.Duration,
		Thumbnail:   result.Thumbnail,
		Author:      result.Author,
		UploadDate:  result.UploadDate,
		ViewCount:   result.ViewCount,
		Formats:     formats,
		CookieId:    result.CookieID, // 添加 cookie ID
		ProxyUrl:    result.ProxyURL, // 添加 proxy URL（目前为空）
	}, nil
}

// ValidateURL 验证URL是否有效
func (s *GRPCServer) ValidateURL(ctx context.Context, req *pb.ValidateURLRequest) (*pb.ValidateURLResponse, error) {
	s.logger.Info("ValidateURL request", zap.String("url", req.Url))

	valid, platform, message := s.parserService.ValidateURL(ctx, req.Url)

	return &pb.ValidateURLResponse{
		Valid:    valid,
		Platform: platform,
		Message:  message,
	}, nil
}

// mapErrorToGRPCStatus 将错误映射到gRPC状态码
func mapErrorToGRPCStatus(err error) error {
	switch err {
	case utils.ErrInvalidURL:
		return status.Error(codes.InvalidArgument, "invalid URL")
	case utils.ErrUnsupportedPlatform:
		return status.Error(codes.InvalidArgument, "unsupported platform")
	case utils.ErrVideoNotFound:
		return status.Error(codes.NotFound, "video not found")
	case utils.ErrVideoPrivate:
		return status.Error(codes.PermissionDenied, "video is private")
	case utils.ErrVideoDeleted:
		return status.Error(codes.NotFound, "video has been deleted")
	case utils.ErrGeoRestricted:
		return status.Error(codes.PermissionDenied, "video is geo-restricted")
	case utils.ErrAgeRestricted:
		return status.Error(codes.PermissionDenied, "video is age-restricted")
	case utils.ErrCopyrightClaim:
		return status.Error(codes.Unavailable, "video removed due to copyright claim")
	case utils.ErrTimeout:
		return status.Error(codes.DeadlineExceeded, "parse timeout")
	case utils.ErrYTDLPNotFound:
		return status.Error(codes.Internal, "yt-dlp binary not found")
	case utils.ErrYTDLPFailed:
		return status.Error(codes.Internal, "yt-dlp execution failed")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
