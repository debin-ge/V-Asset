package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 生成请求 ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 获取请求信息
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
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
			userID = uid.(string)
		}

		// 日志输出
		log.Printf("[API] %s | %3d | %13v | %15s | %s | %s?%s | user:%s",
			endTime.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
			query,
			userID,
		)

		// 慢请求告警 (> 1秒)
		if latency > time.Second {
			log.Printf("[SLOW] Request took %v: %s %s", latency, method, path)
		}
	}
}
