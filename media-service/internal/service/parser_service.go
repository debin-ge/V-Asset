package service

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"youdlp/media-service/internal/adapter"
	"youdlp/media-service/internal/cache"
	"youdlp/media-service/internal/client"
	"youdlp/media-service/internal/config"
	"youdlp/media-service/internal/detector"
	"youdlp/media-service/internal/platformpolicy"
	"youdlp/media-service/internal/redact"
	"youdlp/media-service/internal/utils"
	"youdlp/media-service/internal/ytdlp"
)

// ParserService 解析服务
type ParserService struct {
	detector              *detector.PlatformDetector
	cache                 parserCache
	adapters              map[string]adapter.Adapter
	limiter               *utils.ConcurrencyLimiter
	logger                *zap.Logger
	assetClient           parserAssetClient
	enableCookies         bool
	youtubePolicy         platformpolicy.YouTubePolicy
	proxyRetryMaxAttempts int
}

type parserCache interface {
	Get(ctx context.Context, url string) (*cache.ParseResult, error)
	Set(ctx context.Context, url string, result *cache.ParseResult) error
}

type parserAssetClient interface {
	GetAvailableCookie(platform string) (string, int64, error)
	AcquireProxyForTask(ctx context.Context, taskID, platform string) (*client.ProxyLease, error)
	GetAvailableProxy() (*client.ProxyLease, error)
	ReportProxyUsage(taskID, proxyLeaseID, stage string, success bool) error
	ReportCookieUsage(cookieID int64, success bool) error
	CleanupCookieFile(cookieFile string) error
}

type parseAccessContext struct {
	cookieFile string
	cookieID   int64
	proxyLease *client.ProxyLease
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
		adapters["youtube"] = adapter.NewYouTubeAdapter(ytdlpWrapper, "", cfg.YTDLP.YouTube.Args)
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
		detector:              detector.NewPlatformDetector(),
		cache:                 cacheService,
		adapters:              adapters,
		limiter:               utils.NewConcurrencyLimiter(cfg.YTDLP.MaxConcurrent),
		logger:                logger,
		assetClient:           assetClient,
		enableCookies:         cfg.AssetService.EnableCookies && assetClient != nil,
		youtubePolicy:         cfg.YTDLP.YouTube,
		proxyRetryMaxAttempts: cfg.AssetService.ProxyRetryMaxAttempts,
	}
}

