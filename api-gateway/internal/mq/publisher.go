package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"youdlp/api-gateway/internal/config"
)

var ErrUnavailable = errors.New("download queue unavailable")

// DownloadTask 下载任务消息
type DownloadTask struct {
	TaskID         string                 `json:"task_id"`
	UserID         string                 `json:"user_id"`
	HistoryID      int64                  `json:"history_id"`
	URL            string                 `json:"url"`
	Mode           string                 `json:"mode"`    // quick_download, archive
	Quality        string                 `json:"quality"` // 1080p, 720p, 160kbps, etc.
	Format         string                 `json:"format"`  // mp4, webm, m4a
	FormatID       string                 `json:"format_id"`
	SelectedFormat *SelectedFormatMessage `json:"selected_format,omitempty"`
	Platform       string                 `json:"platform"`
	Title          string                 `json:"title"`
	CookieID       int64                  `json:"cookie_id"`       // parser 使用的 cookie ID
	ProxyURL       string                 `json:"proxy_url"`       // parser 使用的 proxy URL
	ProxyLeaseID   string                 `json:"proxy_lease_id"`  // parser 使用的动态代理租约 ID
	ProxyExpireAt  string                 `json:"proxy_expire_at"` // parser 获取到的代理过期时间
}

// SelectedFormatMessage MQ 内透传的精确格式信息
type SelectedFormatMessage struct {
	FormatID   string  `json:"format_id"`
	Quality    string  `json:"quality,omitempty"`
	Extension  string  `json:"extension,omitempty"`
	Filesize   int64   `json:"filesize,omitempty"`
	Height     int32   `json:"height,omitempty"`
	Width      int32   `json:"width,omitempty"`
	FPS        float64 `json:"fps,omitempty"`
	VideoCodec string  `json:"video_codec,omitempty"`
	AudioCodec string  `json:"audio_codec,omitempty"`
	VBR        float64 `json:"vbr,omitempty"`
	ABR        float64 `json:"abr,omitempty"`
	ASR        int32   `json:"asr,omitempty"`
}

// Publisher RabbitMQ 发布器
type Publisher struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	cfg       *config.RabbitMQConfig
	mu        sync.Mutex
	isClosing bool
}

// NewPublisher 创建发布器
func NewPublisher(cfg *config.RabbitMQConfig) (*Publisher, error) {
	p := &Publisher{cfg: cfg}

	err := p.connect()

	// 启动重连监听
	go p.watchConnection()

	return p, err
}

// connect 连接到 RabbitMQ
func (p *Publisher) connect() error {
	p.mu.Lock()
	if p.isClosing {
		p.mu.Unlock()
		return ErrUnavailable
	}
	cfg := *p.cfg
	p.mu.Unlock()

	// 建立连接
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// 创建通道
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// 声明交换机
	err = channel.ExchangeDeclare(
		cfg.Exchange, // 交换机名称
		"direct",     // 类型
		true,         // 持久化
		false,        // 自动删除
		false,        // 内部
		false,        // 不等待
		nil,          // 参数
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// 声明队列
	_, err = channel.QueueDeclare(
		cfg.Queue, // 队列名称
		true,      // 持久化
		false,     // 自动删除
		false,     // 独占
		false,     // 不等待
		nil,       // 参数
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// 绑定队列到交换机
	err = channel.QueueBind(
		cfg.Queue,      // 队列名称
		cfg.RoutingKey, // 路由键
		cfg.Exchange,   // 交换机
		false,          // 不等待
		nil,            // 参数
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isClosing {
		channel.Close()
		conn.Close()
		return ErrUnavailable
	}
	p.clearConnectionLocked(nil)
	p.conn = conn
	p.channel = channel

	log.Printf("✓ Connected to RabbitMQ: %s", cfg.URL)
	return nil
}

// watchConnection 监听连接关闭
func (p *Publisher) watchConnection() {
	for {
		conn := p.connection()
		if conn == nil || conn.IsClosed() {
			if p.closing() {
				return
			}
			if err := p.connect(); err != nil {
				log.Printf("[MQ] Reconnect failed: %v", err)
				if !p.waitBeforeReconnect(5 * time.Second) {
					return
				}
				continue
			}
			continue
		}

		// 监听连接关闭
		closeC := make(chan *amqp.Error, 1)
		conn.NotifyClose(closeC)

		err, ok := <-closeC
		if p.closing() {
			return
		}
		if ok && err != nil {
			log.Printf("[MQ] Connection closed: %v, reconnecting...", err)
		} else {
			log.Printf("[MQ] Connection closed, reconnecting...")
		}

		p.clearConnection(conn)
	}
}

// Publish 发布下载任务
func (p *Publisher) Publish(ctx context.Context, task *DownloadTask) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel == nil || p.channel.IsClosed() {
		return fmt.Errorf("%w: channel is not available", ErrUnavailable)
	}

	// 序列化任务
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	// 发布消息
	err = p.channel.PublishWithContext(ctx,
		p.cfg.Exchange,   // 交换机
		p.cfg.RoutingKey, // 路由键
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // 持久化
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("failed to publish message: %w", err)
		}
		p.clearConnectionLocked(p.conn)
		return fmt.Errorf("%w: failed to publish message: %w", ErrUnavailable, err)
	}

	log.Printf("[MQ] Published task: %s", task.TaskID)
	return nil
}

func (p *Publisher) IsReady() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return !p.isClosing &&
		p.conn != nil &&
		!p.conn.IsClosed() &&
		p.channel != nil &&
		!p.channel.IsClosed()
}

// Close 关闭连接
func (p *Publisher) Close() error {
	p.mu.Lock()
	p.isClosing = true
	channel := p.channel
	conn := p.conn
	p.channel = nil
	p.conn = nil
	p.mu.Unlock()

	if channel != nil {
		channel.Close()
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (p *Publisher) connection() *amqp.Connection {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.conn
}

func (p *Publisher) closing() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.isClosing
}

func (p *Publisher) waitBeforeReconnect(delay time.Duration) bool {
	time.Sleep(delay)
	return !p.closing()
}

func (p *Publisher) clearConnection(conn *amqp.Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.clearConnectionLocked(conn)
}

func (p *Publisher) clearConnectionLocked(conn *amqp.Connection) {
	if conn != nil && p.conn != conn {
		return
	}

	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	p.channel = nil
	p.conn = nil
}
