package handler

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/service"
	pb "vasset/asset-service/proto"
)

// GRPCServer gRPC 服务器
type GRPCServer struct {
	pb.UnimplementedAssetServiceServer
	historyService *service.HistoryService
	quotaService   *service.QuotaService
	statsService   *service.StatsService
	billingService *service.BillingService
	proxyHandler   *ProxyHandler
	cookieHandler  *CookieHandler
	cfg            *config.Config
}

// NewGRPCServer 创建 gRPC 服务器
func NewGRPCServer(
	historyService *service.HistoryService,
	quotaService *service.QuotaService,
	statsService *service.StatsService,
	billingService *service.BillingService,
	proxyHandler *ProxyHandler,
	cookieHandler *CookieHandler,
	cfg *config.Config,
) *GRPCServer {
	return &GRPCServer{
		historyService: historyService,
		quotaService:   quotaService,
		statsService:   statsService,
		billingService: billingService,
		proxyHandler:   proxyHandler,
		cookieHandler:  cookieHandler,
		cfg:            cfg,
	}
}

// GetHistory 获取下载历史
func (s *GRPCServer) GetHistory(ctx context.Context, req *pb.GetHistoryRequest) (*pb.GetHistoryResponse, error) {
	// 参数验证
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	// 构建过滤条件
	filter := &models.HistoryFilter{
		UserID:    req.UserId,
		Page:      int(req.Page),
		PageSize:  int(req.PageSize),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// 处理可选过滤条件
	if req.Status != 0 {
		status := models.HistoryStatus(req.Status)
		filter.Status = &status
	}

	if req.Platform != "" {
		filter.Platform = &req.Platform
	}

	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			filter.StartDate = &t
		}
	}

	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			// 将结束日期设为当天的23:59:59
			t = t.Add(24*time.Hour - time.Second)
			filter.EndDate = &t
		}
	}

	// 设置分页默认值
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = s.cfg.Pagination.DefaultPageSize
	}
	if filter.PageSize > s.cfg.Pagination.MaxPageSize {
		filter.PageSize = s.cfg.Pagination.MaxPageSize
	}

	// 调用服务层
	result, err := s.historyService.GetHistory(ctx, filter)
	if err != nil {
		log.Printf("GetHistory error: %v", err)
		return nil, status.Error(codes.Internal, "查询历史记录失败")
	}

	// 转换响应
	items := make([]*pb.HistoryItem, 0, len(result.Items))
	for _, h := range result.Items {
		item := &pb.HistoryItem{
			HistoryId: h.ID,
			TaskId:    h.TaskID,
			Url:       h.URL,
			Platform:  h.Platform,
			Title:     h.Title,
			Mode:      h.Mode,
			Quality:   h.Quality,
			Status:    int32(h.Status),
			CreatedAt: h.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if h.FileSize.Valid {
			item.FileSize = h.FileSize.Int64
		}
		if h.FilePath.Valid {
			item.FilePath = h.FilePath.String
		}
		if h.FileName.Valid {
			item.FileName = h.FileName.String
		}
		if h.CompletedAt != nil {
			item.CompletedAt = h.CompletedAt.Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}

	return &pb.GetHistoryResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

// DeleteHistory 删除历史记录
func (s *GRPCServer) DeleteHistory(ctx context.Context, req *pb.DeleteHistoryRequest) (*pb.DeleteHistoryResponse, error) {
	if req.HistoryId == 0 || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "历史ID和用户ID不能为空")
	}

	err := s.historyService.DeleteHistory(ctx, req.HistoryId, req.UserId)
	if err != nil {
		log.Printf("DeleteHistory error: %v", err)
		if err.Error() == "record not found" {
			return nil, status.Error(codes.NotFound, "记录不存在")
		}
		return nil, status.Error(codes.Internal, "删除记录失败")
	}

	return &pb.DeleteHistoryResponse{Success: true}, nil
}

// GetHistoryByTask 按任务查询历史记录并验证归属
func (s *GRPCServer) GetHistoryByTask(ctx context.Context, req *pb.GetHistoryByTaskRequest) (*pb.GetHistoryByTaskResponse, error) {
	if req.TaskId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "任务ID和用户ID不能为空")
	}

	record, err := s.historyService.GetHistoryByTask(ctx, req.TaskId, req.UserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "记录不存在")
		}
		log.Printf("GetHistoryByTask error: %v", err)
		return nil, status.Error(codes.Internal, "查询历史记录失败")
	}

	return &pb.GetHistoryByTaskResponse{
		HistoryId: record.ID,
		Status:    int32(record.Status),
	}, nil
}

