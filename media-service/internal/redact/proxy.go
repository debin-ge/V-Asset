package redact

import (
	"net/url"
	"strings"
)

// ProxyURL 对代理 URL 中的凭据做脱敏处理。
func ProxyURL(raw string) string {
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if parsed.User != nil {
		username := parsed.User.Username()
		if username != "" {
			parsed.User = url.UserPassword(username, "***")
		}
	}

	return parsed.String()
}

// ProxyArgs 返回一份代理参数已脱敏的命令参数副本。
func ProxyArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	sanitized := append([]string(nil), args...)
	for i := 0; i < len(sanitized); i++ {
		if sanitized[i] == "--proxy" && i+1 < len(sanitized) {
			sanitized[i+1] = ProxyURL(sanitized[i+1])
			i++
			continue
		}

		if strings.HasPrefix(sanitized[i], "--proxy=") {
			sanitized[i] = "--proxy=" + ProxyURL(strings.TrimPrefix(sanitized[i], "--proxy="))
		}
	}

	return sanitized
}
