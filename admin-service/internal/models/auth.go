package models

type AdminLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AdminUser struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Role      int32  `json:"role"`
	CreatedAt string `json:"created_at,omitempty"`
}

type AdminMeResponse struct {
	User AdminUser `json:"user"`
}

type LoginResponse struct {
	User AdminUser `json:"user"`
}
