package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"vasset/api-gateway/internal/config"
)

// DownloadTask 下载任务消息
type DownloadTask struct {
	TaskID    string `json:"task_id"`
	UserID    string `json:"user_id"`
	HistoryID int64  `json:"history_id"`
	URL       string `json:"url"`
	Mode      string `json:"mode"`    // quick_download, archive
	Quality   string `json:"quality"` // 1080p, 720p, etc.
	Format    string `json:"format"`  // mp4, webm
	Platform  string `json:"platform"`
	Title     string `json:"title"`
}

// Publisher RabbitMQ 发布器
type Publisher struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	cfg        *config.RabbitMQConfig
	mu         sync.Mutex
	isClosing  bool
	reconnectC chan struct{}
}

// NewPublisher 创建发布器
func NewPublisher(cfg *config.RabbitMQConfig) (*Publisher, error) {
	p := &Publisher{
		cfg:        cfg,
		reconnectC: make(chan struct{}, 1),
	}

	if err := p.connect(); err != nil {
		return nil, err
	}

	// 启动重连监听
	go p.watchConnection()

	return p, nil
}

// connect 连接到 RabbitMQ
func (p *Publisher) connect() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 建立连接
	conn, err := amqp.Dial(p.cfg.URL)
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
		p.cfg.Exchange, // 交换机名称
		"direct",       // 类型
		true,           // 持久化
		false,          // 自动删除
		false,          // 内部
		false,          // 不等待
		nil,            // 参数
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// 声明队列
	_, err = channel.QueueDeclare(
		p.cfg.Queue, // 队列名称
		true,        // 持久化
		false,       // 自动删除
		false,       // 独占
		false,       // 不等待
		nil,         // 参数
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// 绑定队列到交换机
	err = channel.QueueBind(
		p.cfg.Queue,      // 队列名称
		p.cfg.RoutingKey, // 路由键
		p.cfg.Exchange,   // 交换机
		false,            // 不等待
		nil,              // 参数
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	p.conn = conn
	p.channel = channel

	log.Printf("✓ Connected to RabbitMQ: %s", p.cfg.URL)
	return nil
}

// watchConnection 监听连接关闭
func (p *Publisher) watchConnection() {
	for {
		if p.isClosing {
			return
		}

		if p.conn == nil {
			time.Sleep(time.Second)
			continue
		}

		// 监听连接关闭
		closeC := make(chan *amqp.Error)
		p.conn.NotifyClose(closeC)

		err := <-closeC
		if err != nil {
			log.Printf("[MQ] Connection closed: %v, reconnecting...", err)
		}

		if p.isClosing {
			return
		}

		// 重连
		for i := 0; i < 5; i++ {
			if err := p.connect(); err != nil {
				log.Printf("[MQ] Reconnect attempt %d failed: %v", i+1, err)
				time.Sleep(time.Duration(i+1) * time.Second)
				continue
			}
			break
		}
	}
}

// Publish 发布下载任务
func (p *Publisher) Publish(ctx context.Context, task *DownloadTask) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel == nil {
		return fmt.Errorf("channel is not available")
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
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("[MQ] Published task: %s", task.TaskID)
	return nil
}

// Close 关闭连接
func (p *Publisher) Close() error {
	p.isClosing = true

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
