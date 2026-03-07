package models

type AdminSession struct {
	SessionID string    `json:"session_id"`
	User      AdminUser `json:"user"`
}
