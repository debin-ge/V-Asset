package service

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	pb "youdlp/admin-service/proto"
)

type stubBillingAuthClient struct {
	searchResp *pb.SearchUsersResponse
	searchErr  error
	searchReq  *pb.SearchUsersRequest
}

func (s *stubBillingAuthClient) SearchUsers(_ context.Context, in *pb.SearchUsersRequest, _ ...grpc.CallOption) (*pb.SearchUsersResponse, error) {
	reqCopy := *in
	s.searchReq = &reqCopy
	return s.searchResp, s.searchErr
}

func (s *stubBillingAuthClient) GetUserInfo(context.Context, *pb.GetUserInfoRequest, ...grpc.CallOption) (*pb.GetUserInfoResponse, error) {
	return &pb.GetUserInfoResponse{}, nil
}

func (s *stubBillingAuthClient) BatchGetUsers(context.Context, *pb.BatchGetUsersRequest, ...grpc.CallOption) (*pb.BatchGetUsersResponse, error) {
	return &pb.BatchGetUsersResponse{}, nil
}

type stubBillingAssetClient struct {
	listResp                *pb.ListBillingAccountsResponse
	listErr                 error
	listReq                 *pb.ListBillingAccountsRequest
	getWelcomeCreditResp    *pb.GetWelcomeCreditSettingsResponse
	getWelcomeCreditErr     error
	updateWelcomeCreditReq  *pb.UpdateWelcomeCreditSettingsRequest
	updateWelcomeCreditResp *pb.UpdateWelcomeCreditSettingsResponse
	updateWelcomeCreditErr  error
}

func (s *stubBillingAssetClient) ListBillingAccounts(_ context.Context, in *pb.ListBillingAccountsRequest, _ ...grpc.CallOption) (*pb.ListBillingAccountsResponse, error) {
	reqCopy := *in
	reqCopy.UserIds = append([]string(nil), in.GetUserIds()...)
	s.listReq = &reqCopy
	return s.listResp, s.listErr
}

func (s *stubBillingAssetClient) GetBillingAccountDetail(context.Context, *pb.GetBillingAccountDetailRequest, ...grpc.CallOption) (*pb.GetBillingAccountDetailResponse, error) {
	return &pb.GetBillingAccountDetailResponse{}, nil
}

func (s *stubBillingAssetClient) AdjustBillingBalance(context.Context, *pb.AdjustBillingBalanceRequest, ...grpc.CallOption) (*pb.AdjustBillingBalanceResponse, error) {
	return &pb.AdjustBillingBalanceResponse{}, nil
}

func (s *stubBillingAssetClient) ListBillingShortfalls(context.Context, *pb.ListBillingShortfallsRequest, ...grpc.CallOption) (*pb.ListBillingShortfallsResponse, error) {
	return &pb.ListBillingShortfallsResponse{}, nil
}

func (s *stubBillingAssetClient) ReconcileBillingShortfall(context.Context, *pb.ReconcileBillingShortfallRequest, ...grpc.CallOption) (*pb.ReconcileBillingShortfallResponse, error) {
	return &pb.ReconcileBillingShortfallResponse{}, nil
}

func (s *stubBillingAssetClient) ListBillingLedger(context.Context, *pb.ListBillingLedgerRequest, ...grpc.CallOption) (*pb.ListBillingLedgerResponse, error) {
	return &pb.ListBillingLedgerResponse{}, nil
}

func (s *stubBillingAssetClient) ListTrafficUsageRecords(context.Context, *pb.ListTrafficUsageRecordsRequest, ...grpc.CallOption) (*pb.ListTrafficUsageRecordsResponse, error) {
	return &pb.ListTrafficUsageRecordsResponse{}, nil
}

func (s *stubBillingAssetClient) GetBillingPricing(context.Context, *pb.GetBillingPricingRequest, ...grpc.CallOption) (*pb.GetBillingPricingResponse, error) {
	return &pb.GetBillingPricingResponse{}, nil
}

func (s *stubBillingAssetClient) UpdateBillingPricing(context.Context, *pb.UpdateBillingPricingRequest, ...grpc.CallOption) (*pb.UpdateBillingPricingResponse, error) {
	return &pb.UpdateBillingPricingResponse{}, nil
}

