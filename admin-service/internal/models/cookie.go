package models

type CreateCookieRequest struct {
	Platform      string `json:"platform" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Content       string `json:"content" binding:"required"`
	ExpireAt      string `json:"expire_at"`
	FreezeSeconds int32  `json:"freeze_seconds"`
}

type UpdateCookieRequest struct {
	Name          string `json:"name"`
	Content       string `json:"content"`
	ExpireAt      string `json:"expire_at"`
	FreezeSeconds int32  `json:"freeze_seconds"`
}

type FreezeCookieRequest struct {
	FreezeSeconds int32 `json:"freeze_seconds"`
}

type ListCookiesRequest struct {
	Platform string `form:"platform"`
	Status   int32  `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type CookieInfo struct {
	ID            int64  `json:"id"`
	Platform      string `json:"platform"`
	Name          string `json:"name"`
	Content       string `json:"content"`
	Status        int32  `json:"status"`
	ExpireAt      string `json:"expire_at,omitempty"`
	FrozenUntil   string `json:"frozen_until,omitempty"`
	FreezeSeconds int32  `json:"freeze_seconds"`
	LastUsedAt    string `json:"last_used_at,omitempty"`
	UseCount      int64  `json:"use_count"`
	SuccessCount  int64  `json:"success_count"`
	FailCount     int64  `json:"fail_count"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type CookieListResponse struct {
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Items    []CookieInfo `json:"items"`
}

type FreezeCookieResponse struct {
	Success     bool   `json:"success"`
	FrozenUntil string `json:"frozen_until"`
}
