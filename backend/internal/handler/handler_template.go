package handler

import (
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type TemplateHandler struct {
	*Handler
	service *service.TemplateService
}

func NewTemplateHandler(base *Handler, service *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{Handler: base, service: service}
}
func (h *TemplateHandler) Create(c *gin.Context) {
	var req dto.CreateTemplateRequest
	if !h.bind(c, &req) {
		return
	}
	item, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		respondError(c, err)
		return
	}
	created(c, item)
}
func (h *TemplateHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), c.Query("category"))
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, items)
}
func (h *TemplateHandler) Get(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, item)
}
func (h *TemplateHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req dto.UpdateTemplateRequest
	if !h.bind(c, &req) {
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, item)
}
func (h *TemplateHandler) Delete(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
