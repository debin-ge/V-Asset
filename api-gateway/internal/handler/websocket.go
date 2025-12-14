package handler

import (
	"github.com/gin-gonic/gin"

	"vasset/api-gateway/internal/ws"
)

// WebSocketHandler WebSocket 处理器
type WebSocketHandler struct {
	wsManager *ws.Manager
}

// NewWebSocketHandler 创建 WebSocket 处理器
func NewWebSocketHandler(wsManager *ws.Manager) *WebSocketHandler {
	return &WebSocketHandler{
		wsManager: wsManager,
	}
}

// Progress 处理进度推送 WebSocket 连接
func (h *WebSocketHandler) Progress(c *gin.Context) {
	h.wsManager.HandleConnection(c)
}