func (s *stubBillingAssetClient) GetWelcomeCreditSettings(context.Context, *pb.GetWelcomeCreditSettingsRequest, ...grpc.CallOption) (*pb.GetWelcomeCreditSettingsResponse, error) {
	return s.getWelcomeCreditResp, s.getWelcomeCreditErr
}

func (s *stubBillingAssetClient) UpdateWelcomeCreditSettings(_ context.Context, in *pb.UpdateWelcomeCreditSettingsRequest, _ ...grpc.CallOption) (*pb.UpdateWelcomeCreditSettingsResponse, error) {
	if in != nil {
		reqCopy := *in
		s.updateWelcomeCreditReq = &reqCopy
	}
	return s.updateWelcomeCreditResp, s.updateWelcomeCreditErr
}

func TestBillingServiceListAccounts_UsesSearchForDefaultList(t *testing.T) {
	authClient := &stubBillingAuthClient{
		searchResp: &pb.SearchUsersResponse{
			Total: 2,
			Users: []*pb.User{
				{UserId: "admin-user", Email: "admin@example.com", Nickname: "Admin"},
				{UserId: "normal-user", Email: "user@example.com", Nickname: "User"},
			},
		},
	}
	assetClient := &stubBillingAssetClient{
		listResp: &pb.ListBillingAccountsResponse{
			Items: []*pb.BillingAccountSnapshot{
				{UserId: "normal-user", AvailableBalanceYuan: "100", Status: 1},
				{UserId: "admin-user", AvailableBalanceYuan: "200", Status: 1},
			},
		},
	}

	svc := NewBillingService(authClient, assetClient)

	resp, err := svc.ListAccounts(context.Background(), "", 1, 20, 0)
	if err != nil {
		t.Fatalf("ListAccounts returned error: %v", err)
	}

	if authClient.searchReq == nil {
		t.Fatal("expected SearchUsers to be called")
	}
	if authClient.searchReq.GetQuery() != "" {
		t.Fatalf("expected empty query, got %q", authClient.searchReq.GetQuery())
	}
	if authClient.searchReq.GetPage() != 1 || authClient.searchReq.GetPageSize() != 20 {
		t.Fatalf("unexpected SearchUsers pagination: page=%d pageSize=%d", authClient.searchReq.GetPage(), authClient.searchReq.GetPageSize())
	}

	if assetClient.listReq == nil {
		t.Fatal("expected ListBillingAccounts to be called")
	}
	if assetClient.listReq.GetPage() != 1 || assetClient.listReq.GetPageSize() != 2 {
		t.Fatalf("expected asset request to avoid double pagination, got page=%d pageSize=%d", assetClient.listReq.GetPage(), assetClient.listReq.GetPageSize())
	}
	if got := assetClient.listReq.GetUserIds(); len(got) != 2 || got[0] != "admin-user" || got[1] != "normal-user" {
		t.Fatalf("unexpected user ids: %#v", got)
	}

	if resp.Total != 2 || resp.Page != 1 || resp.PageSize != 20 {
		t.Fatalf("unexpected response pagination: %+v", resp)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}
	if resp.Items[0].UserID != "admin-user" || resp.Items[0].Email != "admin@example.com" {
		t.Fatalf("expected first item to follow auth search order, got %+v", resp.Items[0])
	}
	if resp.Items[1].UserID != "normal-user" || resp.Items[1].Email != "user@example.com" {
		t.Fatalf("unexpected second item: %+v", resp.Items[1])
	}
}