// CreateHistory 创建历史记录
func (s *GRPCServer) CreateHistory(ctx context.Context, req *pb.CreateHistoryRequest) (*pb.CreateHistoryResponse, error) {
	log.Printf("[GRPCServer] CreateHistory called for user %s, task %s", req.UserId, req.TaskId)

	// 参数验证
	if req.UserId == "" || req.TaskId == "" || req.Url == "" {
		log.Printf("[GRPCServer] CreateHistory validation failed: missing required fields")
		return nil, status.Error(codes.InvalidArgument, "用户ID、任务ID和URL不能为空")
	}

	// 构建历史记录
	history := &models.DownloadHistory{
		UserID:    req.UserId,
		TaskID:    req.TaskId,
		URL:       req.Url,
		Platform:  req.Platform,
		Title:     req.Title,
		Mode:      req.Mode,
		Quality:   req.Quality,
		Thumbnail: req.Thumbnail,
		Duration:  req.Duration,
		Author:    req.Author,
		Status:    models.StatusPending, // 初始状态为待处理
	}

	// 调用服务层创建
	historyID, err := s.historyService.CreateHistory(ctx, history)
	if err != nil {
		log.Printf("[GRPCServer] CreateHistory error: %v", err)
		return nil, status.Error(codes.Internal, "创建历史记录失败")
	}

	log.Printf("[GRPCServer] CreateHistory success: historyID=%d", historyID)
	return &pb.CreateHistoryResponse{
		HistoryId: historyID,
	}, nil
}

// UpdateHistoryStatus 更新下载历史状态
func (s *GRPCServer) UpdateHistoryStatus(ctx context.Context, req *pb.UpdateHistoryStatusRequest) (*pb.UpdateHistoryStatusResponse, error) {
	if req.TaskId == "" {
		return nil, status.Error(codes.InvalidArgument, "任务ID不能为空")
	}

	historyStatus := models.HistoryStatus(req.Status)

	var fileInfo *models.FileInfo
	if historyStatus == models.StatusCompleted || historyStatus == models.StatusPendingCleanup {
		if req.FilePath == "" || req.FileName == "" {
			return nil, status.Error(codes.InvalidArgument, "完成状态必须提供文件信息")
		}
		fileInfo = &models.FileInfo{
			FilePath: req.FilePath,
			FileName: req.FileName,
			FileSize: req.FileSize,
			FileHash: req.FileHash,
		}
	}

	if err := s.historyService.UpdateHistoryStatus(ctx, req.TaskId, historyStatus, fileInfo, req.ErrorMessage); err != nil {
		log.Printf("UpdateHistoryStatus error: %v", err)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, status.Error(codes.NotFound, "历史记录不存在")
		case err.Error() == "file info is required for completed status", err.Error() == "unsupported history status update":
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "更新历史记录失败")
		}
	}

	return &pb.UpdateHistoryStatusResponse{Success: true}, nil
}

func (s *GRPCServer) GetBillingAccount(ctx context.Context, req *pb.GetBillingAccountRequest) (*pb.GetBillingAccountResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	account, err := s.billingService.GetBillingAccount(ctx, req.GetUserId(), req.GetAutoCreate())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "账务账户不存在")
		}
		return nil, status.Error(codes.Internal, "获取账务账户失败")
	}

	return &pb.GetBillingAccountResponse{Account: toBillingAccountSnapshotPB(account)}, nil
}

func (s *GRPCServer) ListBillingStatements(ctx context.Context, req *pb.ListBillingStatementsRequest) (*pb.ListBillingStatementsResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = s.cfg.Pagination.DefaultPageSize
	}
	if pageSize > s.cfg.Pagination.MaxPageSize {
		pageSize = s.cfg.Pagination.MaxPageSize
	}

	result, err := s.billingService.ListBillingStatements(ctx, req.GetUserId(), page, pageSize, req.GetType(), req.GetStatus())
	if err != nil {
		return nil, status.Error(codes.Internal, "查询账单失败")
	}

	items := make([]*pb.BillingStatementItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &pb.BillingStatementItem{
			StatementId:  item.StatementID,
			Type:         item.Type,
			HistoryId:    item.HistoryID,
			TrafficBytes: item.TrafficBytes,
			AmountFen:    item.AmountFen,
			Status:       item.Status,
			Remark:       item.Remark,
			CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		})
	}

	return &pb.ListBillingStatementsResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

