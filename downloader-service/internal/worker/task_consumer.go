package worker

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"

	"vasset/downloader-service/internal/config"
	"vasset/downloader-service/internal/models"
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
	// 连接 RabbitMQ
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, err
	}

	// 创建通道
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// 声明队列
	_, err = ch.QueueDeclare(
		cfg.Queue, // 队列名
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	// 设置预取数
	if err := ch.Qos(cfg.PrefetchCount, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

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

			task, err := parseTask(msg.Body)
			if err != nil {
				log.Printf("[TaskConsumer] Failed to parse task: %v", err)
				msg.Nack(false, false) // 不重新入队
				continue
			}

			log.Printf("[TaskConsumer] Received task: %s", task.TaskID)

			// 提交到 Worker 池
			c.pool.Submit(task, func(err error) {
				if err != nil {
					log.Printf("[TaskConsumer] Task %s failed: %v", task.TaskID, err)
					msg.Nack(false, true) // 重新入队
				} else {
					log.Printf("[TaskConsumer] Task %s completed", task.TaskID)
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
	return &task, nil
}
