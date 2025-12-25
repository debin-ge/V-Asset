package service

import (
	"context"
	"errors"
	"time"

	"vasset/asset-service/internal/config"
	"vasset/asset-service/internal/models"
	"vasset/asset-service/internal/repository"
)

// CookieService Cookie 业务逻辑层
type CookieService struct {
	repo *repository.CookieRepository
	cfg  *config.Config
}

// NewCookieService 创建 Cookie 服务
func NewCookieService(repo *repository.CookieRepository, cfg *config.Config) *CookieService {
	return &CookieService{
		repo: repo,
		cfg:  cfg,
	}
}

// Create 创建 Cookie
func (s *CookieService) Create(ctx context.Context, cookie *models.Cookie) (int64, error) {
	// 设置默认冷冻时间
	if cookie.FreezeSeconds == 0 {
		cookie.FreezeSeconds = s.cfg.Cookie.DefaultFreezeSeconds
	}

	return s.repo.Create(ctx, cookie)
}

// GetByID 获取 Cookie
func (s *CookieService) GetByID(ctx context.Context, id int64) (*models.Cookie, error) {
	return s.repo.GetByID(ctx, id)
}

// Update 更新 Cookie
func (s *CookieService) Update(ctx context.Context, cookie *models.Cookie) error {
	return s.repo.Update(ctx, cookie)
}

// Delete 删除 Cookie
func (s *CookieService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// List 列表查询
func (s *CookieService) List(ctx context.Context, filter *models.CookieFilter) (*models.CookieResult, error) {
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

	return s.repo.List(ctx, filter)
}

// GetAvailableCookie 获取可用 Cookie
func (s *CookieService) GetAvailableCookie(ctx context.Context, platform string) (int64, string, error) {
	cookie, err := s.repo.GetAvailableCookie(ctx, platform)
	if err != nil {
		return 0, "", err
	}
	if cookie == nil {
		return 0, "", errors.New("no available cookie for platform: " + platform)
	}

	return cookie.ID, cookie.Content, nil
}

// ReportUsage 报告 Cookie 使用结果（成功或失败后冷冻）
func (s *CookieService) ReportUsage(ctx context.Context, id int64, success bool) error {
	cookie, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if cookie == nil {
		return errors.New("cookie not found")
	}

	freezeSeconds := cookie.FreezeSeconds
	if freezeSeconds == 0 {
		freezeSeconds = s.cfg.Cookie.DefaultFreezeSeconds
	}

	return s.repo.UpdateUsage(ctx, id, success, freezeSeconds)
}

// Freeze 手动冷冻 Cookie
func (s *CookieService) Freeze(ctx context.Context, id int64, freezeSeconds int) (*time.Time, error) {
	if freezeSeconds <= 0 {
		freezeSeconds = s.cfg.Cookie.DefaultFreezeSeconds
	}

	return s.repo.Freeze(ctx, id, freezeSeconds)
}
