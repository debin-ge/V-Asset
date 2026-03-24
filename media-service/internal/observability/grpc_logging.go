package observability

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const requestIDMetadataKey = "x-request-id"

// UnaryServerInterceptor records structured gRPC access logs.
func UnaryServerInterceptor(service string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)

		payload := map[string]any{
			"service":     service,
			"layer":       "grpc",
			"request_id":  requestIDFromIncomingContext(ctx),
			"grpc_method": info.FullMethod,
			"grpc_code":   status.Code(err).String(),
			"latency_ms":  time.Since(start).Milliseconds(),
			"peer_ip":     peerIP(ctx),
		}

		serialized, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			log.Printf(`{"service":"%s","layer":"grpc","message":"failed to marshal grpc access log","error":"%v"}`, service, marshalErr)
		} else {
			log.Printf("%s", serialized)
		}

		return resp, err
	}
}

func requestIDFromIncomingContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	values := md.Get(requestIDMetadataKey)
	if len(values) == 0 {
		return ""
	}

	return strings.TrimSpace(values[0])
}

func peerIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return ""
	}

	host, _, err := net.SplitHostPort(p.Addr.String())
	if err == nil {
		return host
	}

	return p.Addr.String()
}
