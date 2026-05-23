package mq

import (
	"context"
	"errors"
	"testing"

	"youdlp/api-gateway/internal/config"
)

func TestNewPublisherReturnsPublisherWhenInitialConnectionFails(t *testing.T) {
	t.Parallel()

	publisher, err := NewPublisher(&config.RabbitMQConfig{
		URL:        "amqp://guest:guest@127.0.0.1:1/",
		Exchange:   "test.exchange",
		Queue:      "test.queue",
		RoutingKey: "test.routing",
	})
	if err == nil {
		t.Fatal("expected initial connection error")
	}
	if publisher == nil {
		t.Fatal("expected publisher to be returned so it can reconnect later")
	}
	t.Cleanup(func() {
		if closeErr := publisher.Close(); closeErr != nil {
			t.Fatalf("close publisher: %v", closeErr)
		}
	})
}

func TestPublishReturnsUnavailableWhenDisconnected(t *testing.T) {
	t.Parallel()

	publisher := &Publisher{}
	err := publisher.Publish(context.Background(), &DownloadTask{TaskID: "task-1"})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
}
