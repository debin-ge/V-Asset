package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"vasset/admin-service/internal/models"
	"vasset/admin-service/internal/service"
)

type CookieHandler struct {
	cookieService *service.CookieService
}

func NewCookieHandler(cookieService *service.CookieService) *CookieHandler {
	return &CookieHandler{cookieService: cookieService}
}

func (h *CookieHandler) List(c *gin.Context) {
	var req models.ListCookiesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	resp, err := h.cookieService.List(c.Request.Context(), req)
	if err != nil {
		models.InternalError(c, "failed to list cookies: "+err.Error())
		return
	}

	models.Success(c, resp)
}

func (h *CookieHandler) Create(c *gin.Context) {
	var req models.CreateCookieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	id, err := h.cookieService.Create(c.Request.Context(), req)
	if err != nil {
		models.InternalError(c, "failed to create cookie: "+err.Error())
		return
	}

	models.Success(c, gin.H{"id": id})
}

func (h *CookieHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	var req models.UpdateCookieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	if err := h.cookieService.Update(c.Request.Context(), id, req); err != nil {
		models.InternalError(c, "failed to update cookie: "+err.Error())
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *CookieHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	if err := h.cookieService.Delete(c.Request.Context(), id); err != nil {
		models.InternalError(c, "failed to delete cookie: "+err.Error())
		return
	}

	models.Success(c, gin.H{"success": true})
}

func (h *CookieHandler) Freeze(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		models.BadRequest(c, "invalid cookie id")
		return
	}

	var req models.FreezeCookieRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		models.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	resp, err := h.cookieService.Freeze(c.Request.Context(), id, req.FreezeSeconds)
	if err != nil {
		models.InternalError(c, "failed to freeze cookie: "+err.Error())
		return
	}

	models.Success(c, resp)
}
