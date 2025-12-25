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
	// 获取 Token (从查询参数或头获取)
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
	}
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "token is required",
		})
		return
	}

	// 可选: 获取特定任务 ID (如果为空，则监听用户所有任务)
	taskID := c.Query("task_id")

	// 升级到 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection: %v", err)
		return
	}

	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	defer func() {
		conn.Close()
		m.connections.Delete(connID)
		log.Printf("[WS] Connection closed: %s", connID)
	}()

	// 保存连接
	m.connections.Store(connID, conn)
	log.Printf("[WS] Connection established: %s (taskID: %s)", connID, taskID)

	// 设置 Pong 处理
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动心跳
	go m.heartbeat(connID, conn)

	ctx := context.Background()

	if taskID != "" {
		// 模式1: 监听特定任务
		channelName := fmt.Sprintf("progress:%s", taskID)
		pubsub := m.rdb.Subscribe(ctx, channelName)
		defer pubsub.Close()

		log.Printf("[WS] Subscribed to channel: %s", channelName)

		ch := pubsub.Channel()
		for msg := range ch {
			var progress ProgressMessage
			if err := json.Unmarshal([]byte(msg.Payload), &progress); err != nil {
				log.Printf("[WS] Failed to parse message: %v", err)
				continue
			}

			if err := conn.WriteJSON(progress); err != nil {
				log.Printf("[WS] Failed to send message: %v", err)
				return
			}

			if progress.Status == "completed" || progress.Status == "failed" {
				return
			}
		}
	} else {
		// 模式2: 使用 pattern 订阅所有进度消息
		pubsub := m.rdb.PSubscribe(ctx, "progress:*")
		defer pubsub.Close()

		log.Printf("[WS] Subscribed to pattern: progress:*")

		ch := pubsub.Channel()
		for msg := range ch {
			var progress ProgressMessage
			if err := json.Unmarshal([]byte(msg.Payload), &progress); err != nil {
				log.Printf("[WS] Failed to parse message: %v", err)
				continue
			}

			if err := conn.WriteJSON(progress); err != nil {
				log.Printf("[WS] Failed to send message: %v", err)
				return
			}
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
