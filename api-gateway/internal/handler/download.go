package handler

import (
	"context"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"youdlp/api-gateway/internal/middleware"
	"youdlp/api-gateway/internal/models"
	"youdlp/api-gateway/internal/mq"
	pb "youdlp/api-gateway/proto"
)

// DownloadHandler 下载处理器
type DownloadHandler struct {
	assetClient    assetDownloadClient
	mediaClient    mediaDownloadClient
	publisher      downloadPublisher
	timeout        time.Duration
	billingEnabled bool
}

type assetDownloadClient interface {
	CheckQuota(ctx context.Context, in *pb.CheckQuotaRequest, opts ...grpc.CallOption) (*pb.CheckQuotaResponse, error)
	CreateHistory(ctx context.Context, in *pb.CreateHistoryRequest, opts ...grpc.CallOption) (*pb.CreateHistoryResponse, error)
	ConsumeQuota(ctx context.Context, in *pb.ConsumeQuotaRequest, opts ...grpc.CallOption) (*pb.ConsumeQuotaResponse, error)
	RefundQuota(ctx context.Context, in *pb.RefundQuotaRequest, opts ...grpc.CallOption) (*pb.RefundQuotaResponse, error)
	DeleteHistory(ctx context.Context, in *pb.DeleteHistoryRequest, opts ...grpc.CallOption) (*pb.DeleteHistoryResponse, error)
	EstimateDownloadBilling(ctx context.Context, in *pb.EstimateDownloadBillingRequest, opts ...grpc.CallOption) (*pb.EstimateDownloadBillingResponse, error)
	HoldInitialDownload(ctx context.Context, in *pb.HoldInitialDownloadRequest, opts ...grpc.CallOption) (*pb.HoldInitialDownloadResponse, error)
	ReleaseInitialDownload(ctx context.Context, in *pb.ReleaseInitialDownloadRequest, opts ...grpc.CallOption) (*pb.ReleaseInitialDownloadResponse, error)
	ReleaseProxyForTask(ctx context.Context, in *pb.ReleaseProxyForTaskRequest, opts ...grpc.CallOption) (*pb.ReleaseProxyForTaskResponse, error)
}

type mediaDownloadClient interface {
	ValidateURL(ctx context.Context, in *pb.ValidateURLRequest, opts ...grpc.CallOption) (*pb.ValidateURLResponse, error)
	ParseURL(ctx context.Context, in *pb.ParseURLRequest, opts ...grpc.CallOption) (*pb.ParseURLResponse, error)
}

type downloadPublisher interface {
	Publish(ctx context.Context, task *mq.DownloadTask) error
}

type unavailableDownloadPublisher struct {
	reason string
}

func (p unavailableDownloadPublisher) Publish(context.Context, *mq.DownloadTask) error {
	if p.reason != "" {
		return status.Error(codes.Unavailable, p.reason)
	}
	return status.Error(codes.Unavailable, "download queue unavailable")
}

// NewDownloadHandler 创建下载处理器
func NewDownloadHandler(
	assetClient assetDownloadClient,
	mediaClient mediaDownloadClient,
	publisher downloadPublisher,
	timeout time.Duration,
	billingEnabled bool,
) *DownloadHandler {
	if isNilDownloadPublisher(publisher) {
		log.Printf("[Download] ⚠ MQ publisher is unavailable, download submissions will return 503 until RabbitMQ recovers")
		publisher = unavailableDownloadPublisher{reason: "download queue unavailable"}
	}

	return &DownloadHandler{
		assetClient:    assetClient,
		mediaClient:    mediaClient,
		publisher:      publisher,
		timeout:        timeout,
		billingEnabled: billingEnabled,
	}
}