func (s *GRPCServer) EstimateDownloadBilling(ctx context.Context, req *pb.EstimateDownloadBillingRequest) (*pb.EstimateDownloadBillingResponse, error) {
	var filesize int64
	if req.GetSelectedFormat() != nil {
		filesize = req.GetSelectedFormat().GetFilesize()
	}

	estimate, _, err := s.billingService.EstimateDownloadBilling(ctx, filesize)
	if err != nil {
		return nil, status.Error(codes.Internal, "预估下载计费失败")
	}

	return &pb.EstimateDownloadBillingResponse{
		EstimatedIngressBytes: estimate.EstimatedIngressBytes,
		EstimatedEgressBytes:  estimate.EstimatedEgressBytes,
		EstimatedTrafficBytes: estimate.EstimatedTrafficBytes,
		EstimatedCostFen:      estimate.EstimatedCostFen,
		PricingVersion:        estimate.PricingVersion,
		IsEstimated:           estimate.IsEstimated,
		EstimateReason:        estimate.EstimateReason,
	}, nil
}

func (s *GRPCServer) HoldInitialDownload(ctx context.Context, req *pb.HoldInitialDownloadRequest) (*pb.HoldInitialDownloadResponse, error) {
	estimate := &models.BillingEstimate{
		EstimatedIngressBytes: req.GetEstimatedIngressBytes(),
		EstimatedEgressBytes:  req.GetEstimatedEgressBytes(),
		EstimatedTrafficBytes: req.GetEstimatedTrafficBytes(),
		EstimatedCostFen:      req.GetEstimatedCostFen(),
		PricingVersion:        req.GetPricingVersion(),
	}

	order, hold, account, err := s.billingService.HoldInitialDownload(ctx, req.GetUserId(), req.GetHistoryId(), req.GetTaskId(), estimate)
	if err != nil {
		if errors.Is(err, service.ErrInsufficientBalance) {
			return nil, status.Error(codes.ResourceExhausted, "余额不足")
		}
		return nil, status.Error(codes.Internal, "创建首次下载预占失败")
	}

	return &pb.HoldInitialDownloadResponse{
		OrderNo:             order.OrderNo,
		HoldNo:              hold.HoldNo,
		HeldAmountFen:       hold.AmountFen,
		AvailableBalanceFen: account.AvailableBalanceFen,
		ReservedBalanceFen:  account.ReservedBalanceFen,
	}, nil
}

func (s *GRPCServer) CaptureIngressUsage(ctx context.Context, req *pb.CaptureIngressUsageRequest) (*pb.CaptureIngressUsageResponse, error) {
	order, capturedAmount, err := s.billingService.CaptureIngressUsage(ctx, req.GetTaskId(), req.GetActualIngressBytes())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "账务订单不存在")
		}
		if errors.Is(err, service.ErrInsufficientBalance) {
			return nil, status.Error(codes.ResourceExhausted, "余额不足")
		}
		return nil, status.Error(codes.Internal, "结算入流量失败")
	}

	return &pb.CaptureIngressUsageResponse{
		OrderNo:              order.OrderNo,
		CapturedAmountFen:    capturedAmount,
		RemainingReservedFen: order.HeldAmountFen - order.CapturedAmountFen - order.ReleasedAmountFen,
		ActualIngressBytes:   order.ActualIngressBytes,
		ActualTrafficBytes:   order.ActualTrafficBytes,
		OrderStatus:          order.Status,
	}, nil
}

func (s *GRPCServer) ReleaseInitialDownload(ctx context.Context, req *pb.ReleaseInitialDownloadRequest) (*pb.ReleaseInitialDownloadResponse, error) {
	order, releasedAmount, err := s.billingService.ReleaseInitialDownload(ctx, req.GetTaskId(), req.GetReason())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "账务订单不存在")
		}
		return nil, status.Error(codes.Internal, "释放首次下载预占失败")
	}

	return &pb.ReleaseInitialDownloadResponse{
		Success:           true,
		OrderNo:           order.OrderNo,
		ReleasedAmountFen: releasedAmount,
	}, nil
}

func (s *GRPCServer) PrepareFileTransferBilling(ctx context.Context, req *pb.PrepareFileTransferBillingRequest) (*pb.PrepareFileTransferBillingResponse, error) {
	order, hold, account, pricing, err := s.billingService.PrepareFileTransferBilling(ctx, req.GetUserId(), req.GetHistoryId(), req.GetFileSizeBytes())
	if err != nil {
		if errors.Is(err, service.ErrInsufficientBalance) {
			return nil, status.Error(codes.ResourceExhausted, "余额不足")
		}
		return nil, status.Error(codes.Internal, "准备文件传输计费失败")
	}

	return &pb.PrepareFileTransferBillingResponse{
		TransferId:          hold.TransferID,
		OrderNo:             order.OrderNo,
		HoldNo:              hold.HoldNo,
		Scene:               order.Scene,
		HoldAmountFen:       hold.AmountFen,
		PricingVersion:      pricing.Version,
		AvailableBalanceFen: account.AvailableBalanceFen,
		ReservedBalanceFen:  account.ReservedBalanceFen,
	}, nil
}