func TestBillingServiceListAccounts_AvoidsDoublePaginationForSearchResults(t *testing.T) {
	authClient := &stubBillingAuthClient{
		searchResp: &pb.SearchUsersResponse{
			Total: 30,
			Users: []*pb.User{
				{UserId: "page-two-user", Email: "two@example.com", Nickname: "Two"},
			},
		},
	}
	assetClient := &stubBillingAssetClient{
		listResp: &pb.ListBillingAccountsResponse{
			Items: []*pb.BillingAccountSnapshot{
				{UserId: "page-two-user", AvailableBalanceYuan: "500", Status: 1},
			},
		},
	}

	svc := NewBillingService(authClient, assetClient)

	resp, err := svc.ListAccounts(context.Background(), "two@example.com", 2, 1, 0)
	if err != nil {
		t.Fatalf("ListAccounts returned error: %v", err)
	}

	if assetClient.listReq == nil {
		t.Fatal("expected ListBillingAccounts to be called")
	}
	if assetClient.listReq.GetPage() != 1 || assetClient.listReq.GetPageSize() != 1 {
		t.Fatalf("expected asset pagination to reset for scoped user ids, got page=%d pageSize=%d", assetClient.listReq.GetPage(), assetClient.listReq.GetPageSize())
	}
	if resp.Total != 30 || len(resp.Items) != 1 || resp.Items[0].UserID != "page-two-user" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestBillingServiceListAccounts_ReturnsEmptyWhenSearchHasNoMatches(t *testing.T) {
	authClient := &stubBillingAuthClient{
		searchResp: &pb.SearchUsersResponse{
			Total: 0,
			Users: []*pb.User{},
		},
	}
	assetClient := &stubBillingAssetClient{}

	svc := NewBillingService(authClient, assetClient)

	resp, err := svc.ListAccounts(context.Background(), "missing@example.com", 1, 20, 0)
	if err != nil {
		t.Fatalf("ListAccounts returned error: %v", err)
	}

	if assetClient.listReq != nil {
		t.Fatalf("expected asset client not to be called when search returned no users, got %+v", assetClient.listReq)
	}
	if resp.Total != 0 || len(resp.Items) != 0 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestGetWelcomeCreditSettings(t *testing.T) {
	authClient := &stubBillingAuthClient{}
	assetClient := &stubBillingAssetClient{
		getWelcomeCreditResp: &pb.GetWelcomeCreditSettingsResponse{
			Settings: &pb.WelcomeCreditSettings{
				Enabled:      true,
				AmountYuan:   "1.50",
				CurrencyCode: "CNY",
				UpdatedAt:    "2026-03-21T12:00:00Z",
				UpdatedBy:    "admin-1",
			},
		},
	}

	svc := NewBillingService(authClient, assetClient)

	resp, err := svc.GetWelcomeCreditSettings(context.Background())
	if err != nil {
		t.Fatalf("GetWelcomeCreditSettings returned error: %v", err)
	}

	if !resp.Enabled || resp.AmountYuan != "1.50" || resp.CurrencyCode != "CNY" || resp.UpdatedAt != "2026-03-21T12:00:00Z" || resp.UpdatedBy != "admin-1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestUpdateWelcomeCreditSettings(t *testing.T) {
	authClient := &stubBillingAuthClient{}
	assetClient := &stubBillingAssetClient{
		updateWelcomeCreditResp: &pb.UpdateWelcomeCreditSettingsResponse{
			Success: true,
			Settings: &pb.WelcomeCreditSettings{
				Enabled:      false,
				AmountYuan:   "2.00",
				CurrencyCode: "CNY",
				UpdatedAt:    "2026-03-21T13:00:00Z",
				UpdatedBy:    "admin-2",
			},
		},
	}

	svc := NewBillingService(authClient, assetClient)

	resp, err := svc.UpdateWelcomeCreditSettings(context.Background(), false, "2.00", "CNY", "admin-2")
	if err != nil {
		t.Fatalf("UpdateWelcomeCreditSettings returned error: %v", err)
	}

	if assetClient.updateWelcomeCreditReq == nil {
		t.Fatal("expected UpdateWelcomeCreditSettings request to be sent")
	}
	if assetClient.updateWelcomeCreditReq.GetEnabled() != false || assetClient.updateWelcomeCreditReq.GetAmountYuan() != "2.00" || assetClient.updateWelcomeCreditReq.GetCurrencyCode() != "CNY" || assetClient.updateWelcomeCreditReq.GetUpdatedBy() != "admin-2" {
		t.Fatalf("unexpected upstream request: %+v", assetClient.updateWelcomeCreditReq)
	}

	if resp.Enabled != false || resp.AmountYuan != "2.00" || resp.CurrencyCode != "CNY" || resp.UpdatedAt != "2026-03-21T13:00:00Z" || resp.UpdatedBy != "admin-2" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