// SubmitDownload 提交下载任务
func (h *DownloadHandler) SubmitDownload(c *gin.Context) {
	log.Printf("[Download] Received download request from %s", c.ClientIP())
	var req models.DownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Download] ❌ Failed to parse request: %v", err)
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	normalizeDownloadRequest(&req)
	log.Printf("[Download] ✓ Request parsed - URL: %s, Mode: %s, Quality: %s, Format: %s",
		req.URL, req.Mode, req.Quality, req.Format)

	userID := middleware.GetUserID(c)
	if userID == "" {
		log.Printf("[Download] ❌ User not authenticated")
		models.Unauthorized(c, "user not authenticated")
		return
	}
	log.Printf("[Download] ✓ User authenticated: %s", userID)

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	if !h.billingEnabled {
		log.Printf("[Download] Step 1/8: Checking quota for user %s...", userID)
		quotaResp, err := h.assetClient.CheckQuota(ctx, &pb.CheckQuotaRequest{UserId: userID})
		if err != nil {
			log.Printf("[Download] ❌ Failed to check quota: %v", err)
			writeGRPCError(c, err)
			return
		}
		log.Printf("[Download] ✓ Quota check passed - Remaining: %d", quotaResp.Remaining)
		if quotaResp.Remaining <= 0 {
			log.Printf("[Download] ❌ Quota exceeded for user %s", userID)
			models.Forbidden(c, "quota exceeded, please try again tomorrow")
			return
		}
	}

	log.Printf("[Download] Step 2/8: Validating URL: %s...", req.URL)
	validateResp, err := h.mediaClient.ValidateURL(ctx, &pb.ValidateURLRequest{Url: req.URL})
	if err != nil {
		log.Printf("[Download] ❌ Failed to validate URL: %v", err)
		writeGRPCError(c, err)
		return
	}
	if !validateResp.Valid {
		log.Printf("[Download] ❌ Invalid URL: %s", validateResp.Message)
		models.BadRequest(c, "invalid URL: "+validateResp.Message)
		return
	}
	log.Printf("[Download] ✓ URL validated - Platform: %s", validateResp.Platform)

	log.Printf("[Download] Step 3/8: Generating task ID...")
	taskID := uuid.New().String()
	log.Printf("[Download] ✓ Task ID generated: %s", taskID)

	log.Printf("[Download] Step 4/8: Parsing URL to get metadata with task %s...", taskID)
	parseResp, err := h.mediaClient.ParseURL(ctx, &pb.ParseURLRequest{
		Url:    req.URL,
		TaskId: taskID,
	})
	if err != nil {
		log.Printf("[Download] ❌ Failed to parse URL: %v", err)
		writeGRPCError(c, err)
		return
	}
	log.Printf("[Download] ✓ URL parsed - Title: %s, Duration: %ds", parseResp.Title, parseResp.Duration)

	log.Printf("[Download] Step 5/8: Creating download history for task %s...", taskID)
	historyResp, err := h.assetClient.CreateHistory(ctx, &pb.CreateHistoryRequest{
		UserId:    userID,
		TaskId:    taskID,
		Url:       req.URL,
		Platform:  validateResp.Platform,
		Title:     parseResp.Title,
		Mode:      req.Mode,
		Quality:   req.Quality,
		Thumbnail: parseResp.Thumbnail,
		Duration:  parseResp.Duration,
		Author:    parseResp.Author,
	})
	if err != nil {
		log.Printf("[Download] ❌ Failed to create history: %v", err)
		h.releaseProxyBinding(ctx, taskID, "create history failed")
		writeGRPCError(c, err)
		return
	}
	log.Printf("[Download] ✓ History created - HistoryID: %d", historyResp.HistoryId)

	if h.billingEnabled {
		log.Printf("[Download] Step 6/8: Estimating billing for task %s...", taskID)
		estimateResp, err := h.assetClient.EstimateDownloadBilling(ctx, &pb.EstimateDownloadBillingRequest{
			UserId:         userID,
			Url:            req.URL,
			Platform:       validateResp.Platform,
			Mode:           req.Mode,
			SelectedFormat: toBillingSelectedFormat(req.SelectedFormat),
		})
		if err != nil {
			log.Printf("[Download] ❌ Failed to estimate billing: %v", err)
			h.cleanupFailedSubmission(ctx, userID, historyResp.HistoryId, taskID, false, false)
			writeGRPCError(c, err)
			return
		}

		log.Printf("[Download] Step 7/8: Holding initial billing for task %s...", taskID)
		_, err = h.assetClient.HoldInitialDownload(ctx, &pb.HoldInitialDownloadRequest{
			UserId:                userID,
			HistoryId:             historyResp.HistoryId,
			TaskId:                taskID,
			EstimatedIngressBytes: estimateResp.GetEstimatedIngressBytes(),
			EstimatedEgressBytes:  estimateResp.GetEstimatedEgressBytes(),
			EstimatedTrafficBytes: estimateResp.GetEstimatedTrafficBytes(),
			EstimatedCostYuan:     estimateResp.GetEstimatedCostYuan(),
			PricingVersion:        estimateResp.GetPricingVersion(),
		})
		if err != nil {
			log.Printf("[Download] ❌ Failed to hold initial billing: %v", err)
			h.cleanupFailedSubmission(ctx, userID, historyResp.HistoryId, taskID, false, false)
			writeGRPCError(c, err)
			return
		}
		log.Printf("[Download] ✓ Billing hold created")
	} else {
		log.Printf("[Download] Step 6/8: Consuming quota for user %s...", userID)
		_, err = h.assetClient.ConsumeQuota(ctx, &pb.ConsumeQuotaRequest{UserId: userID})
		if err != nil {
			log.Printf("[Download] ❌ Failed to consume quota: %v", err)
			h.cleanupFailedSubmission(ctx, userID, historyResp.HistoryId, taskID, false, false)
			writeGRPCError(c, err)
			return
		}
		log.Printf("[Download] ✓ Quota consumed")
	}

	log.Printf("[Download] Step 8/8: Publishing task %s to RabbitMQ...", taskID)
	task := &mq.DownloadTask{
		TaskID:         taskID,
		UserID:         userID,
		HistoryID:      historyResp.HistoryId,
		URL:            req.URL,
		Mode:           req.Mode,
		Quality:        req.Quality,
		Format:         req.Format,
		FormatID:       req.FormatID,
		SelectedFormat: toSelectedFormatMessage(req.SelectedFormat),
		Platform:       validateResp.Platform,
		Title:          parseResp.Title,
		CookieID:       parseResp.CookieId,
		ProxyURL:       parseResp.ProxyUrl,
		ProxyLeaseID:   parseResp.ProxyLeaseId,
		ProxyExpireAt:  parseResp.ProxyExpireAt,
	}

	if err := h.publisher.Publish(ctx, task); err != nil {
		log.Printf("[Download] ❌ Failed to publish task to RabbitMQ: %v", err)
		h.cleanupFailedSubmission(ctx, userID, historyResp.HistoryId, taskID, !h.billingEnabled, h.billingEnabled)
		if status.Code(err) == codes.Unavailable {
			models.Error(c, http.StatusServiceUnavailable, grpcErrorMessage(err))
			return
		}
		writeGRPCError(c, err)
		return
	}
	log.Printf("[Download] ✓ Task %s published to RabbitMQ", taskID)

	estimatedTime := estimateDownloadTime(parseResp.Duration, req.Quality)
	log.Printf("[Download] ✅ Download request completed successfully - TaskID: %s, EstimatedTime: %ds", taskID, estimatedTime)
	models.Accepted(c, models.DownloadResponse{
		TaskID:        taskID,
		HistoryID:     historyResp.HistoryId,
		EstimatedTime: estimatedTime,
	})
}