func (s *GRPCServer) CompleteFileTransferBilling(ctx context.Context, req *pb.CompleteFileTransferBillingRequest) (*pb.CompleteFileTransferBillingResponse, error) {
	order, capturedAmount, err := s.billingService.CompleteFileTransferBilling(ctx, req.GetTransferId(), req.GetActualEgressBytes())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "传输计费记录不存在")
		}
		if errors.Is(err, service.ErrInsufficientBalance) {
			return nil, status.Error(codes.ResourceExhausted, "余额不足")
		}
		return nil, status.Error(codes.Internal, "完成文件传输计费失败")
	}

	return &pb.CompleteFileTransferBillingResponse{
		OrderNo:                order.OrderNo,
		CapturedAmountFen:      capturedAmount,
		ActualTrafficBytes:     order.ActualTrafficBytes,
		TotalCapturedAmountFen: order.CapturedAmountFen,
		OrderStatus:            order.Status,
	}, nil
}

func (s *GRPCServer) AbortFileTransferBilling(ctx context.Context, req *pb.AbortFileTransferBillingRequest) (*pb.AbortFileTransferBillingResponse, error) {
	order, releasedAmount, err := s.billingService.AbortFileTransferBilling(ctx, req.GetTransferId(), req.GetReason())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "传输计费记录不存在")
		}
		return nil, status.Error(codes.Internal, "中止文件传输计费失败")
	}

	return &pb.AbortFileTransferBillingResponse{
		Success:           true,
		OrderNo:           order.OrderNo,
		ReleasedAmountFen: releasedAmount,
	}, nil
}

func (s *GRPCServer) ListBillingAccounts(ctx context.Context, req *pb.ListBillingAccountsRequest) (*pb.ListBillingAccountsResponse, error) {
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = s.cfg.Pagination.DefaultPageSize
	}
	if pageSize > s.cfg.Pagination.MaxPageSize {
		pageSize = s.cfg.Pagination.MaxPageSize
	}

	result, err := s.billingService.ListBillingAccounts(ctx, models.BillingAccountFilter{
		UserIDs:  req.GetUserIds(),
		Page:     page,
		PageSize: pageSize,
		Status:   req.GetStatus(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "查询账务账户列表失败")
	}

	items := make([]*pb.BillingAccountSnapshot, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toBillingAccountSnapshotPB(&item))
	}

	return &pb.ListBillingAccountsResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

func (s *GRPCServer) GetBillingAccountDetail(ctx context.Context, req *pb.GetBillingAccountDetailRequest) (*pb.GetBillingAccountDetailResponse, error) {
	account, err := s.billingService.GetBillingAccountDetail(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "查询账务账户详情失败")
	}
	return &pb.GetBillingAccountDetailResponse{Account: toBillingAccountSnapshotPB(account)}, nil
}

func (s *GRPCServer) AdjustBillingBalance(ctx context.Context, req *pb.AdjustBillingBalanceRequest) (*pb.AdjustBillingBalanceResponse, error) {
	account, entry, err := s.billingService.AdjustBillingBalance(ctx, req.GetUserId(), req.GetOperationId(), req.GetAmountFen(), req.GetRemark(), req.GetOperatorUserId())
	if err != nil {
		if errors.Is(err, service.ErrInsufficientBalance) {
			return nil, status.Error(codes.ResourceExhausted, "余额不足")
		}
		return nil, status.Error(codes.Internal, "调整账务余额失败")
	}

	return &pb.AdjustBillingBalanceResponse{
		Success: true,
		Account: toBillingAccountSnapshotPB(account),
		EntryNo: entry.EntryNo,
	}, nil
}

func (s *GRPCServer) ListBillingLedger(ctx context.Context, req *pb.ListBillingLedgerRequest) (*pb.ListBillingLedgerResponse, error) {
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = s.cfg.Pagination.DefaultPageSize
	}
	if pageSize > s.cfg.Pagination.MaxPageSize {
		pageSize = s.cfg.Pagination.MaxPageSize
	}

	result, err := s.billingService.ListBillingLedger(ctx, models.BillingLedgerFilter{
		UserID:    req.GetUserId(),
		Page:      page,
		PageSize:  pageSize,
		EntryType: req.GetEntryType(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "查询账务流水失败")
	}

	items := make([]*pb.LedgerEntryItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, &pb.LedgerEntryItem{
			EntryNo:                  item.EntryNo,
			UserId:                   item.UserID,
			OrderNo:                  item.OrderNo,
			HoldNo:                   item.HoldNo,
			HistoryId:                item.HistoryID,
			TaskId:                   item.TaskID,
			TransferId:               item.TransferID,
			OperationId:              item.OperationID,
			EntryType:                item.EntryType,
			Scene:                    item.Scene,
			ActionAmountFen:          item.ActionAmountFen,
			AvailableDeltaFen:        item.AvailableDeltaFen,
			ReservedDeltaFen:         item.ReservedDeltaFen,
			BalanceAfterAvailableFen: item.BalanceAfterAvailableFen,
			BalanceAfterReservedFen:  item.BalanceAfterReservedFen,
			OperatorUserId:           item.OperatorUserID,
			Remark:                   item.Remark,
			CreatedAt:                item.CreatedAt.Format(time.RFC3339),
		})
	}

	return &pb.ListBillingLedgerResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

