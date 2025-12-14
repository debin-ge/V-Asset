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

		// 检查来源是否被允许
		if len(cfg.AllowedOrigins) > 0 {
			if allowedOrigins[origin] {
				c.Header("Access-Control-Allow-Origin", origin)
			}
		} else {
			// 如果没有配置，则允许所有来源
			// 注意：与 credentials 一起使用时，不能用 *，必须返回具体的 origin
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
			}
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

		// 允许携带凭证
		c.Header("Access-Control-Allow-Credentials", "true")

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
