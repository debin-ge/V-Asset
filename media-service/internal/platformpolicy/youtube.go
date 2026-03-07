package platformpolicy

import "strings"

// YouTubePolicy 定义 YouTube 的统一访问策略。
type YouTubePolicy struct {
	DisableCookies *bool    `yaml:"disable_cookies"`
	Args           []string `yaml:"args"`
}

var defaultYouTubeArgs = []string{
	"--impersonate",
	"chrome-131:android",
	"--extractor-args",
	"youtube:player_client=android,tv,web_safari",
	"--sleep-interval",
	"3",
	"--max-sleep-interval",
	"8",
}

// NormalizeYouTubePolicy 补齐默认的 YouTube 策略。
func NormalizeYouTubePolicy(policy YouTubePolicy) YouTubePolicy {
	if policy.DisableCookies == nil {
		disableCookies := true
		policy.DisableCookies = &disableCookies
	}
	if len(policy.Args) == 0 {
		policy.Args = append([]string(nil), defaultYouTubeArgs...)
	} else {
		policy.Args = append([]string(nil), policy.Args...)
	}
	return policy
}

// IsYouTubePlatform 判断平台是否为 YouTube。
func IsYouTubePlatform(platform string) bool {
	return strings.EqualFold(platform, "youtube")
}

// CookiesDisabled 返回 YouTube 是否禁用 cookies。
func (p YouTubePolicy) CookiesDisabled() bool {
	return p.DisableCookies != nil && *p.DisableCookies
}
