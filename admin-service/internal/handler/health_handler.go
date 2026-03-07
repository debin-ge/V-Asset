package handler

import (
	"github.com/gin-gonic/gin"

	"vasset/admin-service/internal/models"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	models.Success(c, gin.H{"status": "ok"})
}
