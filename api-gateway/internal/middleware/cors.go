package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/config"
)

// CORS 跨域中间件
func CORS(cfg *config.CORSConfig) gin.HandlerFunc {
	// 构建允许的来源映射
	allowedOrigins := make(map[string]bool)
	for _, origin := range cfg.AllowedOrigins {
		allowedOrigins[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		originAllowed := false

		// 检查来源是否被允许
		if len(cfg.AllowedOrigins) > 0 && allowedOrigins[origin] {
			originAllowed = true
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// 设置允许的方法
		if len(cfg.AllowedMethods) > 0 {
			c.Header("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
		} else {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		}

		// 设置允许的头
		if len(cfg.AllowedHeaders) > 0 {
			c.Header("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
		} else {
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID, Accept, Origin")
		}

		// 仅对显式允许的来源开放带凭证的跨域访问。
		if originAllowed {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 暴露响应头给前端（用于文件下载获取文件名等）
		c.Header("Access-Control-Expose-Headers", "Content-Disposition, Content-Length, Content-Type")

		// 设置预检请求缓存时间
		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
		}

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
