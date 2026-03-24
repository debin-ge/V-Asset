package handler

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"youdlp/api-gateway/internal/models"
	pb "youdlp/api-gateway/proto"
)

type AdminBillingHandler struct {
	adminClient pb.AdminServiceClient
	timeout     time.Duration
}

func NewAdminBillingHandler(adminClient pb.AdminServiceClient, timeout time.Duration) *AdminBillingHandler {
	return &AdminBillingHandler{adminClient: adminClient, timeout: timeout}
}

func (h *AdminBillingHandler) ListAccounts(c *gin.Context) {
	var req models.AdminBillingListRequest
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

	resp, err := h.adminClient.ListBillingAccounts(ctx, &pb.AdminListBillingAccountsRequest{
		Query:    req.Query,
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
		Status:   req.Status,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]models.AdminBillingAccount, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, models.AdminBillingAccount{
			UserID:               item.GetUserId(),
			Email:                item.GetEmail(),
			Nickname:             item.GetNickname(),
			AvailableBalanceYuan: item.GetAvailableBalanceYuan(),
			ReservedBalanceYuan:  item.GetReservedBalanceYuan(),
			TotalRechargedYuan:   item.GetTotalRechargedYuan(),
			TotalSpentYuan:       item.GetTotalSpentYuan(),
			TotalTrafficBytes:    item.GetTotalTrafficBytes(),
			Status:               item.GetStatus(),
			Version:              item.GetVersion(),
			UpdatedAt:            item.GetUpdatedAt(),
		})
	}

	models.Success(c, models.PagedResponse{
		Total:    resp.GetTotal(),
		Page:     int(resp.GetPage()),
		PageSize: int(resp.GetPageSize()),
		Items:    items,
	})
}

