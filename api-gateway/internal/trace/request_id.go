package trace

import (
	"context"
	"strings"
)

type requestIDKeyType struct{}

var requestIDKey requestIDKeyType

// WithRequestID stores request_id in context for cross-layer propagation.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil || strings.TrimSpace(requestID) == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext returns request_id if present.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDKey).(string)
	return strings.TrimSpace(requestID)
}
