package models

const (
	ErrorCategoryNetworkTimeout   = "network_timeout"
	ErrorCategoryProxyAuth        = "proxy_auth"
	ErrorCategoryProxyUnreachable = "proxy_unreachable"
	ErrorCategoryRateLimited      = "rate_limited"
	ErrorCategoryBotDetected      = "bot_detected"
	ErrorCategoryCookieInvalid    = "cookie_invalid"
	ErrorCategoryTerminalVideo    = "terminal_video"
	ErrorCategoryUnknown          = "unknown"
)

const (
	ProxyRiskExcludeThreshold = 80
	ProxyRiskMaxScore         = 100
)