func (s *GRPCServer) ListTrafficUsageRecords(ctx context.Context, req *pb.ListTrafficUsageRecordsRequest) (*pb.ListTrafficUsageRecordsResponse, error) {
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = s.cfg.Pagination.DefaultPageSize
	}
	if pageSize > s.cfg.Pagination.MaxPageSize {
		pageSize = s.cfg.Pagination.MaxPageSize
	}

	result, err := s.billingService.ListTrafficUsageRecords(ctx, models.TrafficUsageFilter{
		UserID:    req.GetUserId(),
		Page:      page,
		PageSize:  pageSize,
		Direction: req.GetDirection(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "查询流量记录失败")
	}

	items := make([]*pb.TrafficUsageRecordItem, 0, len(result.Items))
	for _, item := range result.Items {
		confirmedAt := ""
		if item.ConfirmedAt != nil {
			confirmedAt = item.ConfirmedAt.Format(time.RFC3339)
		}
		items = append(items, &pb.TrafficUsageRecordItem{
			UsageNo:            item.UsageNo,
			OrderNo:            item.OrderNo,
			UserId:             item.UserID,
			HistoryId:          item.HistoryID,
			TaskId:             item.TaskID,
			TransferId:         item.TransferID,
			Direction:          item.Direction,
			TrafficBytes:       item.TrafficBytes,
			UnitPriceFenPerGib: item.UnitPriceFenPerGiB,
			AmountFen:          item.AmountFen,
			PricingVersion:     item.PricingVersion,
			SourceService:      item.SourceService,
			Status:             item.Status,
			CreatedAt:          item.CreatedAt.Format(time.RFC3339),
			ConfirmedAt:        confirmedAt,
		})
	}

	return &pb.ListTrafficUsageRecordsResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

func (s *GRPCServer) GetBillingPricing(ctx context.Context, _ *pb.GetBillingPricingRequest) (*pb.GetBillingPricingResponse, error) {
	pricing, err := s.billingService.GetBillingPricing(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "获取费率失败")
	}
	return &pb.GetBillingPricingResponse{Pricing: toBillingPricingPB(pricing)}, nil
}

func (s *GRPCServer) UpdateBillingPricing(ctx context.Context, req *pb.UpdateBillingPricingRequest) (*pb.UpdateBillingPricingResponse, error) {
	pricing, err := s.billingService.UpdateBillingPricing(ctx, req.GetIngressPriceFenPerGib(), req.GetEgressPriceFenPerGib(), req.GetDefaultEstimateBytes(), req.GetRemark(), req.GetOperatorUserId())
	if err != nil {
		return nil, status.Error(codes.Internal, "更新费率失败")
	}
	return &pb.UpdateBillingPricingResponse{
		Success: true,
		Pricing: toBillingPricingPB(pricing),
	}, nil
}

func (s *GRPCServer) ListBillingShortfalls(ctx context.Context, req *pb.ListBillingShortfallsRequest) (*pb.ListBillingShortfallsResponse, error) {
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = s.cfg.Pagination.DefaultPageSize
	}
	if pageSize > s.cfg.Pagination.MaxPageSize {
		pageSize = s.cfg.Pagination.MaxPageSize
	}

	result, err := s.billingService.ListBillingShortfalls(ctx, models.BillingShortfallFilter{
		UserID:   req.GetUserId(),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "查询短款订单失败")
	}

	items := make([]*pb.BillingShortfallOrderItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toBillingShortfallOrderPB(item))
	}

	return &pb.ListBillingShortfallsResponse{
		Total:    result.Total,
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
		Items:    items,
	}, nil
}

