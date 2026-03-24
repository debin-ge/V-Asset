package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"youdlp/api-gateway/internal/trace"
)

func TestLoggerPropagatesIncomingRequestIDToContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Logger())

	var capturedRequestID string
	router.GET("/test", func(c *gin.Context) {
		capturedRequestID = trace.RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	restore := captureStdLogger(t)
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/test?password=super-secret&authorization=Bearer+abc&cookie=sid-1&foo=bar", nil)
	req.Header.Set("X-Request-ID", "rid-incoming")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got := resp.Header().Get("X-Request-ID"); got != "rid-incoming" {
		t.Fatalf("X-Request-ID = %q, want %q", got, "rid-incoming")
	}
	if capturedRequestID != "rid-incoming" {
		t.Fatalf("request_id in context = %q, want %q", capturedRequestID, "rid-incoming")
	}

	logLine := respLogger.String()
	if !strings.Contains(logLine, `"request_id":"rid-incoming"`) {
		t.Fatalf("log output missing request_id: %s", logLine)
	}
	if strings.Contains(logLine, "super-secret") {
		t.Fatalf("log output leaked sensitive query value: %s", logLine)
	}
	if strings.Contains(logLine, "Bearer+abc") || strings.Contains(logLine, "sid-1") {
		t.Fatalf("log output leaked authorization or cookie value: %s", logLine)
	}
	if !strings.Contains(logLine, "REDACTED") {
		t.Fatalf("log output did not redact sensitive query value: %s", logLine)
	}
}

func TestLoggerGeneratesRequestIDWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Logger())

	var capturedRequestID string
	router.GET("/test", func(c *gin.Context) {
		capturedRequestID = trace.RequestIDFromContext(c.Request.Context())
		c.Status(http.StatusNoContent)
	})

	restore := captureStdLogger(t)
	defer restore()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	generated := strings.TrimSpace(resp.Header().Get("X-Request-ID"))
	if generated == "" {
		t.Fatalf("expected generated X-Request-ID header")
	}
	if capturedRequestID != generated {
		t.Fatalf("request_id in context = %q, want %q", capturedRequestID, generated)
	}
	if !strings.Contains(respLogger.String(), generated) {
		t.Fatalf("log output missing generated request_id: %s", respLogger.String())
	}
}

var respLogger bytes.Buffer

func captureStdLogger(t *testing.T) func() {
	t.Helper()
	respLogger.Reset()

	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetFlags(0)
	log.SetOutput(io.MultiWriter(&respLogger))

	return func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	}
}
