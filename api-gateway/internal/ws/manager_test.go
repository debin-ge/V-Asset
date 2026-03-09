package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

type fakeAuthVerifier struct {
	verifyResp *pb.VerifyTokenResponse
	verifyErr  error
}

func (f *fakeAuthVerifier) VerifyToken(context.Context, *pb.VerifyTokenRequest, ...grpc.CallOption) (*pb.VerifyTokenResponse, error) {
	return f.verifyResp, f.verifyErr
}

type fakeTaskAccessChecker struct {
	resp *pb.GetHistoryByTaskResponse
	err  error
}

func (f *fakeTaskAccessChecker) GetHistoryByTask(context.Context, *pb.GetHistoryByTaskRequest, ...grpc.CallOption) (*pb.GetHistoryByTaskResponse, error) {
	return f.resp, f.err
}

func TestValidateTaskAccessNotFoundMapsToForbidden(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &fakeAuthVerifier{}, &fakeTaskAccessChecker{
		err: grpcstatus.Error(codes.NotFound, "missing"),
	})

	err := manager.validateTaskAccess(context.Background(), "user-1", "task-1")
	if err == nil || err.Error() != "task not found or access denied" {
		t.Fatalf("expected not found access error, got %v", err)
	}
}

func TestValidateTaskAccessInternalFailureIsHidden(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &fakeAuthVerifier{}, &fakeTaskAccessChecker{
		err: grpcstatus.Error(codes.Internal, "db broken"),
	})

	err := manager.validateTaskAccess(context.Background(), "user-1", "task-1")
	if err == nil || err.Error() != "failed to validate task access" {
		t.Fatalf("expected generic validation error, got %v", err)
	}
}

func TestHandleConnectionRequiresTaskID(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &fakeAuthVerifier{
		verifyResp: &pb.VerifyTokenResponse{Valid: true, UserId: "user-1"},
	}, &fakeTaskAccessChecker{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/ws/progress?token=test-token", nil)

	manager.HandleConnection(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	assertResponseMessage(t, w, "task_id is required")
}

func TestHandleConnectionRejectsUnauthorizedTaskAccess(t *testing.T) {
	t.Parallel()

	manager := NewManager(nil, &fakeAuthVerifier{
		verifyResp: &pb.VerifyTokenResponse{Valid: true, UserId: "user-1"},
	}, &fakeTaskAccessChecker{
		err: grpcstatus.Error(codes.NotFound, "missing"),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/ws/progress?token=test-token&task_id=task-1", nil)

	manager.HandleConnection(c)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	assertResponseMessage(t, w, "task not found or access denied")
}

func TestCheckOriginAllowsSameHostnameWithPortDifference(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://ytdlp.obstream.com:8080/api/v1/ws/progress", nil)
	req.Host = "ytdlp.obstream.com:8080"
	req.Header.Set("Origin", "http://ytdlp.obstream.com")

	if !upgrader.CheckOrigin(req) {
		t.Fatalf("expected same hostname origin to be allowed despite port difference")
	}
}

func TestCheckOriginUsesForwardedHeaders(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://internal:8080/api/v1/ws/progress", nil)
	req.Host = "internal:8080"
	req.Header.Set("Origin", "https://ytdlp.obstream.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "ytdlp.obstream.com")

	if !upgrader.CheckOrigin(req) {
		t.Fatalf("expected forwarded host/scheme to be accepted")
	}
}

func TestCheckOriginUsesCloudflareVisitorScheme(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://internal:8080/api/v1/ws/progress", nil)
	req.Host = "internal:8080"
	req.Header.Set("Origin", "https://ytdlp.obstream.com")
	req.Header.Set("X-Forwarded-Host", "ytdlp.obstream.com")
	req.Header.Set("CF-Visitor", `{"scheme":"https"}`)

	if !upgrader.CheckOrigin(req) {
		t.Fatalf("expected CF-Visitor https scheme to be accepted")
	}
}

func TestCheckOriginAllowsInternalProxyHostWithoutForwardedHost(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "http://api-gateway:8080/api/v1/ws/progress", nil)
	req.Host = "api-gateway:8080"
	req.Header.Set("Origin", "http://ytdlp.obstream.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	if !upgrader.CheckOrigin(req) {
		t.Fatalf("expected internal proxy host to trust same-site origin when forwarded host is unavailable")
	}
}

func TestAllowedOriginSchemeAllowsHttpOriginForHTTPSProxy(t *testing.T) {
	t.Parallel()

	if !isAllowedOriginScheme("http", "https") {
		t.Fatalf("expected http origin to be allowed for https proxy")
	}

	if isAllowedOriginScheme("https", "http") {
		t.Fatalf("did not expect https origin to be allowed for http proxy")
	}
}

func assertResponseMessage(t *testing.T, w *httptest.ResponseRecorder, want string) {
	t.Helper()

	var resp models.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if resp.Message != want {
		t.Fatalf("expected message %q, got %q", want, resp.Message)
	}
}
