package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应该限制
	},
}

// ProgressMessage 进度消息
type ProgressMessage struct {
	TaskID          string  `json:"task_id"`
	Status          string  `json:"status"` // pending, downloading, merging, completed, failed
	Percent         float64 `json:"percent"`
	DownloadedBytes int64   `json:"downloaded_bytes,omitempty"`
	TotalBytes      int64   `json:"total_bytes,omitempty"`
	Speed           string  `json:"speed,omitempty"`
	ETA             string  `json:"eta,omitempty"`
	Message         string  `json:"message,omitempty"`
	HistoryID       int64   `json:"history_id,omitempty"`
	FileSize        int64   `json:"file_size,omitempty"`
	ErrorCode       int     `json:"error_code,omitempty"`
}

// Manager WebSocket 连接管理器
type Manager struct {
	connections sync.Map // map[taskID]*websocket.Conn
	rdb         *redis.Client
}

// NewManager 创建 WebSocket 管理器
func NewManager(rdb *redis.Client) *Manager {
	return &Manager{
		rdb: rdb,
	}
}

// HandleConnection 处理 WebSocket 连接
func (m *Manager) HandleConnection(c *gin.Context) {
	// 获取任务 ID
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "task_id is required",
		})
		return
	}

	// 验证 Token (从查询参数或头获取)
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
	}
	// 注意: 实际场景中需要验证 token，这里简化处理

	// 升级到 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection: %v", err)
		return
	}
	defer func() {
		conn.Close()
		m.connections.Delete(taskID)
		log.Printf("[WS] Connection closed: %s", taskID)
	}()

	// 保存连接
	m.connections.Store(taskID, conn)
	log.Printf("[WS] Connection established: %s", taskID)

	// 设置 Pong 处理
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动心跳
	go m.heartbeat(taskID, conn)

	// 订阅 Redis 频道
	ctx := context.Background()
	channelName := fmt.Sprintf("progress:%s", taskID)
	pubsub := m.rdb.Subscribe(ctx, channelName)
	defer pubsub.Close()

	// 监听 Redis 消息
	ch := pubsub.Channel()
	for msg := range ch {
		// 解析消息
		var progress ProgressMessage
		if err := json.Unmarshal([]byte(msg.Payload), &progress); err != nil {
			log.Printf("[WS] Failed to parse message: %v", err)
			continue
		}

		// 发送给客户端
		if err := conn.WriteJSON(progress); err != nil {
			log.Printf("[WS] Failed to send message: %v", err)
			return
		}

		// 如果任务完成或失败，关闭连接
		if progress.Status == "completed" || progress.Status == "failed" {
			return
		}
	}
}

// heartbeat 发送心跳
func (m *Manager) heartbeat(taskID string, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 检查连接是否还存在
		if _, ok := m.connections.Load(taskID); !ok {
			return
		}

		// 发送 Ping
		if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
			log.Printf("[WS] Failed to send ping: %v", err)
			return
		}
	}
}

// Broadcast 向指定任务广播消息
func (m *Manager) Broadcast(taskID string, message *ProgressMessage) error {
	if conn, ok := m.connections.Load(taskID); ok {
		return conn.(*websocket.Conn).WriteJSON(message)
	}
	return nil
}

// GetConnectionCount 获取当前连接数
func (m *Manager) GetConnectionCount() int {
	count := 0
	m.connections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
