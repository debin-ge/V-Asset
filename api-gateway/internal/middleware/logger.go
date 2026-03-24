package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"youdlp/api-gateway/internal/trace"
)

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 生成请求 ID
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Request = c.Request.WithContext(trace.WithRequestID(c.Request.Context(), requestID))

		// 获取请求信息
		path := c.Request.URL.Path
		query := sanitizeQuery(c.Request.URL.Query())
		method := c.Request.Method
		clientIP := c.ClientIP()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// 获取响应状态码
		statusCode := c.Writer.Status()

		// 获取用户 ID (如果已认证)
		userID := ""
		if uid, exists := c.Get("user_id"); exists {
			if typed, ok := uid.(string); ok {
				userID = typed
			} else {
				userID = fmt.Sprint(uid)
			}
		}

		payload, err := json.Marshal(map[string]any{
			"service":    "api-gateway",
			"layer":      "http",
			"ts":         endTime.Format(time.RFC3339Nano),
			"request_id": requestID,
			"method":     method,
			"path":       path,
			"query":      query,
			"status":     statusCode,
			"latency_ms": latency.Milliseconds(),
			"client_ip":  clientIP,
			"user_id":    userID,
		})
		if err != nil {
			log.Printf(`{"service":"api-gateway","layer":"http","request_id":"%s","message":"failed to marshal http access log","error":"%v"}`, requestID, err)
		} else {
			log.Printf("%s", payload)
		}

		// 慢请求告警 (> 1秒)
		if latency > time.Second {
			slowPayload, err := json.Marshal(map[string]any{
				"service":    "api-gateway",
				"layer":      "http",
				"request_id": requestID,
				"message":    "slow request",
				"method":     method,
				"path":       path,
				"latency_ms": latency.Milliseconds(),
			})
			if err != nil {
				log.Printf(`{"service":"api-gateway","layer":"http","request_id":"%s","message":"slow request","latency_ms":%d}`, requestID, latency.Milliseconds())
			} else {
				log.Printf("%s", slowPayload)
			}
		}
	}
}

func sanitizeQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}

	sanitized := make(url.Values, len(values))
	for key, rawValues := range values {
		if isSensitiveQueryKey(key) {
			sanitized[key] = []string{"REDACTED"}
			continue
		}
		copied := make([]string, len(rawValues))
		copy(copied, rawValues)
		sanitized[key] = copied
	}
	return sanitized.Encode()
}

func isSensitiveQueryKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	switch normalized {
	case "token", "access_token", "refresh_token", "password", "cookie", "secret", "session", "authorization":
		return true
	default:
		return strings.Contains(normalized, "token") ||
			strings.Contains(normalized, "password") ||
			strings.Contains(normalized, "authorization") ||
			strings.Contains(normalized, "cookie")
	}
}