func normalizeDownloadRequest(req *models.DownloadRequest) {
	if req == nil {
		return
	}

	if req.SelectedFormat != nil {
		if req.SelectedFormat.FormatID != "" {
			req.FormatID = req.SelectedFormat.FormatID
		}
		if req.SelectedFormat.Quality != "" {
			req.Quality = req.SelectedFormat.Quality
		}
		if req.SelectedFormat.Extension != "" {
			req.Format = req.SelectedFormat.Extension
		}
	}

	if req.Format == "" {
		req.Format = "mp4"
	}
	if req.Quality == "" {
		req.Quality = "best"
	}
}

func toSelectedFormatMessage(selected *models.SelectedFormat) *mq.SelectedFormatMessage {
	if selected == nil {
		return nil
	}

	return &mq.SelectedFormatMessage{
		FormatID:   selected.FormatID,
		Quality:    selected.Quality,
		Extension:  selected.Extension,
		Filesize:   selected.Filesize,
		Height:     selected.Height,
		Width:      selected.Width,
		FPS:        selected.FPS,
		VideoCodec: selected.VideoCodec,
		AudioCodec: selected.AudioCodec,
		VBR:        selected.VBR,
		ABR:        selected.ABR,
		ASR:        selected.ASR,
	}
}

func toBillingSelectedFormat(selected *models.SelectedFormat) *pb.BillingSelectedFormat {
	if selected == nil {
		return nil
	}

	return &pb.BillingSelectedFormat{
		FormatId:   selected.FormatID,
		Quality:    selected.Quality,
		Extension:  selected.Extension,
		Filesize:   selected.Filesize,
		Height:     selected.Height,
		Width:      selected.Width,
		Fps:        selected.FPS,
		VideoCodec: selected.VideoCodec,
		AudioCodec: selected.AudioCodec,
		Vbr:        selected.VBR,
		Abr:        selected.ABR,
		Asr:        selected.ASR,
	}
}

