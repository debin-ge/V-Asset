package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

type fakeBillingAssetClient struct {
	pb.AssetServiceClient

	getBillingAccountResp     *pb.GetBillingAccountResponse
	getBillingAccountErr      error
	listBillingStatementsResp *pb.ListBillingStatementsResponse
	listBillingStatementsErr  error
}

func (f *fakeBillingAssetClient) GetBillingAccount(context.Context, *pb.GetBillingAccountRequest, ...grpc.CallOption) (*pb.GetBillingAccountResponse, error) {
	return f.getBillingAccountResp, f.getBillingAccountErr
}

func (f *fakeBillingAssetClient) ListBillingStatements(context.Context, *pb.ListBillingStatementsRequest, ...grpc.CallOption) (*pb.ListBillingStatementsResponse, error) {
	return f.listBillingStatementsResp, f.listBillingStatementsErr
}

func TestGetAccountResponseShape(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	handler := NewBillingHandler(&fakeBillingAssetClient{
		getBillingAccountResp: &pb.GetBillingAccountResponse{
			Account: &pb.BillingAccountSnapshot{
				UserId:              "user-1",
				CurrencyCode:        "CNY",
				AvailableBalanceFen: "100",
				ReservedBalanceFen:  "0",
				TotalRechargedFen:   "200",
				TotalSpentFen:       "100",
				TotalTrafficBytes:   1024,
				Status:              1,
				Version:             2,
				CreatedAt:           "2026-03-20T00:00:00Z",
				UpdatedAt:           "2026-03-21T00:00:00Z",
			},
		},
	}, time.Second)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/account", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", "user-1")

	handler.GetAccount(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	data := decodeResponseDataAsMap(t, w)
	assertHasKeys(t, data,
		"user_id", "currency_code", "available_balance_fen", "reserved_balance_fen",
		"total_recharged_fen", "total_spent_fen", "total_traffic_bytes",
		"status", "version", "created_at", "updated_at",
	)
	assertMissingKeys(t, data, "items", "total", "page", "page_size")
}

func TestListStatementsResponseShape(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	handler := NewBillingHandler(&fakeBillingAssetClient{
		listBillingStatementsResp: &pb.ListBillingStatementsResponse{
			Total:    1,
			Page:     1,
			PageSize: 20,
			Items: []*pb.BillingStatementItem{
				{
					StatementId:  "stmt-1",
					Type:         1,
					HistoryId:    101,
					TrafficBytes: 2048,
					AmountFen:    "15",
					Status:       1,
					Remark:       "download",
					CreatedAt:    "2026-03-21T00:00:00Z",
				},
			},
		},
	}, time.Second)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user/billing/ledger?page=1&page_size=20", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", "user-1")

	handler.ListStatements(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	data := decodeResponseDataAsMap(t, w)
	assertHasKeys(t, data, "total", "page", "page_size", "items")
	assertMissingKeys(t, data,
		"user_id", "currency_code", "available_balance_fen", "reserved_balance_fen",
		"total_recharged_fen", "total_spent_fen", "total_traffic_bytes", "version",
	)
}

func decodeResponseDataAsMap(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var resp models.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("failed to encode response data: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("failed to decode response data map: %v", err)
	}

	return data
}

func assertHasKeys(t *testing.T, data map[string]any, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if _, ok := data[key]; !ok {
			t.Fatalf("expected key %q in response data, got %#v", key, data)
		}
	}
}

func assertMissingKeys(t *testing.T, data map[string]any, keys ...string) {
	t.Helper()

	for _, key := range keys {
		if _, ok := data[key]; ok {
			t.Fatalf("did not expect key %q in response data, got %#v", key, data)
		}
	}
}
