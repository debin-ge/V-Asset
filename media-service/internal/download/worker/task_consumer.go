package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"youdlp/media-service/internal/download/config"
	"youdlp/media-service/internal/download/models"
)

// TaskConsumer MQ 任务消费者
type TaskConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	pool    *Pool
}

// NewTaskConsumer 创建任务消费者
func NewTaskConsumer(cfg *config.RabbitMQConfig, pool *Pool) (*TaskConsumer, error) {
	log.Printf("[TaskConsumer] Connecting to RabbitMQ: %s", cfg.URL)
	// 连接 RabbitMQ
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		log.Printf("[TaskConsumer] Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}
	log.Println("[TaskConsumer] ✓ Connected to RabbitMQ")

	// 创建通道
	log.Println("[TaskConsumer] Creating channel...")
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("[TaskConsumer] Failed to create channel: %v", err)
		conn.Close()
		return nil, err
	}
	log.Println("[TaskConsumer] ✓ Channel created")

	// 声明队列
	log.Printf("[TaskConsumer] Declaring queue: %s", cfg.Queue)
	_, err = ch.QueueDeclare(
		cfg.Queue, // 队列名
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Printf("[TaskConsumer] Failed to declare queue: %v", err)
		ch.Close()
		conn.Close()
		return nil, err
	}
	log.Printf("[TaskConsumer] ✓ Queue declared: %s", cfg.Queue)

	// 设置预取数
	log.Printf("[TaskConsumer] Setting QoS prefetch count: %d", cfg.PrefetchCount)
	if err := ch.Qos(cfg.PrefetchCount, 0, false); err != nil {
		log.Printf("[TaskConsumer] Failed to set QoS: %v", err)
		ch.Close()
		conn.Close()
		return nil, err
	}
	log.Printf("[TaskConsumer] ✓ QoS configured with prefetch: %d", cfg.PrefetchCount)

	return &TaskConsumer{
		conn:    conn,
		channel: ch,
		queue:   cfg.Queue,
		pool:    pool,
	}, nil
}

// Start 启动消费
func (c *TaskConsumer) Start(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		c.queue, // 队列名
		"",      // 消费者名
		false,   // 手动 ACK
		false,   // 非独占
		false,   // no-local
		false,   // no-wait
		nil,     // args
	)
	if err != nil {
		return err
	}

	log.Printf("[TaskConsumer] Started consuming from queue: %s", c.queue)

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				log.Println("[TaskConsumer] Channel closed")
				return nil
			}

			log.Printf("[TaskConsumer] Received message, size: %d bytes", len(msg.Body))
			task, err := parseTask(msg.Body)
			if err != nil {
				log.Printf("[TaskConsumer] ❌ Failed to parse task: %v, body: %s", err, string(msg.Body))
				msg.Nack(false, false) // 不重新入队
				continue
			}

			log.Printf("[TaskConsumer] ✓ Parsed task: %s, URL: %s, Mode: %s, Quality: %s",
				task.TaskID, task.URL, task.Mode, task.Quality)

			// 提交到 Worker 池
			log.Printf("[TaskConsumer] Submitting task %s to worker pool...", task.TaskID)
			c.pool.Submit(task, func(err error) {
				if err != nil {
					log.Printf("[TaskConsumer] ❌ Task %s failed: %v", task.TaskID, err)
					c.retryOrFinalize(msg, task)
				} else {
					log.Printf("[TaskConsumer] ✓ Task %s completed successfully", task.TaskID)
					log.Printf("[TaskConsumer] Acking message for task %s", task.TaskID)
					msg.Ack(false)
				}
			})

		case <-ctx.Done():
			log.Println("[TaskConsumer] Context cancelled, stopping...")
			return ctx.Err()
		}
	}
}

// Stop 停止消费
func (c *TaskConsumer) Stop() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	log.Println("[TaskConsumer] Stopped")
	return nil
}

// parseTask 解析任务消息
func parseTask(body []byte) (*models.DownloadTask, error) {
	var task models.DownloadTask
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, err
	}
	if task.Metadata.Platform == "" {
		task.Metadata.Platform = task.Platform
	}
	if task.Metadata.Title == "" {
		task.Metadata.Title = task.Title
	}
	return &task, nil
}

func (c *TaskConsumer) retryOrFinalize(msg amqp.Delivery, task *models.DownloadTask) {
	currentAttempt := retryAttemptFromHeaders(msg.Headers)
	nextAttempt := currentAttempt + 1
	maxAttempts := c.maxAttempts()

	if nextAttempt >= maxAttempts {
		log.Printf("[TaskConsumer] Retry budget exhausted for task %s (%d/%d), acking terminal failure", task.TaskID, nextAttempt, maxAttempts)
		if err := msg.Ack(false); err != nil {
			log.Printf("[TaskConsumer] Failed to ack exhausted task %s: %v", task.TaskID, err)
		}
		return
	}

	headers := cloneHeaders(msg.Headers)
	headers["x-retry-attempt"] = int32(nextAttempt)

	publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body := msg.Body
	if c.pool != nil {
		if err := c.pool.RefreshTaskProxy(publishCtx, task); err != nil {
			log.Printf("[TaskConsumer] Failed to refresh proxy for retry task %s: %v", task.TaskID, err)
		} else {
			refreshedBody, err := json.Marshal(task)
			if err != nil {
				log.Printf("[TaskConsumer] Failed to marshal refreshed retry task %s: %v", task.TaskID, err)
			} else {
				body = refreshedBody
			}
		}
	}

	if err := c.channel.PublishWithContext(
		publishCtx,
		"",
		c.queue,
		false,
		false,
		amqp.Publishing{
			Headers:         headers,
			ContentType:     msg.ContentType,
			ContentEncoding: msg.ContentEncoding,
			Body:            body,
			DeliveryMode:    amqp.Persistent,
			Priority:        msg.Priority,
			Timestamp:       time.Now(),
		},
	); err != nil {
		log.Printf("[TaskConsumer] Failed to republish retry for task %s: %v", task.TaskID, err)
		if nackErr := msg.Nack(false, true); nackErr != nil {
			log.Printf("[TaskConsumer] Failed to requeue original message for task %s: %v", task.TaskID, nackErr)
		}
		return
	}

	if c.pool != nil && c.pool.repo != nil {
		retryCtx, retryCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := c.pool.repo.IncrementRetry(retryCtx, task.TaskID); err != nil {
			log.Printf("[TaskConsumer] Failed to increment retry count for task %s: %v", task.TaskID, err)
		}
		retryCancel()
	}

	log.Printf("[TaskConsumer] Requeued task %s for retry attempt %d/%d", task.TaskID, nextAttempt+1, maxAttempts)
	if err := msg.Ack(false); err != nil {
		log.Printf("[TaskConsumer] Failed to ack original message for retried task %s: %v", task.TaskID, err)
	}
}

func (c *TaskConsumer) maxAttempts() int {
	if c.pool != nil && c.pool.retryCfg != nil && c.pool.retryCfg.MaxAttempts > 0 {
		return c.pool.retryCfg.MaxAttempts
	}
	return 1
}

func retryAttemptFromHeaders(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	switch value := headers["x-retry-attempt"].(type) {
	case int:
		return value
	case int8:
		return int(value)
	case int16:
		return int(value)
	case int32:
		return int(value)
	case int64:
		return int(value)
	default:
		return 0
	}
}

func cloneHeaders(headers amqp.Table) amqp.Table {
	if len(headers) == 0 {
		return amqp.Table{}
	}

	cloned := make(amqp.Table, len(headers))
	for key, value := range headers {
		cloned[key] = value
	}
	return cloned
}
