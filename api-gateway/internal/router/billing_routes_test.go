package router

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/grpc"

	"vasset/api-gateway/internal/client"
	"vasset/api-gateway/internal/config"
	"vasset/api-gateway/internal/models"
	pb "vasset/api-gateway/proto"
)

func TestBillingRoutesRequireAuth(t *testing.T) {
	t.Parallel()

	r := SetupRouter(&Dependencies{
		Config: &config.Config{
			RateLimit: config.RateLimitConfig{
				GlobalRPS: 100,
				UserRPS:   100,
				Burst:     100,
			},
			FileDownload: config.FileDownloadConfig{BufferSize: 32768},
		},
		GRPCClients: &client.GRPCClients{},
	})

	for _, path := range []string{"/api/v1/user/account", "/api/v1/user/billing/ledger"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 for %s, got %d", path, w.Code)
		}
	}
}

type fakeAdminClientForWelcomeCreditRoutes struct {
	pb.AdminServiceClient

	getCurrentUserResp              *pb.AdminCurrentUserResponse
	getCurrentUserErr               error
	getWelcomeCreditSettingsResp    *pb.AdminWelcomeCreditSettingsResponse
	getWelcomeCreditSettingsErr     error
	updateWelcomeCreditSettingsReq  *pb.AdminUpdateWelcomeCreditSettingsRequest
	updateWelcomeCreditSettingsResp *pb.AdminWelcomeCreditSettingsResponse
	updateWelcomeCreditSettingsErr  error
}

func (f *fakeAdminClientForWelcomeCreditRoutes) GetCurrentUser(context.Context, *pb.AdminSessionRequest, ...grpc.CallOption) (*pb.AdminCurrentUserResponse, error) {
	return f.getCurrentUserResp, f.getCurrentUserErr
}

func (f *fakeAdminClientForWelcomeCreditRoutes) GetWelcomeCreditSettings(context.Context, *pb.AdminEmpty, ...grpc.CallOption) (*pb.AdminWelcomeCreditSettingsResponse, error) {
	return f.getWelcomeCreditSettingsResp, f.getWelcomeCreditSettingsErr
}

func (f *fakeAdminClientForWelcomeCreditRoutes) UpdateWelcomeCreditSettings(_ context.Context, in *pb.AdminUpdateWelcomeCreditSettingsRequest, _ ...grpc.CallOption) (*pb.AdminWelcomeCreditSettingsResponse, error) {
	if in != nil {
		reqCopy := *in
		f.updateWelcomeCreditSettingsReq = &reqCopy
	}
	return f.updateWelcomeCreditSettingsResp, f.updateWelcomeCreditSettingsErr
}

func TestAdminWelcomeCreditRoutesRequireSession(t *testing.T) {
	t.Parallel()

	r := SetupRouter(&Dependencies{
		Config: &config.Config{
			RateLimit:    config.RateLimitConfig{GlobalRPS: 100, UserRPS: 100, Burst: 100},
			FileDownload: config.FileDownloadConfig{BufferSize: 32768},
			AdminSession: config.AdminSessionConfig{CookieName: "vasset_admin_session"},
			GRPC:         config.GRPCConfig{Timeout: time.Second},
		},
		GRPCClients: &client.GRPCClients{AdminClient: &fakeAdminClientForWelcomeCreditRoutes{}},
	})

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/billing/welcome-credit", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for GET welcome-credit route, got %d", getW.Code)
	}

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/billing/welcome-credit", bytes.NewBufferString(`{"enabled":true,"amount_yuan":"1.00","currency_code":"CNY"}`))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for PUT welcome-credit route, got %d", putW.Code)
	}
}

func TestAdminWelcomeCreditRoutesProxySettings(t *testing.T) {
	t.Parallel()

	adminClient := &fakeAdminClientForWelcomeCreditRoutes{
		getCurrentUserResp: &pb.AdminCurrentUserResponse{User: &pb.AdminUser{UserId: "admin-1", Role: 99}},
		getWelcomeCreditSettingsResp: &pb.AdminWelcomeCreditSettingsResponse{
			Enabled:      true,
			AmountYuan:   "1.00",
			CurrencyCode: "CNY",
			UpdatedAt:    "2026-03-21T12:00:00Z",
			UpdatedBy:    "system",
		},
		updateWelcomeCreditSettingsResp: &pb.AdminWelcomeCreditSettingsResponse{
			Enabled:      false,
			AmountYuan:   "1.50",
			CurrencyCode: "CNY",
			UpdatedAt:    "2026-03-21T13:00:00Z",
			UpdatedBy:    "admin-1",
		},
	}

	r := SetupRouter(&Dependencies{
		Config: &config.Config{
			RateLimit:    config.RateLimitConfig{GlobalRPS: 100, UserRPS: 100, Burst: 100},
			FileDownload: config.FileDownloadConfig{BufferSize: 32768},
			AdminSession: config.AdminSessionConfig{CookieName: "vasset_admin_session"},
			GRPC:         config.GRPCConfig{Timeout: time.Second},
		},
		GRPCClients: &client.GRPCClients{AdminClient: adminClient},
	})

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/billing/welcome-credit", nil)
	getReq.AddCookie(&http.Cookie{Name: "vasset_admin_session", Value: "session-1"})
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected status 200 for GET welcome-credit route, got %d", getW.Code)
	}

	getResp := decodeGatewayResponseData(t, getW)
	assertResponseHasFields(t, getResp, "enabled", "amount_yuan", "currency_code", "updated_at", "updated_by")
	if getResp["amount_yuan"] != "1.00" {
		t.Fatalf("unexpected GET amount_yuan: %#v", getResp["amount_yuan"])
	}

	updateBody := map[string]any{"enabled": false, "amount_yuan": "1.50", "currency_code": "CNY"}
	updateBodyBytes, err := json.Marshal(updateBody)
	if err != nil {
		t.Fatalf("failed to marshal update body: %v", err)
	}
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/billing/welcome-credit", bytes.NewBuffer(updateBodyBytes))
	putReq.Header.Set("Content-Type", "application/json")
	putReq.AddCookie(&http.Cookie{Name: "vasset_admin_session", Value: "session-1"})
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("expected status 200 for PUT welcome-credit route, got %d", putW.Code)
	}

	if adminClient.updateWelcomeCreditSettingsReq == nil {
		t.Fatal("expected UpdateWelcomeCreditSettings to be called")
	}
	if adminClient.updateWelcomeCreditSettingsReq.GetAmountYuan() != "1.50" || adminClient.updateWelcomeCreditSettingsReq.GetCurrencyCode() != "CNY" || adminClient.updateWelcomeCreditSettingsReq.GetOperatorUserId() != "admin-1" {
		t.Fatalf("unexpected downstream update request: %+v", adminClient.updateWelcomeCreditSettingsReq)
	}

	putResp := decodeGatewayResponseData(t, putW)
	assertResponseHasFields(t, putResp, "enabled", "amount_yuan", "currency_code", "updated_at", "updated_by")
	if putResp["updated_by"] != "admin-1" {
		t.Fatalf("unexpected PUT updated_by: %#v", putResp["updated_by"])
	}
}

func decodeGatewayResponseData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var resp models.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode gateway response: %v", err)
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("failed to marshal gateway data: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("failed to decode gateway data: %v", err)
	}
	return data
}

func assertResponseHasFields(t *testing.T, data map[string]any, fields ...string) {
	t.Helper()
	for _, field := range fields {
		if _, ok := data[field]; !ok {
			t.Fatalf("expected field %q, got %#v", field, data)
		}
	}
}
