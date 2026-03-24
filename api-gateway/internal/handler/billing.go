package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"youdlp/api-gateway/internal/middleware"
	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

type BillingHandler struct {
	assetClient pb.AssetServiceClient
	timeout     time.Duration
}

func NewBillingHandler(assetClient pb.AssetServiceClient, timeout time.Duration) *BillingHandler {
	return &BillingHandler{
		assetClient: assetClient,
		timeout:     timeout,
	}
}

func (h *BillingHandler) GetAccount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.GetBillingAccount(ctx, &pb.GetBillingAccountRequest{
		UserId:     userID,
		AutoCreate: true,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	account := resp.GetAccount()
	models.Success(c, models.BillingAccountOverviewResponse{
		UserID:               account.GetUserId(),
		CurrencyCode:         account.GetCurrencyCode(),
		AvailableBalanceYuan: account.GetAvailableBalanceYuan(),
		ReservedBalanceYuan:  account.GetReservedBalanceYuan(),
		TotalRechargedYuan:   account.GetTotalRechargedYuan(),
		TotalSpentYuan:       account.GetTotalSpentYuan(),
		TotalTrafficBytes:    account.GetTotalTrafficBytes(),
		Status:               account.GetStatus(),
		Version:              account.GetVersion(),
		CreatedAt:            account.GetCreatedAt(),
		UpdatedAt:            account.GetUpdatedAt(),
	})
}

func (h *BillingHandler) ListStatements(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	var req models.BillingStatementRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.assetClient.ListBillingStatements(ctx, &pb.ListBillingStatementsRequest{
		UserId:   userID,
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
		Type:     req.Type,
		Status:   req.Status,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]models.BillingStatementItem, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, models.BillingStatementItem{
			StatementID:  item.GetStatementId(),
			Type:         item.GetType(),
			HistoryID:    item.GetHistoryId(),
			TrafficBytes: item.GetTrafficBytes(),
			AmountYuan:   item.GetAmountYuan(),
			Status:       item.GetStatus(),
			Remark:       item.GetRemark(),
			CreatedAt:    item.GetCreatedAt(),
		})
	}

	models.Success(c, models.BillingStatementListResponse{
		Total:    resp.GetTotal(),
		Page:     int(resp.GetPage()),
		PageSize: int(resp.GetPageSize()),
		Items:    items,
	})
}

func (h *BillingHandler) Estimate(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		models.Unauthorized(c, "user not authenticated")
		return
	}

	var req models.BillingEstimateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	var selectedFormat *pb.BillingSelectedFormat
	if req.SelectedFormat != nil {
		selectedFormat = &pb.BillingSelectedFormat{
			FormatId:   req.SelectedFormat.FormatID,
			Quality:    req.SelectedFormat.Quality,
			Extension:  req.SelectedFormat.Extension,
			Filesize:   req.SelectedFormat.Filesize,
			Height:     req.SelectedFormat.Height,
			Width:      req.SelectedFormat.Width,
			Fps:        req.SelectedFormat.FPS,
			VideoCodec: req.SelectedFormat.VideoCodec,
			AudioCodec: req.SelectedFormat.AudioCodec,
			Vbr:        req.SelectedFormat.VBR,
			Abr:        req.SelectedFormat.ABR,
			Asr:        req.SelectedFormat.ASR,
		}
	}

	resp, err := h.assetClient.EstimateDownloadBilling(ctx, &pb.EstimateDownloadBillingRequest{
		UserId:         userID,
		Url:            req.URL,
		Platform:       req.Platform,
		Mode:           req.Mode,
		SelectedFormat: selectedFormat,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, models.BillingEstimateResponse{
		EstimatedTrafficBytes: resp.GetEstimatedTrafficBytes(),
		EstimatedCostYuan:     resp.GetEstimatedCostYuan(),
		PricingVersion:        resp.GetPricingVersion(),
		IsEstimated:           resp.GetIsEstimated(),
		EstimateReason:        resp.GetEstimateReason(),
	})
}
