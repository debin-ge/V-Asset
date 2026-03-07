package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"vasset/admin-service/internal/models"
	"vasset/admin-service/internal/service"
)

type ProxyHandler struct {
	proxyService *service.ProxyService
}

func NewProxyHandler(proxyService *service.ProxyService) *ProxyHandler {
	return &ProxyHandler{proxyService: proxyService}
}

func (h *ProxyHandler) GetSourceStatus(c *gin.Context) {
	resp, err := h.proxyService.GetSourceStatus(c.Request.Context())
	if err != nil {
		models.InternalError(c, "failed to get proxy status: "+err.Error())
		return
	}

	models.Success(c, resp)
}

func (h *ProxyHandler) GetSourcePolicy(c *gin.Context) {
	resp, err := h.proxyService.GetSourcePolicy(c.Request.Context())
	if err != nil {
		models.InternalError(c, "failed to get proxy policy: "+err.Error())
		return
	}

	models.Success(c, resp)
}

func (h *ProxyHandler) UpdateSourcePolicy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid policy id")
		return
	}

	var req models.UpdateProxySourcePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	if err := h.proxyService.UpdateSourcePolicy(c.Request.Context(), id, req); err != nil {
		models.InternalError(c, "failed to update proxy policy: "+err.Error())
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *ProxyHandler) List(c *gin.Context) {
	var req models.ListProxiesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	resp, err := h.proxyService.List(c.Request.Context(), req)
	if err != nil {
		models.InternalError(c, "failed to list proxies: "+err.Error())
		return
	}

	models.Success(c, resp)
}

func (h *ProxyHandler) Create(c *gin.Context) {
	var req models.CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	id, err := h.proxyService.Create(c.Request.Context(), req)
	if err != nil {
		models.InternalError(c, "failed to create proxy: "+err.Error())
		return
	}

	models.Success(c, gin.H{"id": id})
}

func (h *ProxyHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	var req models.UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	if err := h.proxyService.Update(c.Request.Context(), id, req); err != nil {
		models.InternalError(c, "failed to update proxy: "+err.Error())
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *ProxyHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	var req models.UpdateProxyStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	if err := h.proxyService.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		models.InternalError(c, "failed to update proxy status: "+err.Error())
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *ProxyHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid proxy id")
		return
	}

	if err := h.proxyService.Delete(c.Request.Context(), id); err != nil {
		models.InternalError(c, "failed to delete proxy: "+err.Error())
		return
	}

	models.Success(c, gin.H{"success": true})
}
