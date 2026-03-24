package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"vasset/api-gateway/internal/middleware"
	pb "vasset/api-gateway/proto"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			return true
		}

		parsedOrigin, err := url.Parse(origin)
		if err != nil {
			return false
		}

		scheme := requestScheme(r)
		requestHost := requestHostName(r)
		originHost := parsedOrigin.Hostname()
		hostMatches := originHost != "" && strings.EqualFold(originHost, requestHost)
		if !hostMatches && isInternalProxyHost(requestHost) {
			hostMatches = originHost != ""
		}

		allowed := originHost != "" && hostMatches
		if !allowed {
			log.Printf("[WS] Rejecting websocket origin: remote=%s host=%s origin=%s expected_scheme=%s expected_host=%s", r.RemoteAddr, r.Host, origin, scheme, requestHost)
		}
		return allowed
	},
}

const (
	wsReadTimeout  = 60 * time.Second
	wsPingInterval = 30 * time.Second
	wsWriteTimeout = 10 * time.Second
)

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
	authClient  authVerifier
	assetClient taskAccessChecker
}

type authVerifier interface {
	VerifyToken(ctx context.Context, in *pb.VerifyTokenRequest, opts ...grpc.CallOption) (*pb.VerifyTokenResponse, error)
}

type taskAccessChecker interface {
	GetHistoryByTask(ctx context.Context, in *pb.GetHistoryByTaskRequest, opts ...grpc.CallOption) (*pb.GetHistoryByTaskResponse, error)
}

// NewManager 创建 WebSocket 管理器
func NewManager(rdb *redis.Client, authClient authVerifier, assetClient taskAccessChecker) *Manager {
	return &Manager{
		rdb:         rdb,
		authClient:  authClient,
		assetClient: assetClient,
	}
}

// HandleConnection 处理 WebSocket 连接
func (m *Manager) HandleConnection(c *gin.Context) {
	token, selectedProtocol := extractWebSocketToken(c.Request)

	claims, err := middleware.AuthenticateToken(c.Request.Context(), m.authClient, m.rdb, token)
	if err != nil {
		log.Printf("[WS] Rejecting connection from %s: auth failed (%v)", c.ClientIP(), err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": err.Error(),
		})
		return
	}

	taskID := strings.TrimSpace(c.Query("task_id"))
	if taskID == "" {
		log.Printf("[WS] Rejecting connection from %s for user %s: missing task_id", c.ClientIP(), claims.UserID)
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "task_id is required",
		})
		return
	}

	if err := m.validateTaskAccess(c.Request.Context(), claims.UserID, taskID); err != nil {
		log.Printf("[WS] Rejecting connection from %s for user %s task %s: access denied (%v)", c.ClientIP(), claims.UserID, taskID, err)
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": err.Error(),
		})
		return
	}

	// 升级到 WebSocket
	var responseHeader http.Header
	if selectedProtocol != "" {
		responseHeader = http.Header{
			"Sec-WebSocket-Protocol": []string{selectedProtocol},
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection from %s for user %s task %s: %v", c.ClientIP(), claims.UserID, taskID, err)
		return
	}

	connID := fmt.Sprintf("conn_%d", time.Now().UnixNano())
	done := make(chan struct{})
	var stopOnce sync.Once
	stop := func() {
		stopOnce.Do(func() {
			close(done)
			conn.Close()
			m.connections.Delete(connID)
			log.Printf("[WS] Connection closed: %s", connID)
		})
	}
	defer stop()

	// 保存连接
	m.connections.Store(connID, conn)
	log.Printf("[WS] Connection established: %s (taskID: %s userID: %s)", connID, taskID, claims.UserID)

	// 设置 Pong 处理
	conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
		return nil
	})

	// 启动心跳
	go m.heartbeat(connID, conn, done, stop)
	go m.readPump(connID, conn, stop)

	ctx := c.Request.Context()
	channelName := fmt.Sprintf("progress:%s", taskID)
	pubsub := m.rdb.Subscribe(ctx, channelName)
	defer pubsub.Close()

	log.Printf("[WS] Subscribed to channel: %s", channelName)

	ch := pubsub.Channel()
	for {
		select {
		case <-done:
			return
		case msg, ok := <-ch:
			if !ok {
				stop()
				return
			}

			var progress ProgressMessage
			if err := json.Unmarshal([]byte(msg.Payload), &progress); err != nil {
				log.Printf("[WS] Failed to parse message: %v", err)
				continue
			}

			conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := conn.WriteJSON(progress); err != nil {
				log.Printf("[WS] Failed to send message: %v", err)
				stop()
				return
			}
			log.Printf("[WS] Forwarded progress to %s: task=%s status=%s percent=%.2f", connID, progress.TaskID, progress.Status, progress.Percent)

			if progress.Status == "completed" || progress.Status == "failed" {
				stop()
				return
			}
		}
	}
}

func extractWebSocketToken(r *http.Request) (string, string) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader != "" {
		return authHeader, ""
	}

	subprotocols := websocket.Subprotocols(r)
	if len(subprotocols) >= 2 && strings.EqualFold(subprotocols[0], "bearer") {
		return strings.TrimSpace(subprotocols[1]), subprotocols[0]
	}

	return "", ""
}

func (m *Manager) validateTaskAccess(ctx context.Context, userID, taskID string) error {
	if m.assetClient == nil {
		return fmt.Errorf("task validation service unavailable")
	}

	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := m.assetClient.GetHistoryByTask(checkCtx, &pb.GetHistoryByTaskRequest{
		TaskId: taskID,
		UserId: userID,
	})
	if err == nil {
		return nil
	}

	if grpcstatus.Code(err) == codes.NotFound {
		return fmt.Errorf("task not found or access denied")
	}

	log.Printf("[WS] Failed to validate task ownership: user=%s task=%s err=%v", userID, taskID, err)
	return fmt.Errorf("failed to validate task access")
}

func requestScheme(r *http.Request) string {
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		if idx := strings.Index(forwardedProto, ","); idx >= 0 {
			forwardedProto = forwardedProto[:idx]
		}
		return strings.TrimSpace(strings.ToLower(forwardedProto))
	}

	if cfVisitor := strings.TrimSpace(r.Header.Get("CF-Visitor")); cfVisitor != "" {
		if strings.Contains(strings.ToLower(cfVisitor), `"scheme":"https"`) {
			return "https"
		}
		if strings.Contains(strings.ToLower(cfVisitor), `"scheme":"http"`) {
			return "http"
		}
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}

func requestHostName(r *http.Request) string {
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	} else if idx := strings.Index(host, ","); idx >= 0 {
		host = host[:idx]
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return parsedHost
	}

	return host
}

func isInternalProxyHost(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return false
	}

	if host == "localhost" || host == "api-gateway" {
		return true
	}

	if strings.Contains(host, ".") {
		if ip := net.ParseIP(host); ip != nil {
			return ip.IsLoopback() || ip.IsPrivate()
		}
		return false
	}

	return true
}

// heartbeat 发送心跳
func (m *Manager) heartbeat(taskID string, conn *websocket.Conn, done <-chan struct{}, stop func()) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// 检查连接是否还存在
			if _, ok := m.connections.Load(taskID); !ok {
				return
			}

			// 发送 Ping
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(wsWriteTimeout)); err != nil {
				log.Printf("[WS] Failed to send ping: %v", err)
				stop()
				return
			}
		}
	}
}

func (m *Manager) readPump(taskID string, conn *websocket.Conn, stop func()) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("[WS] Read loop ended for %s: %v", taskID, err)
			}
			stop()
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
