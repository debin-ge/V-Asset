package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"vasset/admin-service/internal/models"
	"vasset/admin-service/internal/service"
)

type StatsHandler struct {
	statsService *service.StatsService
}

func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

func (h *StatsHandler) Overview(c *gin.Context) {
	resp, err := h.statsService.GetOverview(c.Request.Context())
	if err != nil {
		models.InternalError(c, "failed to get overview: "+err.Error())
		return
	}

	models.Success(c, resp)
}

func (h *StatsHandler) RequestTrend(c *gin.Context) {
	granularity := c.DefaultQuery("granularity", "day")
	limitValue := c.DefaultQuery("limit", "7")
	limit, err := strconv.Atoi(limitValue)
	if err != nil || limit <= 0 {
		limit = 7
	}

	resp, err := h.statsService.GetRequestTrend(c.Request.Context(), granularity, int32(limit))
	if err != nil {
		models.InternalError(c, "failed to get request trend: "+err.Error())
		return
	}

	models.Success(c, resp)
}

func (h *StatsHandler) Users(c *gin.Context) {
	resp, err := h.statsService.GetUserStats(c.Request.Context())
	if err != nil {
		models.InternalError(c, "failed to get user stats: "+err.Error())
		return
	}

	models.Success(c, resp)
}
