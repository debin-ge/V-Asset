package service

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"vasset/parser-service/internal/adapter"
	"vasset/parser-service/internal/cache"
	"vasset/parser-service/internal/config"
	"vasset/parser-service/internal/detector"
	"vasset/parser-service/internal/utils"
	"vasset/parser-service/internal/ytdlp"
)

// ParserService 解析服务
type ParserService struct {
	detector *detector.PlatformDetector
	cache    *cache.Service
	adapters map[string]adapter.Adapter
	limiter  *utils.ConcurrencyLimiter
	logger   *zap.Logger
}

// NewParserService 创建解析服务
func NewParserService(
	cfg *config.Config,
	cacheService *cache.Service,
	logger *zap.Logger,
) *ParserService {
	// 创建yt-dlp wrapper
	ytdlpWrapper := ytdlp.NewWrapper(&cfg.YTDLP)

	// 创建平台适配器
	adapters := make(map[string]adapter.Adapter)

	// YouTube适配器
	if platformCfg, ok := cfg.Platforms["youtube"]; ok && platformCfg.Enabled {
		adapters["youtube"] = adapter.NewYouTubeAdapter(ytdlpWrapper, platformCfg.CookieFile, platformCfg.ExtraArgs)
	}

	// Bilibili适配器
	if platformCfg, ok := cfg.Platforms["bilibili"]; ok && platformCfg.Enabled {
		adapters["bilibili"] = adapter.NewBilibiliAdapter(ytdlpWrapper, platformCfg.CookieFile, platformCfg.ExtraArgs)
	}

	// TikTok适配器
	if platformCfg, ok := cfg.Platforms["tiktok"]; ok && platformCfg.Enabled {
		adapters["tiktok"] = adapter.NewTikTokAdapter(ytdlpWrapper, platformCfg.CookieFile, platformCfg.ExtraArgs)
	}

	// 通用适配器
	if platformCfg, ok := cfg.Platforms["generic"]; ok && platformCfg.Enabled {
		adapters["generic"] = adapter.NewGenericAdapter(ytdlpWrapper)
	}

	return &ParserService{
		detector: detector.NewPlatformDetector(),
		cache:    cacheService,
		adapters: adapters,
		limiter:  utils.NewConcurrencyLimiter(cfg.YTDLP.MaxConcurrent),
		logger:   logger,
	}
}

// ParseURL 解析视频URL
func (s *ParserService) ParseURL(ctx context.Context, url string, skipCache bool) (*cache.ParseResult, error) {
	// 1. 标准化URL
	url = utils.NormalizeURL(url)

	// 2. 验证URL
	if !utils.IsValidURL(url) {
		return nil, utils.ErrInvalidURL
	}

	// 3. 检查缓存
	if !skipCache {
		if cached, err := s.cache.Get(ctx, url); err == nil {
			s.logger.Info("cache hit", zap.String("url", url))
			return cached, nil
		}
	}

	// 4. 检测平台
	platform, err := s.detector.Detect(url)
	if err != nil {
		return nil, err
	}

	// 5. 获取对应的适配器
	adpt, ok := s.adapters[platform]
	if !ok {
		// 如果没有对应适配器,使用通用适配器
		adpt, ok = s.adapters["generic"]
		if !ok {
			return nil, utils.ErrUnsupportedPlatform
		}
	}

	// 6. 并发控制
	s.limiter.Acquire()
	defer s.limiter.Release()

	// 7. 调用适配器解析
	s.logger.Info("parsing video",
		zap.String("url", url),
		zap.String("platform", platform))

	videoInfo, err := adpt.Parse(url)
	if err != nil {
		s.logger.Error("parse failed",
			zap.String("url", url),
			zap.Error(err))
		return nil, err
	}

	// 8. 标准化格式
	formats := utils.NormalizeFormats(videoInfo.Formats)

	// 9. 构造结果
	result := &cache.ParseResult{
		VideoID:     videoInfo.ID,
		Platform:    platform,
		Title:       utils.SanitizeString(videoInfo.Title),
		Description: utils.SanitizeString(videoInfo.Description),
		Duration:    videoInfo.Duration,
		Thumbnail:   videoInfo.Thumbnail,
		Author:      utils.SanitizeString(videoInfo.Uploader),
		UploadDate:  videoInfo.UploadDate,
		ViewCount:   videoInfo.ViewCount,
		Formats:     formats,
	}

	// 10. 写入缓存
	if err := s.cache.Set(ctx, url, result); err != nil {
		s.logger.Warn("cache set failed", zap.Error(err))
	}

	s.logger.Info("parse success",
		zap.String("url", url),
		zap.String("video_id", result.VideoID),
		zap.Int("format_count", len(result.Formats)))

	return result, nil
}

// ValidateURL 验证URL是否有效
func (s *ParserService) ValidateURL(ctx context.Context, url string) (bool, string, string) {
	// 1. 标准化URL
	url = utils.NormalizeURL(url)

	// 2. 验证URL格式
	if !utils.IsValidURL(url) {
		return false, "", "invalid URL format"
	}

	// 3. 检测平台
	platform, err := s.detector.Detect(url)
	if err != nil {
		return false, "", fmt.Sprintf("unsupported platform: %v", err)
	}

	// 4. 检查是否有对应的适配器
	_, ok := s.adapters[platform]
	if !ok {
		_, ok = s.adapters["generic"]
		if !ok {
			return false, platform, "no adapter available for this platform"
		}
	}

	return true, platform, ""
}
