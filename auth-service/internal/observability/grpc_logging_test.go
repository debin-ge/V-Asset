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

	interceptor := UnaryServerInterceptor("auth-service")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-request-id", "rid-123"))

	_, err := interceptor(
		ctx,
		struct{}{},
		&grpc.UnaryServerInfo{FullMethod: "/vasset.AuthService/Login"},
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

	if got := payload["request_id"]; got != "rid-123" {
		t.Fatalf("request_id = %#v, want %q", got, "rid-123")
	}
	if got := payload["grpc_method"]; got != "/vasset.AuthService/Login" {
		t.Fatalf("grpc_method = %#v, want login method", got)
	}
	if got := payload["grpc_code"]; got != "OK" {
		t.Fatalf("grpc_code = %#v, want OK", got)
	}
	if _, ok := payload["latency_ms"]; !ok {
		t.Fatalf("missing latency_ms field: %#v", payload)
	}
}