func (h *DownloadHandler) cleanupFailedSubmission(parentCtx context.Context, userID string, historyID int64, taskID string, refundQuota bool, releaseBilling bool) {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	compensateCtx, cancel := context.WithTimeout(parentCtx, h.timeout)
	defer cancel()

	h.releaseProxyBinding(compensateCtx, taskID, "submit compensation")

	if refundQuota {
		if _, err := h.assetClient.RefundQuota(compensateCtx, &pb.RefundQuotaRequest{
			UserId: userID,
		}); err != nil {
			log.Printf("[Download] ⚠ Failed to refund quota for user %s during compensation: %v", userID, err)
		} else {
			log.Printf("[Download] ✓ Refunded quota for user %s during compensation", userID)
		}
	}

	if releaseBilling && taskID != "" {
		if _, err := h.assetClient.ReleaseInitialDownload(compensateCtx, &pb.ReleaseInitialDownloadRequest{
			TaskId: taskID,
			Reason: "submit compensation",
		}); err != nil && status.Code(err) != codes.NotFound {
			log.Printf("[Download] ⚠ Failed to release billing hold for task %s during compensation: %v", taskID, err)
		} else {
			log.Printf("[Download] ✓ Released billing hold for task %s during compensation", taskID)
		}
	}

	if historyID == 0 {
		return
	}

	if _, err := h.assetClient.DeleteHistory(compensateCtx, &pb.DeleteHistoryRequest{
		HistoryId: historyID,
		UserId:    userID,
	}); err != nil {
		log.Printf("[Download] ⚠ Failed to delete history %d during compensation: %v", historyID, err)
	} else {
		log.Printf("[Download] ✓ Deleted history %d during compensation", historyID)
	}
}

func (h *DownloadHandler) releaseProxyBinding(ctx context.Context, taskID, reason string) {
	if taskID == "" {
		return
	}
	if _, err := h.assetClient.ReleaseProxyForTask(ctx, &pb.ReleaseProxyForTaskRequest{
		TaskId: taskID,
		Reason: reason,
	}); err != nil && status.Code(err) != codes.NotFound && status.Code(err) != codes.Unimplemented {
		log.Printf("[Download] ⚠ Failed to release proxy binding for task %s: %v", taskID, err)
	} else {
		log.Printf("[Download] ✓ Released proxy binding for task %s during compensation", taskID)
	}
}

// estimateDownloadTime 估算下载时间
func estimateDownloadTime(duration int64, quality string) int {
	base := int(duration / 10)
	if base < 30 {
		base = 30
	}

	switch quality {
	case "1080p":
		return base + 60
	case "720p":
		return base + 30
	default:
		return base + 15
	}
}

func isNilDownloadPublisher(publisher downloadPublisher) bool {
	if publisher == nil {
		return true
	}

	value := reflect.ValueOf(publisher)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
