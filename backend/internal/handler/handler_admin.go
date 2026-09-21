package handler

import (
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	*Handler
	service *service.StatusService
}

func NewAdminHandler(base *Handler, service *service.StatusService) *AdminHandler {
	return &AdminHandler{Handler: base, service: service}
}
func (h *AdminHandler) Statuses(c *gin.Context) {
	page, size := pagination(c)
	items, total, err := h.service.List(c.Request.Context(), page, size)
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, gin.H{"items": items, "pagination": dto.Pagination{Page: page, PageSize: size, Total: total}})
}
