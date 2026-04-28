package service

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/zap"

	"youdlp/media-service/internal/adapter"
	"youdlp/media-service/internal/cache"
	"youdlp/media-service/internal/client"
	"youdlp/media-service/internal/detector"
	"youdlp/media-service/internal/platformpolicy"
	"youdlp/media-service/internal/utils"
	"youdlp/media-service/internal/ytdlp"
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
	proxyLease   *client.ProxyLease
	proxyErr     error
	proxyLeases  []*client.ProxyLease
	proxyErrs    []error
	acquireCalls int
	reports      []proxyUsageReport
}

type proxyUsageReport struct {
	taskID       string
	proxyLeaseID string
	stage        string
	success      bool
}

func (f *fakeParserAssetClient) GetAvailableCookie(string) (string, int64, error) {
	return "", 0, nil
}

func (f *fakeParserAssetClient) AcquireProxyForTask(context.Context, string, string) (*client.ProxyLease, error) {
	return f.nextProxyLease()
}

func (f *fakeParserAssetClient) GetAvailableProxy() (*client.ProxyLease, error) {
	return f.nextProxyLease()
}

func (f *fakeParserAssetClient) ReportProxyUsage(taskID, proxyLeaseID, stage string, success bool) error {
	f.reports = append(f.reports, proxyUsageReport{
		taskID:       taskID,
		proxyLeaseID: proxyLeaseID,
		stage:        stage,
		success:      success,
	})
	return nil
}

func (f *fakeParserAssetClient) ReportCookieUsage(int64, bool) error {
	return nil
}

func (f *fakeParserAssetClient) CleanupCookieFile(string) error {
	return nil
}

func (f *fakeParserAssetClient) nextProxyLease() (*client.ProxyLease, error) {
	idx := f.acquireCalls
	f.acquireCalls++

	if idx < len(f.proxyErrs) && f.proxyErrs[idx] != nil {
		return nil, f.proxyErrs[idx]
	}
	if idx < len(f.proxyLeases) {
		return f.proxyLeases[idx], nil
	}
	return f.proxyLease, f.proxyErr
}

type fakeAdapter struct {
	lastProxyURL   string
	lastCookie     string
	proxyURLs      []string
	parseCalls     int
	parseResponse  *ytdlp.VideoInfo
	parseResponses []*ytdlp.VideoInfo
	parseErrors    []error
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
	f.proxyURLs = append(f.proxyURLs, proxyURL)

	idx := f.parseCalls
	f.parseCalls++
	if idx < len(f.parseErrors) && f.parseErrors[idx] != nil {
		return nil, f.parseErrors[idx]
	}
	if idx < len(f.parseResponses) && f.parseResponses[idx] != nil {
		return f.parseResponses[idx], nil
	}
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

func TestParseURLRotatesProxyOnBotDetection(t *testing.T) {
	t.Parallel()

	firstProxy := &client.ProxyLease{URL: "http://proxy-a:8080", LeaseID: "lease-a", ExpireAt: "2026-04-28T10:00:00Z"}
	secondProxy := &client.ProxyLease{URL: "http://proxy-b:8080", LeaseID: "lease-b", ExpireAt: "2026-04-28T10:05:00Z"}
	cacheStub := &fakeParserCache{getErr: utils.ErrCacheMiss}
	assetStub := &fakeParserAssetClient{proxyLeases: []*client.ProxyLease{firstProxy, secondProxy}}
	adapterStub := &fakeAdapter{
		parseErrors: []error{
			fmt.Errorf("%w: Sign in to confirm you’re not a bot. Use --cookies-from-browser or --cookies for the authentication.", utils.ErrYTDLPFailed),
			nil,
		},
		parseResponses: []*ytdlp.VideoInfo{
			nil,
			{
				ID:        "vid-2",
				Title:     "Retried Title",
				Uploader:  "Author",
				Duration:  60,
				Thumbnail: "thumb",
			},
		},
	}

	svc := &ParserService{
		detector: detectorForTests(),
		cache:    cacheStub,
		adapters: map[string]adapter.Adapter{
			"generic": adapterStub,
		},
		limiter:               utils.NewConcurrencyLimiter(1),
		logger:                zap.NewNop(),
		assetClient:           assetStub,
		enableCookies:         false,
		youtubePolicy:         platformpolicy.YouTubePolicy{},
		proxyRetryMaxAttempts: 2,
	}

	result, err := svc.ParseURL(context.Background(), "task-rotate", "https://example.com/video", true)
	if err != nil {
		t.Fatalf("ParseURL returned error: %v", err)
	}

	if assetStub.acquireCalls != 2 {
		t.Fatalf("expected two proxy acquisitions, got %d", assetStub.acquireCalls)
	}
	if len(adapterStub.proxyURLs) != 2 || adapterStub.proxyURLs[0] != firstProxy.URL || adapterStub.proxyURLs[1] != secondProxy.URL {
		t.Fatalf("expected parse attempts with both proxies, got %#v", adapterStub.proxyURLs)
	}
	if result.ProxyURL != secondProxy.URL || result.ProxyLeaseID != secondProxy.LeaseID || result.ProxyExpireAt != secondProxy.ExpireAt {
		t.Fatalf("expected successful result to carry second proxy, got %+v", result)
	}
	if len(assetStub.reports) != 2 {
		t.Fatalf("expected two proxy usage reports, got %#v", assetStub.reports)
	}
	if assetStub.reports[0].proxyLeaseID != firstProxy.LeaseID || assetStub.reports[0].success {
		t.Fatalf("expected first proxy to be reported failed, got %+v", assetStub.reports[0])
	}
	if assetStub.reports[1].proxyLeaseID != secondProxy.LeaseID || !assetStub.reports[1].success {
		t.Fatalf("expected second proxy to be reported successful, got %+v", assetStub.reports[1])
	}
}

func TestParseURLDoesNotRotateProxyForTerminalVideoError(t *testing.T) {
	t.Parallel()

	proxy := &client.ProxyLease{URL: "http://proxy-a:8080", LeaseID: "lease-a"}
	cacheStub := &fakeParserCache{getErr: utils.ErrCacheMiss}
	assetStub := &fakeParserAssetClient{proxyLeases: []*client.ProxyLease{proxy}}
	adapterStub := &fakeAdapter{
		parseErrors: []error{fmt.Errorf("%w: private video", utils.ErrVideoPrivate)},
	}

	svc := &ParserService{
		detector: detectorForTests(),
		cache:    cacheStub,
		adapters: map[string]adapter.Adapter{
			"generic": adapterStub,
		},
		limiter:               utils.NewConcurrencyLimiter(1),
		logger:                zap.NewNop(),
		assetClient:           assetStub,
		enableCookies:         false,
		youtubePolicy:         platformpolicy.YouTubePolicy{},
		proxyRetryMaxAttempts: 2,
	}

	_, err := svc.ParseURL(context.Background(), "task-private", "https://example.com/video", true)
	if err == nil {
		t.Fatalf("expected ParseURL to fail")
	}
	if assetStub.acquireCalls != 1 {
		t.Fatalf("expected one proxy acquisition, got %d", assetStub.acquireCalls)
	}
	if adapterStub.parseCalls != 1 {
		t.Fatalf("expected one parse attempt, got %d", adapterStub.parseCalls)
	}
	if len(assetStub.reports) != 1 || assetStub.reports[0].success {
		t.Fatalf("expected one failed proxy report, got %#v", assetStub.reports)
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
