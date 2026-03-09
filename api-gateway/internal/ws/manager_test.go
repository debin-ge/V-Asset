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