// ParseURL 解析视频URL
func (s *ParserService) ParseURL(ctx context.Context, taskID, url string, skipCache bool) (*cache.ParseResult, error) {
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
			if err := s.attachDynamicAccess(cached, taskID); err != nil {
				return nil, err
			}
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

	videoInfo, accessCtx, err := s.parseWithProxyRetry(ctx, taskID, url, platform, adpt)
	if err != nil {
		s.logger.Error("parse failed",
			zap.String("url", url),
			zap.String("platform", platform),
			zap.Error(err))
		return nil, err
	}
	if accessCtx != nil && accessCtx.cookieFile != "" {
		defer s.cleanupCookieFile(accessCtx.cookieFile)
	}

	proxyURL := ""
	proxyLeaseID := ""
	proxyExpireAt := ""
	if accessCtx != nil && accessCtx.proxyLease != nil {
		proxyURL = accessCtx.proxyLease.URL
		proxyLeaseID = accessCtx.proxyLease.LeaseID
		proxyExpireAt = accessCtx.proxyLease.ExpireAt
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
		VideoID:       videoInfo.ID,
		Platform:      platform,
		Title:         utils.SanitizeString(videoInfo.Title),
		Description:   utils.SanitizeString(videoInfo.Description),
		Duration:      videoInfo.Duration,
		Thumbnail:     videoInfo.Thumbnail,
		Author:        utils.SanitizeString(videoInfo.Uploader),
		UploadDate:    videoInfo.UploadDate,
		ViewCount:     videoInfo.ViewCount,
		Formats:       formats,
		CookieID:      accessCtx.cookieID,
		ProxyURL:      proxyURL,
		ProxyLeaseID:  proxyLeaseID,
		ProxyExpireAt: proxyExpireAt,
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

func (s *ParserService) parseWithProxyRetry(ctx context.Context, taskID, url, platform string, adpt adapter.Adapter) (*ytdlp.VideoInfo, *parseAccessContext, error) {
	maxAttempts := s.parseProxyMaxAttempts()
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		accessCtx, err := s.getParseAccessContext(ctx, taskID, platform)
		if err != nil {
			s.logger.Error("failed to get proxy lease for parsing",
				zap.String("url", url),
				zap.String("platform", platform),
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", maxAttempts),
				zap.Error(err))
			if lastErr != nil {
				return nil, nil, lastErr
			}
			return nil, nil, err
		}

		s.logger.Info("parsing video",
			zap.String("url", url),
			zap.String("platform", platform),
			zap.String("adapter", fmt.Sprintf("%T", adpt)),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
			zap.Bool("youtube_cookie_disabled", platformpolicy.IsYouTubePlatform(platform) && s.youtubePolicy.CookiesDisabled()),
			zap.Bool("has_cookie", accessCtx.cookieFile != ""),
			zap.Bool("has_proxy", accessCtx.proxyLease != nil && accessCtx.proxyLease.URL != ""))

		proxyURL := ""
		if accessCtx.proxyLease != nil {
			proxyURL = accessCtx.proxyLease.URL
		}

		videoInfo, err := adpt.ParseWithProxyAndCookie(ctx, url, proxyURL, accessCtx.cookieFile)
		usageReported := s.reportParseAccessUsage(taskID, accessCtx, err == nil)
		if err == nil {
			return videoInfo, accessCtx, nil
		}

		lastErr = err
		if !s.shouldRetryParseWithNewProxy(attempt, maxAttempts, accessCtx, err, usageReported) {
			s.cleanupCookieFile(accessCtx.cookieFile)
			return nil, nil, err
		}

		s.logger.Warn("parse failed with retryable proxy or bot-detection error, rotating proxy",
			zap.String("url", url),
			zap.String("platform", platform),
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
			zap.Error(err))
		s.cleanupCookieFile(accessCtx.cookieFile)
	}

	return nil, nil, lastErr
}

func (s *ParserService) reportParseAccessUsage(taskID string, accessCtx *parseAccessContext, success bool) bool {
	if accessCtx == nil || s.assetClient == nil {
		return true
	}

	usageReported := true
	if accessCtx.proxyLease != nil {
		if reportErr := s.assetClient.ReportProxyUsage(taskID, accessCtx.proxyLease.LeaseID, "parse", success); reportErr != nil {
			s.logger.Warn("failed to report proxy usage", zap.Error(reportErr))
			usageReported = false
		}
	}

	if accessCtx.cookieID != 0 {
		s.logger.Info("reporting cookie usage",
			zap.Int64("cookie_id", accessCtx.cookieID),
			zap.Bool("success", success))

		if reportErr := s.assetClient.ReportCookieUsage(accessCtx.cookieID, success); reportErr != nil {
			s.logger.Warn("failed to report cookie usage", zap.Error(reportErr))
		}
	}

	return usageReported
}

func (s *ParserService) shouldRetryParseWithNewProxy(attempt, maxAttempts int, accessCtx *parseAccessContext, err error, usageReported bool) bool {
	if !usageReported || attempt >= maxAttempts || accessCtx == nil || accessCtx.proxyLease == nil || accessCtx.proxyLease.URL == "" {
		return false
	}
	return utils.IsProxyOrBotRetryableError(err)
}

func (s *ParserService) parseProxyMaxAttempts() int {
	if s.proxyRetryMaxAttempts <= 0 {
		return 1
	}
	return s.proxyRetryMaxAttempts
}

func (s *ParserService) getParseAccessContext(ctx context.Context, taskID, platform string) (*parseAccessContext, error) {
	accessCtx := &parseAccessContext{}

	if platformpolicy.IsYouTubePlatform(platform) && s.youtubePolicy.CookiesDisabled() {
		s.logger.Info("youtube access policy applied",
			zap.String("platform", platform),
			zap.Bool("youtube_cookie_disabled", true))
	} else if s.enableCookies && s.assetClient != nil {
		s.logger.Info("attempting to get cookie", zap.String("platform", platform))

		cookieFile, cookieID, err := s.assetClient.GetAvailableCookie(platform)
		if err != nil {
			s.logger.Warn("failed to get cookie, continuing without cookie",
				zap.String("platform", platform),
				zap.Error(err))
		} else {
			accessCtx.cookieFile = cookieFile
			accessCtx.cookieID = cookieID
		}
	} else {
		s.logger.Info("cookie feature disabled or not configured")
	}

	if s.assetClient == nil {
		s.logger.Info("asset service not configured, using direct connection for parsing")
		return accessCtx, nil
	}

	s.logger.Info("attempting to get proxy lease from Asset Service", zap.String("task_id", taskID))
	var proxyLease *client.ProxyLease
	var err error
	if taskID != "" {
		proxyLease, err = s.assetClient.AcquireProxyForTask(ctx, taskID, platform)
	} else {
		proxyLease, err = s.assetClient.GetAvailableProxy()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy lease: %w", err)
	}
	if proxyLease == nil || proxyLease.URL == "" {
		s.logger.Info("no proxy lease available, using direct connection", zap.String("task_id", taskID))
		return accessCtx, nil
	}

	s.logger.Info("using proxy for parsing",
		zap.String("proxy_url", redact.ProxyURL(proxyLease.URL)),
		zap.String("proxy_lease_id", proxyLease.LeaseID),
		zap.String("proxy_expire_at", proxyLease.ExpireAt))

	accessCtx.proxyLease = proxyLease
	return accessCtx, nil
}

func (s *ParserService) attachDynamicAccess(result *cache.ParseResult, taskID string) error {
	accessCtx, err := s.getParseAccessContext(context.Background(), taskID, result.Platform)
	if err != nil {
		s.logger.Warn("failed to refresh dynamic proxy context for cached result", zap.Error(err))
		return err
	}

	result.CookieID = accessCtx.cookieID
	if accessCtx.proxyLease != nil {
		result.ProxyURL = accessCtx.proxyLease.URL
		result.ProxyLeaseID = accessCtx.proxyLease.LeaseID
		result.ProxyExpireAt = accessCtx.proxyLease.ExpireAt
	} else {
		result.ProxyURL = ""
		result.ProxyLeaseID = ""
		result.ProxyExpireAt = ""
	}
	s.cleanupCookieFile(accessCtx.cookieFile)
	return nil
}

func (s *ParserService) cleanupCookieFile(cookieFile string) {
	if cookieFile == "" || s.assetClient == nil {
		return
	}
	if cleanErr := s.assetClient.CleanupCookieFile(cookieFile); cleanErr != nil {
		s.logger.Warn("failed to cleanup cookie file",
			zap.String("cookie_file", cookieFile),
			zap.Error(cleanErr))
		return
	}
	s.logger.Info("cleaned up cookie file", zap.String("cookie_file", cookieFile))
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
