package service

import (
	"context"

	"google.golang.org/grpc"

	"youdlp/admin-service/internal/models"
	pb "youdlp/admin-service/proto"
)

type BillingService struct {
	authClient  billingAuthClient
	assetClient billingAssetClient
}

type billingAuthClient interface {
	SearchUsers(ctx context.Context, in *pb.SearchUsersRequest, opts ...grpc.CallOption) (*pb.SearchUsersResponse, error)
	GetUserInfo(ctx context.Context, in *pb.GetUserInfoRequest, opts ...grpc.CallOption) (*pb.GetUserInfoResponse, error)
	BatchGetUsers(ctx context.Context, in *pb.BatchGetUsersRequest, opts ...grpc.CallOption) (*pb.BatchGetUsersResponse, error)
}

type billingAssetClient interface {
	ListBillingAccounts(ctx context.Context, in *pb.ListBillingAccountsRequest, opts ...grpc.CallOption) (*pb.ListBillingAccountsResponse, error)
	GetBillingAccountDetail(ctx context.Context, in *pb.GetBillingAccountDetailRequest, opts ...grpc.CallOption) (*pb.GetBillingAccountDetailResponse, error)
	AdjustBillingBalance(ctx context.Context, in *pb.AdjustBillingBalanceRequest, opts ...grpc.CallOption) (*pb.AdjustBillingBalanceResponse, error)
	ListBillingShortfalls(ctx context.Context, in *pb.ListBillingShortfallsRequest, opts ...grpc.CallOption) (*pb.ListBillingShortfallsResponse, error)
	ReconcileBillingShortfall(ctx context.Context, in *pb.ReconcileBillingShortfallRequest, opts ...grpc.CallOption) (*pb.ReconcileBillingShortfallResponse, error)
	ListBillingLedger(ctx context.Context, in *pb.ListBillingLedgerRequest, opts ...grpc.CallOption) (*pb.ListBillingLedgerResponse, error)
	ListTrafficUsageRecords(ctx context.Context, in *pb.ListTrafficUsageRecordsRequest, opts ...grpc.CallOption) (*pb.ListTrafficUsageRecordsResponse, error)
	GetBillingPricing(ctx context.Context, in *pb.GetBillingPricingRequest, opts ...grpc.CallOption) (*pb.GetBillingPricingResponse, error)
	UpdateBillingPricing(ctx context.Context, in *pb.UpdateBillingPricingRequest, opts ...grpc.CallOption) (*pb.UpdateBillingPricingResponse, error)
	GetWelcomeCreditSettings(ctx context.Context, in *pb.GetWelcomeCreditSettingsRequest, opts ...grpc.CallOption) (*pb.GetWelcomeCreditSettingsResponse, error)
	UpdateWelcomeCreditSettings(ctx context.Context, in *pb.UpdateWelcomeCreditSettingsRequest, opts ...grpc.CallOption) (*pb.UpdateWelcomeCreditSettingsResponse, error)
}

func NewBillingService(authClient billingAuthClient, assetClient billingAssetClient) *BillingService {
	return &BillingService{
		authClient:  authClient,
		assetClient: assetClient,
	}
}

