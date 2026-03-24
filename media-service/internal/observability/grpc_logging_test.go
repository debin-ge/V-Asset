package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestUnaryServerInterceptorLogsRequestIDAndMethod(t *testing.T) {
	var buffer bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetFlags(0)
	log.SetOutput(&buffer)
	defer func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	}()

	interceptor := UnaryServerInterceptor("media-service")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-request-id", "rid-media"))

	_, err := interceptor(
		ctx,
		struct{}{},
		&grpc.UnaryServerInfo{FullMethod: "/youdlp.MediaService/ParseURL"},
		func(context.Context, any) (any, error) {
			return "ok", nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}

	line := strings.TrimSpace(buffer.String())
	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("failed to parse log line %q: %v", line, err)
	}

	if got := payload["request_id"]; got != "rid-media" {
		t.Fatalf("request_id = %#v, want %q", got, "rid-media")
	}
	if got := payload["grpc_method"]; got != "/youdlp.MediaService/ParseURL" {
		t.Fatalf("grpc_method = %#v, want ParseURL method", got)
	}
	if got := payload["grpc_code"]; got != "OK" {
		t.Fatalf("grpc_code = %#v, want OK", got)
	}
	if _, ok := payload["latency_ms"]; !ok {
		t.Fatalf("missing latency_ms field: %#v", payload)
	}
}
