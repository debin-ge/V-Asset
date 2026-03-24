package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

type fakeFileAssetClient struct {
	getFileInfoResp *pb.GetFileInfoResponse
	getFileInfoErr  error
}

func (f *fakeFileAssetClient) GetFileInfo(context.Context, *pb.GetFileInfoRequest, ...grpc.CallOption) (*pb.GetFileInfoResponse, error) {
	return f.getFileInfoResp, f.getFileInfoErr
}

func (f *fakeFileAssetClient) PrepareFileTransferBilling(context.Context, *pb.PrepareFileTransferBillingRequest, ...grpc.CallOption) (*pb.PrepareFileTransferBillingResponse, error) {
	return &pb.PrepareFileTransferBillingResponse{}, nil
}

func (f *fakeFileAssetClient) CompleteFileTransferBilling(context.Context, *pb.CompleteFileTransferBillingRequest, ...grpc.CallOption) (*pb.CompleteFileTransferBillingResponse, error) {
	return &pb.CompleteFileTransferBillingResponse{}, nil
}

func (f *fakeFileAssetClient) AbortFileTransferBilling(context.Context, *pb.AbortFileTransferBillingRequest, ...grpc.CallOption) (*pb.AbortFileTransferBillingResponse, error) {
	return &pb.AbortFileTransferBillingResponse{}, nil
}

type fakeDownloadTicketStore struct {
	saveCalls []savedTicket
	payloads  map[string]*downloadTicketPayload
	saveErr   error
	loadErr   error
}

type savedTicket struct {
	ticket  string
	payload *downloadTicketPayload
	ttl     time.Duration
}

func (f *fakeDownloadTicketStore) Save(_ context.Context, ticket string, payload *downloadTicketPayload, ttl time.Duration) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.payloads == nil {
		f.payloads = make(map[string]*downloadTicketPayload)
	}
	clone := *payload
	f.payloads[ticket] = &clone
	f.saveCalls = append(f.saveCalls, savedTicket{ticket: ticket, payload: &clone, ttl: ttl})
	return nil
}

func (f *fakeDownloadTicketStore) Load(_ context.Context, ticket string) (*downloadTicketPayload, error) {
	if f.loadErr != nil {
		return nil, f.loadErr
	}
	payload, ok := f.payloads[ticket]
	if !ok {
		return nil, errDownloadTicketNotFound
	}
	clone := *payload
	return &clone, nil
}

func TestCreateDownloadTicketReturnsTicket(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	assetClient := &fakeFileAssetClient{
		getFileInfoResp: &pb.GetFileInfoResponse{
			FilePath: "/tmp/example.mp4",
			FileName: "example.mp4",
		},
	}
	ticketStore := &fakeDownloadTicketStore{}
	handler := NewFileHandler(assetClient, ticketStore, time.Second, 32*1024, false)

	body := bytes.NewBufferString(`{"history_id":42}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/download/file-ticket", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req
	c.Set("user_id", "user-1")

	handler.CreateDownloadTicket(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if len(ticketStore.saveCalls) != 1 {
		t.Fatalf("expected one saved ticket, got %d", len(ticketStore.saveCalls))
	}
	if ticketStore.saveCalls[0].payload.UserID != "user-1" || ticketStore.saveCalls[0].payload.HistoryID != 42 {
		t.Fatalf("unexpected saved payload: %#v", ticketStore.saveCalls[0].payload)
	}
	if ticketStore.saveCalls[0].ttl != downloadTicketTTL {
		t.Fatalf("expected ttl %v, got %v", downloadTicketTTL, ticketStore.saveCalls[0].ttl)
	}

	var resp models.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	dataBytes, _ := json.Marshal(resp.Data)
	var data models.FileDownloadTicketResponse
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		t.Fatalf("failed to decode data: %v", err)
	}
	if data.Ticket == "" {
		t.Fatal("expected non-empty ticket")
	}
	if data.ExpiresIn != int64(downloadTicketTTL/time.Second) {
		t.Fatalf("expected expires_in %d, got %d", int64(downloadTicketTTL/time.Second), data.ExpiresIn)
	}
}

func TestDownloadFileByTicketStreamsFile(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	filePath := filepath.Join(dir, "sample.txt")
	content := []byte("hello from browser download")
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	assetClient := &fakeFileAssetClient{
		getFileInfoResp: &pb.GetFileInfoResponse{
			FilePath: filePath,
			FileName: "sample.txt",
		},
	}
	ticketStore := &fakeDownloadTicketStore{
		payloads: map[string]*downloadTicketPayload{
			"ticket-1": {UserID: "user-1", HistoryID: 42},
		},
	}
	handler := NewFileHandler(assetClient, ticketStore, time.Second, 8, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/download/file/browser?ticket=ticket-1", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	handler.DownloadFileByTicket(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if body := w.Body.Bytes(); !bytes.Equal(body, content) {
		t.Fatalf("unexpected file body: %q", string(body))
	}
	disposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(disposition, "attachment;") {
		t.Fatalf("expected attachment disposition, got %q", disposition)
	}
}
