package service

import (
	"context"

	"vasset/admin-service/internal/models"
	pb "vasset/admin-service/proto"
)

type CookieService struct {
	assetClient pb.AssetServiceClient
}

func NewCookieService(assetClient pb.AssetServiceClient) *CookieService {
	return &CookieService{assetClient: assetClient}
}

func (s *CookieService) List(ctx context.Context, req models.ListCookiesRequest) (*models.CookieListResponse, error) {
	resp, err := s.assetClient.ListCookies(ctx, &pb.ListCookiesRequest{
		Platform: req.Platform,
		Status:   req.Status,
		Page:     int32(req.Page),
		PageSize: int32(req.PageSize),
	})
	if err != nil {
		return nil, err
	}

	items := make([]models.CookieInfo, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, cookieFromProto(item))
	}

	return &models.CookieListResponse{
		Total:    resp.Total,
		Page:     int(resp.Page),
		PageSize: int(resp.PageSize),
		Items:    items,
	}, nil
}

func (s *CookieService) Get(ctx context.Context, id int64) (*models.CookieInfo, error) {
	resp, err := s.assetClient.GetCookie(ctx, &pb.GetCookieRequest{Id: id})
	if err != nil {
		return nil, err
	}
	if resp.GetCookie() == nil {
		return nil, nil
	}

	item := cookieFromProto(resp.GetCookie())
	return &item, nil
}

func (s *CookieService) Create(ctx context.Context, req models.CreateCookieRequest) (int64, error) {
	resp, err := s.assetClient.CreateCookie(ctx, &pb.CreateCookieRequest{
		Platform:      req.Platform,
		Name:          req.Name,
		Content:       req.Content,
		ExpireAt:      req.ExpireAt,
		FreezeSeconds: req.FreezeSeconds,
	})
	if err != nil {
		return 0, err
	}
	return resp.Id, nil
}

func (s *CookieService) Update(ctx context.Context, id int64, req models.UpdateCookieRequest) error {
	_, err := s.assetClient.UpdateCookie(ctx, &pb.UpdateCookieRequest{
		Id:            id,
		Name:          req.Name,
		Content:       req.Content,
		ExpireAt:      req.ExpireAt,
		FreezeSeconds: req.FreezeSeconds,
	})
	return err
}

func (s *CookieService) Delete(ctx context.Context, id int64) error {
	_, err := s.assetClient.DeleteCookie(ctx, &pb.DeleteCookieRequest{Id: id})
	return err
}

func (s *CookieService) Freeze(ctx context.Context, id int64, freezeSeconds int32) (*models.FreezeCookieResponse, error) {
	resp, err := s.assetClient.FreezeCookie(ctx, &pb.FreezeCookieRequest{
		CookieId:      id,
		FreezeSeconds: freezeSeconds,
	})
	if err != nil {
		return nil, err
	}

	return &models.FreezeCookieResponse{
		Success:     resp.Success,
		FrozenUntil: resp.FrozenUntil,
	}, nil
}

func cookieFromProto(item *pb.CookieInfo) models.CookieInfo {
	return models.CookieInfo{
		ID:            item.Id,
		Platform:      item.Platform,
		Name:          item.Name,
		Content:       item.Content,
		Status:        item.Status,
		ExpireAt:      item.ExpireAt,
		FrozenUntil:   item.FrozenUntil,
		FreezeSeconds: item.FreezeSeconds,
		LastUsedAt:    item.LastUsedAt,
		UseCount:      int64(item.UseCount),
		SuccessCount:  int64(item.SuccessCount),
		FailCount:     int64(item.FailCount),
		CreatedAt:     item.CreatedAt,
		UpdatedAt:     item.UpdatedAt,
	}
}