func (s *BillingService) ListAccounts(ctx context.Context, query string, page, pageSize, status int32) (*models.BillingAccountListResponse, error) {
	usersResp, err := s.authClient.SearchUsers(ctx, &pb.SearchUsersRequest{
		Query:    query,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]*pb.User, len(usersResp.GetUsers()))
	userIDs := make([]string, 0, len(usersResp.GetUsers()))
	for _, user := range usersResp.GetUsers() {
		userMap[user.GetUserId()] = user
		userIDs = append(userIDs, user.GetUserId())
	}

	if len(userIDs) == 0 {
		return &models.BillingAccountListResponse{
			Total:    usersResp.GetTotal(),
			Page:     page,
			PageSize: pageSize,
			Items:    []models.BillingAccount{},
		}, nil
	}

	accountsResp, err := s.assetClient.ListBillingAccounts(ctx, &pb.ListBillingAccountsRequest{
		UserIds:  userIDs,
		Page:     1,
		PageSize: int32(len(userIDs)),
		Status:   status,
	})
	if err != nil {
		return nil, err
	}

	accountMap := make(map[string]*pb.BillingAccountSnapshot, len(accountsResp.GetItems()))
	for _, account := range accountsResp.GetItems() {
		accountMap[account.GetUserId()] = account
	}

	items := make([]models.BillingAccount, 0, len(accountsResp.GetItems()))
	for _, userID := range userIDs {
		account, ok := accountMap[userID]
		if !ok {
			continue
		}
		user := userMap[userID]
		items = append(items, models.BillingAccount{
			UserID:               account.GetUserId(),
			Email:                safeUserEmail(user),
			Nickname:             safeUserNickname(user),
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

	return &models.BillingAccountListResponse{
		Total:    usersResp.GetTotal(),
		Page:     page,
		PageSize: pageSize,
		Items:    items,
	}, nil
}

func (s *BillingService) GetAccountDetail(ctx context.Context, userID string) (*models.BillingAccount, error) {
	accountResp, err := s.assetClient.GetBillingAccountDetail(ctx, &pb.GetBillingAccountDetailRequest{UserId: userID})
	if err != nil {
		return nil, err
	}

	userResp, err := s.authClient.GetUserInfo(ctx, &pb.GetUserInfoRequest{UserId: userID})
	if err != nil {
		return nil, err
	}

	account := accountResp.GetAccount()
	user := userResp.GetUser()
	return &models.BillingAccount{
		UserID:               account.GetUserId(),
		Email:                user.GetEmail(),
		Nickname:             user.GetNickname(),
		AvailableBalanceYuan: account.GetAvailableBalanceYuan(),
		ReservedBalanceYuan:  account.GetReservedBalanceYuan(),
		TotalRechargedYuan:   account.GetTotalRechargedYuan(),
		TotalSpentYuan:       account.GetTotalSpentYuan(),
		TotalTrafficBytes:    account.GetTotalTrafficBytes(),
		Status:               account.GetStatus(),
		Version:              account.GetVersion(),
		UpdatedAt:            account.GetUpdatedAt(),
	}, nil
}

func (s *BillingService) AdjustBalance(ctx context.Context, userID, operationID, amountYuan, remark, operatorUserID string) (*models.BillingAccount, string, error) {
	resp, err := s.assetClient.AdjustBillingBalance(ctx, &pb.AdjustBillingBalanceRequest{
		UserId:         userID,
		OperationId:    operationID,
		AmountYuan:     amountYuan,
		Remark:         remark,
		OperatorUserId: operatorUserID,
	})
	if err != nil {
		return nil, "", err
	}

	userResp, err := s.authClient.GetUserInfo(ctx, &pb.GetUserInfoRequest{UserId: userID})
	if err != nil {
		return nil, "", err
	}

	account := resp.GetAccount()
	return &models.BillingAccount{
		UserID:               account.GetUserId(),
		Email:                userResp.GetUser().GetEmail(),
		Nickname:             userResp.GetUser().GetNickname(),
		AvailableBalanceYuan: account.GetAvailableBalanceYuan(),
		ReservedBalanceYuan:  account.GetReservedBalanceYuan(),
		TotalRechargedYuan:   account.GetTotalRechargedYuan(),
		TotalSpentYuan:       account.GetTotalSpentYuan(),
		TotalTrafficBytes:    account.GetTotalTrafficBytes(),
		Status:               account.GetStatus(),
		Version:              account.GetVersion(),
		UpdatedAt:            account.GetUpdatedAt(),
	}, resp.GetEntryNo(), nil
}

func (s *BillingService) ListShortfalls(ctx context.Context, userID string, page, pageSize int32) (*models.BillingShortfallListResponse, error) {
	resp, err := s.assetClient.ListBillingShortfalls(ctx, &pb.ListBillingShortfallsRequest{
		UserId:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}

	userMap, _ := s.loadUsersByIDs(ctx, extractUserIDsFromShortfalls(resp.GetItems()))
	items := make([]models.BillingShortfallOrder, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		user := userMap[item.GetUserId()]
		items = append(items, models.BillingShortfallOrder{
			OrderNo:            item.GetOrderNo(),
			UserID:             item.GetUserId(),
			Email:              safeUserEmail(user),
			Nickname:           safeUserNickname(user),
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

	return &models.BillingShortfallListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		Items:    items,
	}, nil
}

func (s *BillingService) ReconcileShortfall(ctx context.Context, orderNo, remark, operatorUserID string) (*models.BillingShortfallOrder, *models.BillingAccount, string, error) {
	resp, err := s.assetClient.ReconcileBillingShortfall(ctx, &pb.ReconcileBillingShortfallRequest{
		OrderNo:        orderNo,
		OperatorUserId: operatorUserID,
		Remark:         remark,
	})
	if err != nil {
		return nil, nil, "", err
	}

	order := resp.GetOrder()
	account := resp.GetAccount()

	userMap, _ := s.loadUsersByIDs(ctx, []string{order.GetUserId()})
	user := userMap[order.GetUserId()]

	return &models.BillingShortfallOrder{
			OrderNo:            order.GetOrderNo(),
			UserID:             order.GetUserId(),
			Email:              safeUserEmail(user),
			Nickname:           safeUserNickname(user),
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
		}, &models.BillingAccount{
			UserID:               account.GetUserId(),
			Email:                safeUserEmail(user),
			Nickname:             safeUserNickname(user),
			AvailableBalanceYuan: account.GetAvailableBalanceYuan(),
			ReservedBalanceYuan:  account.GetReservedBalanceYuan(),
			TotalRechargedYuan:   account.GetTotalRechargedYuan(),
			TotalSpentYuan:       account.GetTotalSpentYuan(),
			TotalTrafficBytes:    account.GetTotalTrafficBytes(),
			Status:               account.GetStatus(),
			Version:              account.GetVersion(),
			UpdatedAt:            account.GetUpdatedAt(),
		}, resp.GetEntryNo(), nil
}

func (s *BillingService) ListLedger(ctx context.Context, userID string, page, pageSize, entryType int32) (*models.BillingLedgerListResponse, error) {
	resp, err := s.assetClient.ListBillingLedger(ctx, &pb.ListBillingLedgerRequest{
		UserId:    userID,
		Page:      page,
		PageSize:  pageSize,
		EntryType: entryType,
	})
	if err != nil {
		return nil, err
	}

	userMap, _ := s.loadUsersByIDs(ctx, extractUserIDsFromLedger(resp.GetItems()))
	items := make([]models.BillingLedgerEntry, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		user := userMap[item.GetUserId()]
		items = append(items, models.BillingLedgerEntry{
			EntryNo:                   item.GetEntryNo(),
			UserID:                    item.GetUserId(),
			Email:                     safeUserEmail(user),
			Nickname:                  safeUserNickname(user),
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

	return &models.BillingLedgerListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		Items:    items,
	}, nil
}

func (s *BillingService) ListUsageRecords(ctx context.Context, userID string, page, pageSize, direction int32) (*models.BillingUsageListResponse, error) {
	resp, err := s.assetClient.ListTrafficUsageRecords(ctx, &pb.ListTrafficUsageRecordsRequest{
		UserId:    userID,
		Page:      page,
		PageSize:  pageSize,
		Direction: direction,
	})
	if err != nil {
		return nil, err
	}

	userMap, _ := s.loadUsersByIDs(ctx, extractUserIDsFromUsage(resp.GetItems()))
	items := make([]models.BillingUsageRecord, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		user := userMap[item.GetUserId()]
		items = append(items, models.BillingUsageRecord{
			UsageNo:            item.GetUsageNo(),
			OrderNo:            item.GetOrderNo(),
			UserID:             item.GetUserId(),
			Email:              safeUserEmail(user),
			Nickname:           safeUserNickname(user),
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

	return &models.BillingUsageListResponse{
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
		Items:    items,
	}, nil
}

func (s *BillingService) GetPricing(ctx context.Context) (*models.BillingPricing, error) {
	resp, err := s.assetClient.GetBillingPricing(ctx, &pb.GetBillingPricingRequest{})
	if err != nil {
		return nil, err
	}
	return pricingFromProto(resp.GetPricing()), nil
}

func (s *BillingService) UpdatePricing(ctx context.Context, ingressPrice, egressPrice, remark, operatorUserID string) (*models.BillingPricing, error) {
	resp, err := s.assetClient.UpdateBillingPricing(ctx, &pb.UpdateBillingPricingRequest{
		IngressPriceYuanPerGb: ingressPrice,
		EgressPriceYuanPerGb:  egressPrice,
		Remark:                remark,
		OperatorUserId:        operatorUserID,
	})
	if err != nil {
		return nil, err
	}
	return pricingFromProto(resp.GetPricing()), nil
}

func (s *BillingService) GetWelcomeCreditSettings(ctx context.Context) (*models.WelcomeCreditSettings, error) {
	resp, err := s.assetClient.GetWelcomeCreditSettings(ctx, &pb.GetWelcomeCreditSettingsRequest{})
	if err != nil {
		return nil, err
	}
	return welcomeCreditSettingsFromProto(resp.GetSettings()), nil
}

func (s *BillingService) UpdateWelcomeCreditSettings(ctx context.Context, enabled bool, amountYuan, currencyCode, updatedBy string) (*models.WelcomeCreditSettings, error) {
	resp, err := s.assetClient.UpdateWelcomeCreditSettings(ctx, &pb.UpdateWelcomeCreditSettingsRequest{
		Enabled:      enabled,
		AmountYuan:   amountYuan,
		CurrencyCode: currencyCode,
		UpdatedBy:    updatedBy,
	})
	if err != nil {
		return nil, err
	}
	return welcomeCreditSettingsFromProto(resp.GetSettings()), nil
}

func (s *BillingService) loadUsersByIDs(ctx context.Context, userIDs []string) (map[string]*pb.User, error) {
	result := map[string]*pb.User{}
	if len(userIDs) == 0 {
		return result, nil
	}

	resp, err := s.authClient.BatchGetUsers(ctx, &pb.BatchGetUsersRequest{UserIds: dedupeStrings(userIDs)})
	if err != nil {
		return result, err
	}
	for _, user := range resp.GetUsers() {
		result[user.GetUserId()] = user
	}
	return result, nil
}

func pricingFromProto(pricing *pb.BillingPricing) *models.BillingPricing {
	if pricing == nil {
		return &models.BillingPricing{}
	}
	return &models.BillingPricing{
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

func welcomeCreditSettingsFromProto(settings *pb.WelcomeCreditSettings) *models.WelcomeCreditSettings {
	if settings == nil {
		return &models.WelcomeCreditSettings{}
	}
	return &models.WelcomeCreditSettings{
		Enabled:      settings.GetEnabled(),
		AmountYuan:   settings.GetAmountYuan(),
		CurrencyCode: settings.GetCurrencyCode(),
		UpdatedAt:    settings.GetUpdatedAt(),
		UpdatedBy:    settings.GetUpdatedBy(),
	}
}

func extractUserIDs(items []*pb.BillingAccountSnapshot) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.GetUserId())
	}
	return result
}

func extractUserIDsFromLedger(items []*pb.LedgerEntryItem) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.GetUserId())
	}
	return result
}

func extractUserIDsFromUsage(items []*pb.TrafficUsageRecordItem) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.GetUserId())
	}
	return result
}

func extractUserIDsFromShortfalls(items []*pb.BillingShortfallOrderItem) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.GetUserId())
	}
	return result
}

func dedupeStrings(items []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func safeUserEmail(user *pb.User) string {
	if user == nil {
		return ""
	}
	return user.GetEmail()
}

func safeUserNickname(user *pb.User) string {
	if user == nil {
		return ""
	}
	return user.GetNickname()
}
