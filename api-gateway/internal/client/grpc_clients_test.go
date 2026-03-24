package client

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"youdlp/api-gateway/internal/trace"
)

func TestRequestIDUnaryClientInterceptorInjectsMetadata(t *testing.T) {
	ctx := trace.WithRequestID(context.Background(), "rid-123")

	err := requestIDUnaryClientInterceptor(
		ctx,
		"/youdlp.AuthService/Login",
		nil,
		nil,
		nil,
		func(ctx context.Context, _ string, _ any, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatalf("expected outgoing metadata")
			}
			if got := md.Get("x-request-id"); len(got) != 1 || got[0] != "rid-123" {
				t.Fatalf("x-request-id = %#v, want [\"rid-123\"]", got)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("requestIDUnaryClientInterceptor returned error: %v", err)
	}
}

func TestRequestIDUnaryClientInterceptorSkipsWhenMissing(t *testing.T) {
	err := requestIDUnaryClientInterceptor(
		context.Background(),
		"/youdlp.AuthService/Login",
		nil,
		nil,
		nil,
		func(ctx context.Context, _ string, _ any, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			md, ok := metadata.FromOutgoingContext(ctx)
			if ok {
				if got := md.Get("x-request-id"); len(got) > 0 {
					t.Fatalf("x-request-id = %#v, want empty", got)
				}
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("requestIDUnaryClientInterceptor returned error: %v", err)
	}
}