func (s *GRPCServer) ReconcileBillingShortfall(ctx context.Context, req *pb.ReconcileBillingShortfallRequest) (*pb.ReconcileBillingShortfallResponse, error) {
	order, account, entry, err := s.billingService.ReconcileBillingShortfall(ctx, req.GetOrderNo(), req.GetRemark(), req.GetOperatorUserId())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.NotFound, "短款订单不存在")
		}
		if errors.Is(err, service.ErrInsufficientBalance) {
			return nil, status.Error(codes.ResourceExhausted, "余额不足")
		}
		return nil, status.Error(codes.Internal, "补扣短款订单失败")
	}

	entryNo := ""
	if entry != nil {
		entryNo = entry.EntryNo
	}

	return &pb.ReconcileBillingShortfallResponse{
		Success: true,
		Order: toBillingShortfallOrderPB(models.BillingShortfallOrder{
			OrderNo:            order.OrderNo,
			UserID:             order.UserID,
			HistoryID:          order.HistoryID,
			TaskID:             order.TaskID,
			Scene:              order.Scene,
			Status:             order.Status,
			PricingVersion:     order.PricingVersion,
			ActualIngressBytes: order.ActualIngressBytes,
			ActualEgressBytes:  order.ActualEgressBytes,
			ActualTrafficBytes: order.ActualTrafficBytes,
			HeldAmountFen:      order.HeldAmountFen,
			CapturedAmountFen:  order.CapturedAmountFen,
			ReleasedAmountFen:  order.ReleasedAmountFen,
			ShortfallFen:       order.ShortfallFen,
			Remark:             order.Remark,
			CreatedAt:          order.CreatedAt,
			UpdatedAt:          order.UpdatedAt,
		}),
		Account: toBillingAccountSnapshotPB(account),
		EntryNo: entryNo,
	}, nil
}

func toBillingAccountSnapshotPB(account *models.BillingAccount) *pb.BillingAccountSnapshot {
	return &pb.BillingAccountSnapshot{
		UserId:              account.UserID,
		CurrencyCode:        account.CurrencyCode,
		AvailableBalanceFen: account.AvailableBalanceFen,
		ReservedBalanceFen:  account.ReservedBalanceFen,
		TotalRechargedFen:   account.TotalRechargedFen,
		TotalSpentFen:       account.TotalSpentFen,
		TotalTrafficBytes:   account.TotalTrafficBytes,
		Status:              account.Status,
		Version:             account.Version,
		CreatedAt:           account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           account.UpdatedAt.Format(time.RFC3339),
	}
}

func toBillingShortfallOrderPB(order models.BillingShortfallOrder) *pb.BillingShortfallOrderItem {
	return &pb.BillingShortfallOrderItem{
		OrderNo:            order.OrderNo,
		UserId:             order.UserID,
		HistoryId:          order.HistoryID,
		TaskId:             order.TaskID,
		Scene:              order.Scene,
		Status:             order.Status,
		PricingVersion:     order.PricingVersion,
		ActualIngressBytes: order.ActualIngressBytes,
		ActualEgressBytes:  order.ActualEgressBytes,
		ActualTrafficBytes: order.ActualTrafficBytes,
		HeldAmountFen:      order.HeldAmountFen,
		CapturedAmountFen:  order.CapturedAmountFen,
		ReleasedAmountFen:  order.ReleasedAmountFen,
		ShortfallFen:       order.ShortfallFen,
		Remark:             order.Remark,
		CreatedAt:          order.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          order.UpdatedAt.Format(time.RFC3339),
	}
}

func toBillingPricingPB(pricing *models.BillingPricing) *pb.BillingPricing {
	return &pb.BillingPricing{
		Version:               pricing.Version,
		IngressPriceFenPerGib: pricing.IngressPriceFenPerGiB,
		EgressPriceFenPerGib:  pricing.EgressPriceFenPerGiB,
		DefaultEstimateBytes:  pricing.DefaultEstimateBytes,
		Enabled:               pricing.Enabled,
		Remark:                pricing.Remark,
		UpdatedByUserId:       pricing.UpdatedByUserID,
		EffectiveAt:           pricing.EffectiveAt.Format(time.RFC3339),
		CreatedAt:             pricing.CreatedAt.Format(time.RFC3339),
	}
}