func (h *AdminBillingHandler) GetAccountDetail(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		models.BadRequest(c, "userId is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetBillingAccountDetail(ctx, &pb.AdminGetBillingAccountDetailRequest{UserId: userID})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	account := resp.GetAccount()
	models.Success(c, models.AdminBillingAccount{
		UserID:               account.GetUserId(),
		Email:                account.GetEmail(),
		Nickname:             account.GetNickname(),
		AvailableBalanceYuan: account.GetAvailableBalanceYuan(),
		ReservedBalanceYuan:  account.GetReservedBalanceYuan(),
		TotalRechargedYuan:   account.GetTotalRechargedYuan(),
		TotalSpentYuan:       account.GetTotalSpentYuan(),
		TotalTrafficBytes:    account.GetTotalTrafficBytes(),
		Status:               account.GetStatus(),
		Version:              account.GetVersion(),
		UpdatedAt:            account.GetUpdatedAt(),
	})
}

func (h *AdminBillingHandler) AdjustBalance(c *gin.Context) {
	userID := c.Param("userId")
	if userID == "" {
		models.BadRequest(c, "userId is required")
		return
	}

	var req models.AdminAdjustBillingBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	adminUser, ok := getAdminUserFromContext(c)
	if !ok {
		models.Unauthorized(c, "invalid admin user")
		return
	}

	operationID := req.OperationID
	if operationID == "" {
		operationID = uuid.NewString()
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.AdjustBillingBalance(ctx, &pb.AdminAdjustBillingBalanceRequest{
		UserId:         userID,
		OperationId:    operationID,
		AmountYuan:     req.AmountYuan,
		Remark:         req.Remark,
		OperatorUserId: adminUser.GetUserId(),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	account := resp.GetAccount()
	models.Success(c, gin.H{
		"success":  resp.GetSuccess(),
		"entry_no": resp.GetEntryNo(),
		"account": models.AdminBillingAccount{
			UserID:               account.GetUserId(),
			Email:                account.GetEmail(),
			Nickname:             account.GetNickname(),
			AvailableBalanceYuan: account.GetAvailableBalanceYuan(),
			ReservedBalanceYuan:  account.GetReservedBalanceYuan(),
			TotalRechargedYuan:   account.GetTotalRechargedYuan(),
			TotalSpentYuan:       account.GetTotalSpentYuan(),
			TotalTrafficBytes:    account.GetTotalTrafficBytes(),
			Status:               account.GetStatus(),
			Version:              account.GetVersion(),
			UpdatedAt:            account.GetUpdatedAt(),
		},
	})
}

func (h *AdminBillingHandler) ListShortfalls(c *gin.Context) {
	var req models.AdminBillingShortfallRequest
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

	resp, err := h.adminClient.ListBillingShortfalls(ctx, &pb.AdminListBillingShortfallsRequest{
		UserId:   req.UserID,
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]models.AdminBillingShortfallOrder, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, models.AdminBillingShortfallOrder{
			OrderNo:            item.GetOrderNo(),
			UserID:             item.GetUserId(),
			Email:              item.GetEmail(),
			Nickname:           item.GetNickname(),
			HistoryID:          item.GetHistoryId(),
			TaskID:             item.GetTaskId(),
			Scene:              item.GetScene(),
			Status:             item.GetStatus(),
			PricingVersion:     item.GetPricingVersion(),
			ActualIngressBytes: item.GetActualIngressBytes(),
			ActualEgressBytes:  item.GetActualEgressBytes(),
			ActualTrafficBytes: item.GetActualTrafficBytes(),
			HeldAmountYuan:     item.GetHeldAmountYuan(),
			CapturedAmountYuan: item.GetCapturedAmountYuan(),
			ReleasedAmountYuan: item.GetReleasedAmountYuan(),
			ShortfallYuan:      item.GetShortfallYuan(),
			Remark:             item.GetRemark(),
			CreatedAt:          item.GetCreatedAt(),
			UpdatedAt:          item.GetUpdatedAt(),
		})
	}

	models.Success(c, models.PagedResponse{
		Total:    resp.GetTotal(),
		Page:     int(resp.GetPage()),
		PageSize: int(resp.GetPageSize()),
		Items:    items,
	})
}

func (h *AdminBillingHandler) ReconcileShortfall(c *gin.Context) {
	orderNo := c.Param("orderNo")
	if orderNo == "" {
		models.BadRequest(c, "orderNo is required")
		return
	}

	var req models.AdminReconcileBillingShortfallRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	adminUser, ok := getAdminUserFromContext(c)
	if !ok {
		models.Unauthorized(c, "invalid admin user")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.ReconcileBillingShortfall(ctx, &pb.AdminReconcileBillingShortfallRequest{
		OrderNo:        orderNo,
		OperatorUserId: adminUser.GetUserId(),
		Remark:         req.Remark,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	order := resp.GetOrder()
	account := resp.GetAccount()
	models.Success(c, gin.H{
		"success":  resp.GetSuccess(),
		"entry_no": resp.GetEntryNo(),
		"order": models.AdminBillingShortfallOrder{
			OrderNo:            order.GetOrderNo(),
			UserID:             order.GetUserId(),
			Email:              order.GetEmail(),
			Nickname:           order.GetNickname(),
			HistoryID:          order.GetHistoryId(),
			TaskID:             order.GetTaskId(),
			Scene:              order.GetScene(),
			Status:             order.GetStatus(),
			PricingVersion:     order.GetPricingVersion(),
			ActualIngressBytes: order.GetActualIngressBytes(),
			ActualEgressBytes:  order.GetActualEgressBytes(),
			ActualTrafficBytes: order.GetActualTrafficBytes(),
			HeldAmountYuan:     order.GetHeldAmountYuan(),
			CapturedAmountYuan: order.GetCapturedAmountYuan(),
			ReleasedAmountYuan: order.GetReleasedAmountYuan(),
			ShortfallYuan:      order.GetShortfallYuan(),
			Remark:             order.GetRemark(),
			CreatedAt:          order.GetCreatedAt(),
			UpdatedAt:          order.GetUpdatedAt(),
		},
		"account": models.AdminBillingAccount{
			UserID:               account.GetUserId(),
			Email:                account.GetEmail(),
			Nickname:             account.GetNickname(),
			AvailableBalanceYuan: account.GetAvailableBalanceYuan(),
			ReservedBalanceYuan:  account.GetReservedBalanceYuan(),
			TotalRechargedYuan:   account.GetTotalRechargedYuan(),
			TotalSpentYuan:       account.GetTotalSpentYuan(),
			TotalTrafficBytes:    account.GetTotalTrafficBytes(),
			Status:               account.GetStatus(),
			Version:              account.GetVersion(),
			UpdatedAt:            account.GetUpdatedAt(),
		},
	})
}

func (h *AdminBillingHandler) ListLedger(c *gin.Context) {
	var req models.AdminBillingLedgerRequest
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

	resp, err := h.adminClient.ListBillingLedger(ctx, &pb.AdminListBillingLedgerRequest{
		UserId:    req.UserID,
		Page:      int32(req.Page),
		PageSize:  int32(req.PageSize),
		EntryType: req.EntryType,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]models.AdminBillingLedgerEntry, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, models.AdminBillingLedgerEntry{
			EntryNo:                   item.GetEntryNo(),
			UserID:                    item.GetUserId(),
			Email:                     item.GetEmail(),
			Nickname:                  item.GetNickname(),
			OrderNo:                   item.GetOrderNo(),
			HoldNo:                    item.GetHoldNo(),
			HistoryID:                 item.GetHistoryId(),
			TaskID:                    item.GetTaskId(),
			TransferID:                item.GetTransferId(),
			OperationID:               item.GetOperationId(),
			EntryType:                 item.GetEntryType(),
			Scene:                     item.GetScene(),
			ActionAmountYuan:          item.GetActionAmountYuan(),
			AvailableDeltaYuan:        item.GetAvailableDeltaYuan(),
			ReservedDeltaYuan:         item.GetReservedDeltaYuan(),
			BalanceAfterAvailableYuan: item.GetBalanceAfterAvailableYuan(),
			BalanceAfterReservedYuan:  item.GetBalanceAfterReservedYuan(),
			OperatorUserID:            item.GetOperatorUserId(),
			Remark:                    item.GetRemark(),
			CreatedAt:                 item.GetCreatedAt(),
		})
	}

	models.Success(c, models.PagedResponse{
		Total:    resp.GetTotal(),
		Page:     int(resp.GetPage()),
		PageSize: int(resp.GetPageSize()),
		Items:    items,
	})
}

func (h *AdminBillingHandler) ListUsageRecords(c *gin.Context) {
	var req models.AdminBillingUsageRequest
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

	resp, err := h.adminClient.ListBillingUsageRecords(ctx, &pb.AdminListBillingUsageRecordsRequest{
		UserId:    req.UserID,
		Page:      int32(req.Page),
		PageSize:  int32(req.PageSize),
		Direction: req.Direction,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	items := make([]models.AdminBillingUsageRecord, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, models.AdminBillingUsageRecord{
			UsageNo:            item.GetUsageNo(),
			OrderNo:            item.GetOrderNo(),
			UserID:             item.GetUserId(),
			Email:              item.GetEmail(),
			Nickname:           item.GetNickname(),
			HistoryID:          item.GetHistoryId(),
			TaskID:             item.GetTaskId(),
			TransferID:         item.GetTransferId(),
			Direction:          item.GetDirection(),
			TrafficBytes:       item.GetTrafficBytes(),
			UnitPriceYuanPerGB: item.GetUnitPriceYuanPerGb(),
			AmountYuan:         item.GetAmountYuan(),
			PricingVersion:     item.GetPricingVersion(),
			SourceService:      item.GetSourceService(),
			Status:             item.GetStatus(),
			CreatedAt:          item.GetCreatedAt(),
			ConfirmedAt:        item.GetConfirmedAt(),
		})
	}

	models.Success(c, models.PagedResponse{
		Total:    resp.GetTotal(),
		Page:     int(resp.GetPage()),
		PageSize: int(resp.GetPageSize()),
		Items:    items,
	})
}

func (h *AdminBillingHandler) GetPricing(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetBillingPricing(ctx, &pb.AdminEmpty{})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, pricingFromAdminProto(resp))
}

func (h *AdminBillingHandler) UpdatePricing(c *gin.Context) {
	var req models.AdminUpdateBillingPricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	adminUser, ok := getAdminUserFromContext(c)
	if !ok {
		models.Unauthorized(c, "invalid admin user")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.UpdateBillingPricing(ctx, &pb.AdminUpdateBillingPricingRequest{
		IngressPriceYuanPerGb: req.IngressPriceYuanPerGB,
		EgressPriceYuanPerGb:  req.EgressPriceYuanPerGB,
		Remark:                req.Remark,
		OperatorUserId:        adminUser.GetUserId(),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, pricingFromAdminProto(resp))
}

func (h *AdminBillingHandler) GetWelcomeCreditSettings(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.GetWelcomeCreditSettings(ctx, &pb.AdminEmpty{})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, welcomeCreditSettingsFromAdminProto(resp))
}

func (h *AdminBillingHandler) UpdateWelcomeCreditSettings(c *gin.Context) {
	var req models.AdminUpdateWelcomeCreditSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	adminUser, ok := getAdminUserFromContext(c)
	if !ok {
		models.Unauthorized(c, "invalid admin user")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	resp, err := h.adminClient.UpdateWelcomeCreditSettings(ctx, &pb.AdminUpdateWelcomeCreditSettingsRequest{
		Enabled:        req.Enabled,
		AmountYuan:     req.AmountYuan,
		CurrencyCode:   req.CurrencyCode,
		OperatorUserId: adminUser.GetUserId(),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	models.Success(c, welcomeCreditSettingsFromAdminProto(resp))
}

func getAdminUserFromContext(c *gin.Context) (*pb.AdminUser, bool) {
	user, exists := c.Get("admin_user")
	if !exists {
		return nil, false
	}
	adminUser, ok := user.(*pb.AdminUser)
	return adminUser, ok
}

func pricingFromAdminProto(pricing *pb.AdminBillingPricingResponse) models.AdminBillingPricing {
	if pricing == nil {
		return models.AdminBillingPricing{}
	}
	return models.AdminBillingPricing{
		Version:               pricing.GetVersion(),
		IngressPriceYuanPerGB: pricing.GetIngressPriceYuanPerGb(),
		EgressPriceYuanPerGB:  pricing.GetEgressPriceYuanPerGb(),
		Enabled:               pricing.GetEnabled(),
		Remark:                pricing.GetRemark(),
		UpdatedByUserID:       pricing.GetUpdatedByUserId(),
		EffectiveAt:           pricing.GetEffectiveAt(),
		CreatedAt:             pricing.GetCreatedAt(),
	}
}

func welcomeCreditSettingsFromAdminProto(settings *pb.AdminWelcomeCreditSettingsResponse) models.AdminWelcomeCreditSettings {
	if settings == nil {
		return models.AdminWelcomeCreditSettings{}
	}
	return models.AdminWelcomeCreditSettings{
		Enabled:      settings.GetEnabled(),
		AmountYuan:   settings.GetAmountYuan(),
		CurrencyCode: settings.GetCurrencyCode(),
		UpdatedAt:    settings.GetUpdatedAt(),
		UpdatedBy:    settings.GetUpdatedBy(),
	}
}
