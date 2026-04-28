package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"youdlp/api-gateway/internal/models"
	"youdlp/api-gateway/internal/mq"
	pb "youdlp/api-gateway/proto"
)

type fakeAssetDownloadClient struct {
	checkQuotaResp    *pb.CheckQuotaResponse
	checkQuotaErr     error
	createHistoryResp *pb.CreateHistoryResponse
	createHistoryErr  error
	consumeQuotaResp  *pb.ConsumeQuotaResponse
	consumeQuotaErr   error
	refundQuotaResp   *pb.RefundQuotaResponse
	refundQuotaErr    error
	deleteHistoryResp *pb.DeleteHistoryResponse
	deleteHistoryErr  error
	estimateResp      *pb.EstimateDownloadBillingResponse
	estimateErr       error
	holdResp          *pb.HoldInitialDownloadResponse
	holdErr           error
	releaseResp       *pb.ReleaseInitialDownloadResponse
	releaseErr        error
	releaseProxyResp  *pb.ReleaseProxyForTaskResponse
	releaseProxyErr   error

	refundCalls       []string
	deleteCalls       []int64
	releaseCalls      []string
	releaseProxyCalls []string
}

func (f *fakeAssetDownloadClient) CheckQuota(context.Context, *pb.CheckQuotaRequest, ...grpc.CallOption) (*pb.CheckQuotaResponse, error) {
	return f.checkQuotaResp, f.checkQuotaErr
}

func (f *fakeAssetDownloadClient) CreateHistory(_ context.Context, _ *pb.CreateHistoryRequest, _ ...grpc.CallOption) (*pb.CreateHistoryResponse, error) {
	return f.createHistoryResp, f.createHistoryErr
}

func (f *fakeAssetDownloadClient) ConsumeQuota(context.Context, *pb.ConsumeQuotaRequest, ...grpc.CallOption) (*pb.ConsumeQuotaResponse, error) {
	return f.consumeQuotaResp, f.consumeQuotaErr
}

func (f *fakeAssetDownloadClient) RefundQuota(_ context.Context, req *pb.RefundQuotaRequest, _ ...grpc.CallOption) (*pb.RefundQuotaResponse, error) {
	f.refundCalls = append(f.refundCalls, req.UserId)
	return f.refundQuotaResp, f.refundQuotaErr
}

func (f *fakeAssetDownloadClient) DeleteHistory(_ context.Context, req *pb.DeleteHistoryRequest, _ ...grpc.CallOption) (*pb.DeleteHistoryResponse, error) {
	f.deleteCalls = append(f.deleteCalls, req.HistoryId)
	return f.deleteHistoryResp, f.deleteHistoryErr
}

func (f *fakeAssetDownloadClient) EstimateDownloadBilling(context.Context, *pb.EstimateDownloadBillingRequest, ...grpc.CallOption) (*pb.EstimateDownloadBillingResponse, error) {
	return f.estimateResp, f.estimateErr
}

func (f *fakeAssetDownloadClient) HoldInitialDownload(context.Context, *pb.HoldInitialDownloadRequest, ...grpc.CallOption) (*pb.HoldInitialDownloadResponse, error) {
	return f.holdResp, f.holdErr
}

func (f *fakeAssetDownloadClient) ReleaseInitialDownload(_ context.Context, req *pb.ReleaseInitialDownloadRequest, _ ...grpc.CallOption) (*pb.ReleaseInitialDownloadResponse, error) {
	f.releaseCalls = append(f.releaseCalls, req.TaskId)
	return f.releaseResp, f.releaseErr
}

func (f *fakeAssetDownloadClient) ReleaseProxyForTask(_ context.Context, req *pb.ReleaseProxyForTaskRequest, _ ...grpc.CallOption) (*pb.ReleaseProxyForTaskResponse, error) {
	f.releaseProxyCalls = append(f.releaseProxyCalls, req.TaskId)
	return f.releaseProxyResp, f.releaseProxyErr
}

type fakeMediaDownloadClient struct {
	validateResp *pb.ValidateURLResponse
	validateErr  error
	parseResp    *pb.ParseURLResponse
	parseErr     error
}

func (f *fakeMediaDownloadClient) ValidateURL(context.Context, *pb.ValidateURLRequest, ...grpc.CallOption) (*pb.ValidateURLResponse, error) {
	return f.validateResp, f.validateErr
}

func (f *fakeMediaDownloadClient) ParseURL(context.Context, *pb.ParseURLRequest, ...grpc.CallOption) (*pb.ParseURLResponse, error) {
	return f.parseResp, f.parseErr
}

type fakeDownloadPublisher struct {
	publishErr error
	tasks      []*mq.DownloadTask
}

func (f *fakeDownloadPublisher) Publish(_ context.Context, task *mq.DownloadTask) error {
	f.tasks = append(f.tasks, task)
	return f.publishErr
}

func TestSubmitDownloadCompensatesWhenConsumeQuotaFails(t *testing.T) {
	t.Parallel()

	handler, assetClient, _ := newTestDownloadHandler()
	assetClient.consumeQuotaErr = errors.New("quota broken")

	w := performSubmitDownload(t, handler)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	if len(assetClient.deleteCalls) != 1 || assetClient.deleteCalls[0] != 101 {
		t.Fatalf("expected history compensation for 101, got %#v", assetClient.deleteCalls)
	}

	if len(assetClient.refundCalls) != 0 {
		t.Fatalf("did not expect refund on consume failure, got %#v", assetClient.refundCalls)
	}
}

