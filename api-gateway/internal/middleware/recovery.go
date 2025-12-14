package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery Panic 恢复中间件
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录错误日志和堆栈
				log.Printf("[PANIC] %v\n%s", err, debug.Stack())

				// 获取请求 ID
				requestID := ""
				if rid, exists := c.Get("request_id"); exists {
					requestID = rid.(string)
				}

				// 返回 500 错误
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":       500,
					"message":    "internal server error",
					"request_id": requestID,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
