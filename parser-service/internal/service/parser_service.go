package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"vasset/parser-service/internal/adapter"
	"vasset/parser-service/internal/cache"
	"vasset/parser-service/internal/client"
	"vasset/parser-service/internal/config"
	"vasset/parser-service/internal/detector"
	"vasset/parser-service/internal/utils"
	"vasset/parser-service/internal/ytdlp"
)

// ParserService 解析服务
type ParserService struct {
	detector      *detector.PlatformDetector
	cache         *cache.Service
	adapters      map[string]adapter.Adapter
	limiter       *utils.ConcurrencyLimiter
	logger        *zap.Logger
	assetClient   *client.AssetClient
	enableCookies bool
}

// NewParserService 创建解析服务
func NewParserService(
	cfg *config.Config,
	cacheService *cache.Service,
	logger *zap.Logger,
) *ParserService {
	// 创建yt-dlp wrapper
	ytdlpWrapper := ytdlp.NewWrapper(&cfg.YTDLP)

	// 创建平台适配器（不传递静态 cookie 文件，改为动态获取）
	adapters := make(map[string]adapter.Adapter)

	// YouTube适配器
	if platformCfg, ok := cfg.Platforms["youtube"]; ok && platformCfg.Enabled {
		adapters["youtube"] = adapter.NewYouTubeAdapter(ytdlpWrapper, "", platformCfg.ExtraArgs)
	}

	// Bilibili适配器
	if platformCfg, ok := cfg.Platforms["bilibili"]; ok && platformCfg.Enabled {
		adapters["bilibili"] = adapter.NewBilibiliAdapter(ytdlpWrapper, "", platformCfg.ExtraArgs)
	}

	// TikTok适配器
	if platformCfg, ok := cfg.Platforms["tiktok"]; ok && platformCfg.Enabled {
		adapters["tiktok"] = adapter.NewTikTokAdapter(ytdlpWrapper, "", platformCfg.ExtraArgs)
	}

	// 通用适配器
	if platformCfg, ok := cfg.Platforms["generic"]; ok && platformCfg.Enabled {
		adapters["generic"] = adapter.NewGenericAdapter(ytdlpWrapper)
	}

	// 创建 Asset 客户端（可选）
	var assetClient *client.AssetClient
	if cfg.AssetService.EnableCookies && cfg.AssetService.Addr != "" {
		var err error
		assetClient, err = client.NewAssetClient(
			cfg.AssetService.Addr,
			cfg.AssetService.Timeout,
			cfg.AssetService.CookieTempDir,
		)
		if err != nil {
			logger.Warn("failed to create asset client, cookie feature disabled", zap.Error(err))
		} else {
			logger.Info("asset client created successfully",
				zap.String("addr", cfg.AssetService.Addr),
				zap.Bool("cookies_enabled", true))
		}
	} else {
		logger.Info("asset service not configured",
			zap.String("addr", cfg.AssetService.Addr),
			zap.Bool("enable_cookies", cfg.AssetService.EnableCookies))
	}

	return &ParserService{
		detector:      detector.NewPlatformDetector(),
		cache:         cacheService,
		adapters:      adapters,
		limiter:       utils.NewConcurrencyLimiter(cfg.YTDLP.MaxConcurrent),
		logger:        logger,
		assetClient:   assetClient,
		enableCookies: cfg.AssetService.EnableCookies && assetClient != nil,
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

	s.logger.Info("acquired concurrency slot",
		zap.String("url", url),
		zap.String("platform", platform))

	// 7. 获取Cookie（如果启用）
	var cookieFile string
	var cookieID int64
	if s.enableCookies && s.assetClient != nil {
		s.logger.Info("attempting to get cookie",
			zap.String("platform", platform))

		var err error
		cookieFile, cookieID, err = s.assetClient.GetAvailableCookie(platform)
		if err != nil {
			s.logger.Warn("failed to get cookie, continuing without cookie",
				zap.String("platform", platform),
				zap.Error(err))
		} else if cookieFile != "" {
			s.logger.Info("using cookie for parsing",
				zap.String("platform", platform),
				zap.String("cookie_file", cookieFile),
				zap.Int64("cookie_id", cookieID))
			defer func() {
				if cleanErr := s.assetClient.CleanupCookieFile(cookieFile); cleanErr != nil {
					s.logger.Warn("failed to cleanup cookie file",
						zap.String("cookie_file", cookieFile),
						zap.Error(cleanErr))
				} else {
					s.logger.Info("cleaned up cookie file",
						zap.String("cookie_file", cookieFile))
				}
			}()
		} else {
			s.logger.Info("no cookie available for platform",
				zap.String("platform", platform))
		}
	} else {
		s.logger.Info("cookie feature disabled or not configured")
	}

	// 7.5 获取代理（如果可用）
	var proxyURL string
	var proxyID int64
	if s.assetClient != nil {
		s.logger.Info("attempting to get proxy from Asset Service")
		var err error
		proxyURL, proxyID, err = s.assetClient.GetAvailableProxy()
		if err != nil {
			s.logger.Warn("failed to get proxy, continuing with direct connection",
				zap.Error(err))
		} else if proxyURL != "" {
			s.logger.Info("using proxy for parsing",
				zap.String("proxy_url", proxyURL),
				zap.Int64("proxy_id", proxyID))
		} else {
			s.logger.Info("no proxy available, using direct connection")
		}
	}

	// 8. 调用适配器解析（传递 proxy 和 cookie）
	s.logger.Info("parsing video",
		zap.String("url", url),
		zap.String("platform", platform),
		zap.String("adapter", fmt.Sprintf("%T", adpt)),
		zap.Bool("has_cookie", cookieFile != ""),
		zap.Bool("has_proxy", proxyURL != ""))

	videoInfo, err := adpt.ParseWithProxyAndCookie(url, proxyURL, cookieFile)

	// 报告代理使用结果
	if proxyID != 0 && s.assetClient != nil {
		success := err == nil
		if reportErr := s.assetClient.ReportProxyUsage(proxyID, success); reportErr != nil {
			s.logger.Warn("failed to report proxy usage", zap.Error(reportErr))
		}
	}

	// 报告Cookie使用结果
	if cookieID != 0 && s.assetClient != nil {
		success := err == nil
		s.logger.Info("reporting cookie usage",
			zap.Int64("cookie_id", cookieID),
			zap.Bool("success", success))

		if reportErr := s.assetClient.ReportCookieUsage(cookieID, success); reportErr != nil {
			s.logger.Warn("failed to report cookie usage", zap.Error(reportErr))
		}
	}

	if err != nil {
		s.logger.Error("parse failed",
			zap.String("url", url),
			zap.String("platform", platform),
			zap.Error(err))
		return nil, err
	}

	s.logger.Info("parse completed successfully",
		zap.String("url", url),
		zap.String("video_id", videoInfo.ID),
		zap.String("title", videoInfo.Title),
		zap.Int("format_count", len(videoInfo.Formats)))

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
		CookieID:    cookieID, // 添加 cookie ID
		ProxyURL:    proxyURL, // 添加 proxy URL，确保 downloader 使用相同代理
	}

	// 10. 写入缓存（使用独立的 context 避免超时）
	cacheCtx, cacheCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cacheCancel()
	if err := s.cache.Set(cacheCtx, url, result); err != nil {
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