// CheckQuota 检查配额
func (s *GRPCServer) CheckQuota(ctx context.Context, req *pb.CheckQuotaRequest) (*pb.CheckQuotaResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	quota, err := s.quotaService.CheckQuota(ctx, req.UserId)
	if err != nil {
		log.Printf("CheckQuota error: %v", err)
		return nil, status.Error(codes.Internal, "检查配额失败")
	}

	remaining := s.quotaService.GetRemaining(quota)

	return &pb.CheckQuotaResponse{
		DailyLimit: int32(quota.DailyLimit),
		DailyUsed:  int32(quota.DailyUsed),
		Remaining:  int32(remaining),
		ResetAt:    quota.ResetAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ConsumeQuota 消费配额
func (s *GRPCServer) ConsumeQuota(ctx context.Context, req *pb.ConsumeQuotaRequest) (*pb.ConsumeQuotaResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	quota, err := s.quotaService.ConsumeQuota(ctx, req.UserId)
	if err != nil {
		log.Printf("ConsumeQuota error: %v", err)
		if err.Error() == "daily quota exceeded" {
			return nil, status.Error(codes.ResourceExhausted, "每日配额已用完")
		}
		return nil, status.Error(codes.Internal, "消费配额失败")
	}

	remaining := s.quotaService.GetRemaining(quota)

	return &pb.ConsumeQuotaResponse{
		Success:   true,
		Remaining: int32(remaining),
	}, nil
}

// RefundQuota 退还提交失败时已消费的配额
func (s *GRPCServer) RefundQuota(ctx context.Context, req *pb.RefundQuotaRequest) (*pb.RefundQuotaResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	quota, err := s.quotaService.RefundQuota(ctx, req.UserId)
	if err != nil {
		log.Printf("RefundQuota error: %v", err)
		return nil, status.Error(codes.Internal, "退还配额失败")
	}

	return &pb.RefundQuotaResponse{
		Success:   true,
		Remaining: int32(s.quotaService.GetRemaining(quota)),
	}, nil
}

// GetUserStats 获取用户统计
func (s *GRPCServer) GetUserStats(ctx context.Context, req *pb.GetUserStatsRequest) (*pb.GetUserStatsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "用户ID不能为空")
	}

	stats, err := s.statsService.GetUserStats(ctx, req.UserId)
	if err != nil {
		log.Printf("GetUserStats error: %v", err)
		return nil, status.Error(codes.Internal, "获取统计信息失败")
	}

	// 转换平台统计
	platforms := make([]*pb.PlatformStat, 0, len(stats.TopPlatforms))
	for _, p := range stats.TopPlatforms {
		platforms = append(platforms, &pb.PlatformStat{
			Platform: p.Platform,
			Count:    p.Count,
		})
	}

	// 转换日活统计
	activities := make([]*pb.DailyActivity, 0, len(stats.RecentActivity))
	for _, a := range stats.RecentActivity {
		activities = append(activities, &pb.DailyActivity{
			Date:  a.Date,
			Count: a.Count,
		})
	}

	return &pb.GetUserStatsResponse{
		TotalDownloads:   stats.TotalDownloads,
		SuccessDownloads: stats.SuccessDownloads,
		FailedDownloads:  stats.FailedDownloads,
		TotalSizeBytes:   stats.TotalSize,
		TopPlatforms:     platforms,
		RecentActivity:   activities,
	}, nil
}

// GetPlatformStats 获取平台统计
func (s *GRPCServer) GetPlatformStats(ctx context.Context, req *pb.GetPlatformStatsRequest) (*pb.GetPlatformStatsResponse, error) {
	stats, err := s.statsService.GetPlatformStats(ctx)
	if err != nil {
		log.Printf("GetPlatformStats error: %v", err)
		return nil, status.Error(codes.Internal, "获取平台统计失败")
	}

	return &pb.GetPlatformStatsResponse{
		TotalDownloads:    stats.TotalDownloads,
		SuccessDownloads:  stats.SuccessDownloads,
		FailedDownloads:   stats.FailedDownloads,
		DownloadsToday:    stats.DownloadsToday,
		DailyActiveUsers:  stats.DailyActiveUsers,
		WeeklyActiveUsers: stats.WeeklyActiveUsers,
	}, nil
}

// GetRequestTrend 获取平台请求趋势
func (s *GRPCServer) GetRequestTrend(ctx context.Context, req *pb.GetRequestTrendRequest) (*pb.GetRequestTrendResponse, error) {
	granularity := req.Granularity
	if granularity == "" {
		granularity = "day"
	}

	points, err := s.statsService.GetRequestTrend(ctx, granularity, int(req.Limit))
	if err != nil {
		log.Printf("GetRequestTrend error: %v", err)
		return nil, status.Error(codes.Internal, "获取请求趋势失败")
	}

	respPoints := make([]*pb.TrendPoint, 0, len(points))
	for _, point := range points {
		respPoints = append(respPoints, &pb.TrendPoint{
			Label: point.Label,
			Count: point.Count,
		})
	}

	return &pb.GetRequestTrendResponse{
		Granularity: granularity,
		Points:      respPoints,
	}, nil
}

// GetFileInfo 获取文件信息
func (s *GRPCServer) GetFileInfo(ctx context.Context, req *pb.GetFileInfoRequest) (*pb.GetFileInfoResponse, error) {
	if req.HistoryId == 0 || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "历史ID和用户ID不能为空")
	}

	fileInfo, err := s.historyService.GetFileInfo(ctx, req.HistoryId, req.UserId)
	if err != nil {
		log.Printf("GetFileInfo error: %v", err)
		switch err.Error() {
		case "access denied":
			return nil, status.Error(codes.PermissionDenied, "无权访问该文件")
		case "download not completed":
			return nil, status.Error(codes.FailedPrecondition, "下载尚未完成")
		case "file not found":
			return nil, status.Error(codes.NotFound, "文件不存在")
		default:
			return nil, status.Error(codes.Internal, "获取文件信息失败")
		}
	}

	return &pb.GetFileInfoResponse{
		FilePath: fileInfo.FilePath,
		FileName: fileInfo.FileName,
		FileSize: fileInfo.FileSize,
		FileHash: fileInfo.FileHash,
	}, nil
}