func TestSubmitDownloadCompensatesWhenPublishFails(t *testing.T) {
	t.Parallel()

	handler, assetClient, publisher := newTestDownloadHandler()
	publisher.publishErr = errors.New("mq unavailable")

	w := performSubmitDownload(t, handler)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	if len(assetClient.deleteCalls) != 1 || assetClient.deleteCalls[0] != 101 {
		t.Fatalf("expected history deletion for 101, got %#v", assetClient.deleteCalls)
	}

	if len(assetClient.refundCalls) != 1 || assetClient.refundCalls[0] != "user-1" {
		t.Fatalf("expected quota refund for user-1, got %#v", assetClient.refundCalls)
	}
}

func TestSubmitDownloadReturnsServiceUnavailableWhenPublisherIsTypedNil(t *testing.T) {
	t.Parallel()

	assetClient := &fakeAssetDownloadClient{
		checkQuotaResp: &pb.CheckQuotaResponse{Remaining: 3},
		createHistoryResp: &pb.CreateHistoryResponse{
			HistoryId: 101,
		},
		consumeQuotaResp:  &pb.ConsumeQuotaResponse{Success: true, Remaining: 2},
		refundQuotaResp:   &pb.RefundQuotaResponse{Success: true, Remaining: 3},
		deleteHistoryResp: &pb.DeleteHistoryResponse{Success: true},
	}
	mediaClient := &fakeMediaDownloadClient{
		validateResp: &pb.ValidateURLResponse{Valid: true, Platform: "youtube"},
		parseResp: &pb.ParseURLResponse{
			Title:    "Example",
			Duration: 120,
		},
	}

	var publisher *mq.Publisher
	handler := NewDownloadHandler(assetClient, mediaClient, publisher, time.Second, false)

	w := performSubmitDownload(t, handler)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	if len(assetClient.deleteCalls) != 1 || assetClient.deleteCalls[0] != 101 {
		t.Fatalf("expected history deletion for 101, got %#v", assetClient.deleteCalls)
	}

	if len(assetClient.refundCalls) != 1 || assetClient.refundCalls[0] != "user-1" {
		t.Fatalf("expected quota refund for user-1, got %#v", assetClient.refundCalls)
	}
}

func TestSubmitDownloadAcceptedWithoutCompensation(t *testing.T) {
	t.Parallel()

	handler, assetClient, publisher := newTestDownloadHandler()

	w := performSubmitDownload(t, handler)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", w.Code)
	}

	if len(assetClient.deleteCalls) != 0 {
		t.Fatalf("did not expect history compensation, got %#v", assetClient.deleteCalls)
	}

	if len(assetClient.refundCalls) != 0 {
		t.Fatalf("did not expect quota refund, got %#v", assetClient.refundCalls)
	}

	if len(publisher.tasks) != 1 {
		t.Fatalf("expected exactly one published task, got %d", len(publisher.tasks))
	}
	if publisher.tasks[0].FormatID != "137" {
		t.Fatalf("expected format_id 137, got %q", publisher.tasks[0].FormatID)
	}
	if publisher.tasks[0].SelectedFormat == nil {
		t.Fatal("expected selected format payload to be forwarded")
	}
	if publisher.tasks[0].Quality != "1080p" {
		t.Fatalf("expected selected quality to override request quality, got %q", publisher.tasks[0].Quality)
	}
	if publisher.tasks[0].Format != "webm" {
		t.Fatalf("expected selected extension to override request format, got %q", publisher.tasks[0].Format)
	}

	var resp models.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
}

func newTestDownloadHandler() (*DownloadHandler, *fakeAssetDownloadClient, *fakeDownloadPublisher) {
	assetClient := &fakeAssetDownloadClient{
		checkQuotaResp: &pb.CheckQuotaResponse{Remaining: 3},
		createHistoryResp: &pb.CreateHistoryResponse{
			HistoryId: 101,
		},
		consumeQuotaResp: &pb.ConsumeQuotaResponse{Success: true, Remaining: 2},
		refundQuotaResp:  &pb.RefundQuotaResponse{Success: true, Remaining: 3},
		deleteHistoryResp: &pb.DeleteHistoryResponse{
			Success: true,
		},
		estimateResp: &pb.EstimateDownloadBillingResponse{
			EstimatedIngressBytes: 100,
			EstimatedEgressBytes:  100,
			EstimatedTrafficBytes: 200,
			EstimatedCostYuan:     "10",
			PricingVersion:        1,
		},
		holdResp: &pb.HoldInitialDownloadResponse{
			OrderNo:              "ord-1",
			HoldNo:               "hold-1",
			HeldAmountYuan:       "10",
			AvailableBalanceYuan: "90",
			ReservedBalanceYuan:  "10",
		},
		releaseResp: &pb.ReleaseInitialDownloadResponse{
			Success:            true,
			OrderNo:            "ord-1",
			ReleasedAmountYuan: "10",
		},
		releaseProxyResp: &pb.ReleaseProxyForTaskResponse{Success: true},
	}
	mediaClient := &fakeMediaDownloadClient{
		validateResp: &pb.ValidateURLResponse{Valid: true, Platform: "youtube"},
		parseResp: &pb.ParseURLResponse{
			Title:    "Example",
			Duration: 120,
		},
	}
	publisher := &fakeDownloadPublisher{}

	return NewDownloadHandler(assetClient, mediaClient, publisher, time.Second, false), assetClient, publisher
}

func performSubmitDownload(t *testing.T, handler *DownloadHandler) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)

	body := bytes.NewBufferString(`{"url":"https://example.com/video","mode":"quick_download","quality":"best","format":"mp4","format_id":"137","selected_format":{"format_id":"137","quality":"1080p","extension":"webm","height":1080,"video_codec":"vp09","audio_codec":"none"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/download", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", "user-1")

	handler.SubmitDownload(c)
	return w
}
