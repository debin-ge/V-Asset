package service

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"vasset/media-service/internal/adapter"
	"vasset/media-service/internal/cache"
	"vasset/media-service/internal/client"
	"vasset/media-service/internal/detector"
	"vasset/media-service/internal/platformpolicy"
	"vasset/media-service/internal/utils"
	"vasset/media-service/internal/ytdlp"
)

type fakeParserCache struct {
	getResult *cache.ParseResult
	getErr    error
	setCalls  int
	setResult *cache.ParseResult
}

func (f *fakeParserCache) Get(context.Context, string) (*cache.ParseResult, error) {
	return f.getResult, f.getErr
}

func (f *fakeParserCache) Set(_ context.Context, _ string, result *cache.ParseResult) error {
	f.setCalls++
	cloned := *result
	f.setResult = &cloned
	return nil
}

type fakeParserAssetClient struct {
	proxyLease *client.ProxyLease
	proxyErr   error
}

func (f *fakeParserAssetClient) GetAvailableCookie(string) (string, int64, error) {
	return "", 0, nil
}

func (f *fakeParserAssetClient) AcquireProxyForTask(context.Context, string, string) (*client.ProxyLease, error) {
	return f.proxyLease, f.proxyErr
}

func (f *fakeParserAssetClient) GetAvailableProxy() (*client.ProxyLease, error) {
	return f.proxyLease, f.proxyErr
}

func (f *fakeParserAssetClient) ReportProxyUsage(string, string, string, bool) error {
	return nil
}

func (f *fakeParserAssetClient) ReportCookieUsage(int64, bool) error {
	return nil
}

func (f *fakeParserAssetClient) CleanupCookieFile(string) error {
	return nil
}

type fakeAdapter struct {
	lastProxyURL  string
	lastCookie    string
	parseResponse *ytdlp.VideoInfo
}

func (f *fakeAdapter) Parse(context.Context, string) (*ytdlp.VideoInfo, error) {
	return f.parseResponse, nil
}

func (f *fakeAdapter) ParseWithCookie(context.Context, string, string) (*ytdlp.VideoInfo, error) {
	return f.parseResponse, nil
}

func (f *fakeAdapter) ParseWithProxyAndCookie(_ context.Context, _ string, proxyURL, cookieFile string) (*ytdlp.VideoInfo, error) {
	f.lastProxyURL = proxyURL
	f.lastCookie = cookieFile
	return f.parseResponse, nil
}

var _ adapter.Adapter = (*fakeAdapter)(nil)

func TestParseURLFallsBackToDirectConnectionWhenProxyUnavailable(t *testing.T) {
	t.Parallel()

	cacheStub := &fakeParserCache{getErr: utils.ErrCacheMiss}
	adapterStub := &fakeAdapter{
		parseResponse: &ytdlp.VideoInfo{
			ID:        "vid-1",
			Title:     "Title",
			Uploader:  "Author",
			Duration:  42,
			Formats:   nil,
			Thumbnail: "thumb",
		},
	}

	svc := &ParserService{
		detector: detectorForTests(),
		cache:    cacheStub,
		adapters: map[string]adapter.Adapter{
			"generic": adapterStub,
		},
		limiter:       utils.NewConcurrencyLimiter(1),
		logger:        zap.NewNop(),
		assetClient:   &fakeParserAssetClient{proxyLease: nil},
		enableCookies: false,
		youtubePolicy: platformpolicy.YouTubePolicy{},
	}

	result, err := svc.ParseURL(context.Background(), "task-1", "https://example.com/video", true)
	if err != nil {
		t.Fatalf("ParseURL returned error: %v", err)
	}

	if adapterStub.lastProxyURL != "" {
		t.Fatalf("expected direct connection, got proxy %q", adapterStub.lastProxyURL)
	}

	if result.ProxyURL != "" || result.ProxyLeaseID != "" || result.ProxyExpireAt != "" {
		t.Fatalf("expected empty proxy metadata, got %+v", result)
	}

	if cacheStub.setCalls != 1 {
		t.Fatalf("expected cache set once, got %d", cacheStub.setCalls)
	}
}

func TestGetParseAccessContextWithoutAssetClientUsesDirectConnection(t *testing.T) {
	t.Parallel()

	svc := &ParserService{
		logger:        zap.NewNop(),
		enableCookies: false,
		youtubePolicy: platformpolicy.YouTubePolicy{},
	}

	accessCtx, err := svc.getParseAccessContext(context.Background(), "task-1", "generic")
	if err != nil {
		t.Fatalf("getParseAccessContext returned error: %v", err)
	}

	if accessCtx.proxyLease != nil {
		t.Fatalf("expected no proxy lease, got %+v", accessCtx.proxyLease)
	}
}

func detectorForTests() *detector.PlatformDetector {
	return detector.NewPlatformDetector()
}

var _ parserCache = (*fakeParserCache)(nil)
var _ parserAssetClient = (*fakeParserAssetClient)(nil)