// ========== 代理管理 ==========

func (s *GRPCServer) AcquireProxyForTask(ctx context.Context, req *pb.AcquireProxyForTaskRequest) (*pb.AcquireProxyForTaskResponse, error) {
	return s.proxyHandler.AcquireProxyForTask(ctx, req)
}

func (s *GRPCServer) GetAvailableProxy(ctx context.Context, req *pb.GetAvailableProxyRequest) (*pb.GetAvailableProxyResponse, error) {
	return s.proxyHandler.GetAvailableProxy(ctx, req)
}

func (s *GRPCServer) ReportProxyUsage(ctx context.Context, req *pb.ReportProxyUsageRequest) (*pb.ReportProxyUsageResponse, error) {
	return s.proxyHandler.ReportProxyUsage(ctx, req)
}

func (s *GRPCServer) GetProxySourcePolicy(ctx context.Context, req *pb.GetProxySourcePolicyRequest) (*pb.GetProxySourcePolicyResponse, error) {
	return s.proxyHandler.GetProxySourcePolicy(ctx, req)
}

func (s *GRPCServer) UpdateProxySourcePolicy(ctx context.Context, req *pb.UpdateProxySourcePolicyRequest) (*pb.UpdateProxySourcePolicyResponse, error) {
	return s.proxyHandler.UpdateProxySourcePolicy(ctx, req)
}

func (s *GRPCServer) ListProxies(ctx context.Context, req *pb.ListProxiesRequest) (*pb.ListProxiesResponse, error) {
	return s.proxyHandler.ListProxies(ctx, req)
}

func (s *GRPCServer) CreateProxy(ctx context.Context, req *pb.CreateProxyRequest) (*pb.CreateProxyResponse, error) {
	return s.proxyHandler.CreateProxy(ctx, req)
}

func (s *GRPCServer) UpdateProxy(ctx context.Context, req *pb.UpdateProxyRequest) (*pb.UpdateProxyResponse, error) {
	return s.proxyHandler.UpdateProxy(ctx, req)
}

func (s *GRPCServer) UpdateProxyStatus(ctx context.Context, req *pb.UpdateProxyStatusRequest) (*pb.UpdateProxyStatusResponse, error) {
	return s.proxyHandler.UpdateProxyStatus(ctx, req)
}

func (s *GRPCServer) DeleteProxy(ctx context.Context, req *pb.DeleteProxyRequest) (*pb.DeleteProxyResponse, error) {
	return s.proxyHandler.DeleteProxy(ctx, req)
}

// ========== Cookie 管理 ==========

func (s *GRPCServer) CreateCookie(ctx context.Context, req *pb.CreateCookieRequest) (*pb.CreateCookieResponse, error) {
	return s.cookieHandler.CreateCookie(ctx, req)
}

func (s *GRPCServer) UpdateCookie(ctx context.Context, req *pb.UpdateCookieRequest) (*pb.UpdateCookieResponse, error) {
	return s.cookieHandler.UpdateCookie(ctx, req)
}

func (s *GRPCServer) DeleteCookie(ctx context.Context, req *pb.DeleteCookieRequest) (*pb.DeleteCookieResponse, error) {
	return s.cookieHandler.DeleteCookie(ctx, req)
}

func (s *GRPCServer) GetCookie(ctx context.Context, req *pb.GetCookieRequest) (*pb.GetCookieResponse, error) {
	return s.cookieHandler.GetCookie(ctx, req)
}

func (s *GRPCServer) ListCookies(ctx context.Context, req *pb.ListCookiesRequest) (*pb.ListCookiesResponse, error) {
	return s.cookieHandler.ListCookies(ctx, req)
}

func (s *GRPCServer) GetAvailableCookie(ctx context.Context, req *pb.GetAvailableCookieRequest) (*pb.GetAvailableCookieResponse, error) {
	return s.cookieHandler.GetAvailableCookie(ctx, req)
}

func (s *GRPCServer) ReportCookieUsage(ctx context.Context, req *pb.ReportCookieUsageRequest) (*pb.ReportCookieUsageResponse, error) {
	return s.cookieHandler.ReportCookieUsage(ctx, req)
}

func (s *GRPCServer) FreezeCookie(ctx context.Context, req *pb.FreezeCookieRequest) (*pb.FreezeCookieResponse, error) {
	return s.cookieHandler.FreezeCookie(ctx, req)
}
