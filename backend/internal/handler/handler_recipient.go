package handler

import (
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
)

type RecipientHandler struct {
	*Handler
	service *service.RecipientService
}

func NewRecipientHandler(base *Handler, service *service.RecipientService) *RecipientHandler {
	return &RecipientHandler{Handler: base, service: service}
}
func (h *RecipientHandler) Create(c *gin.Context) {
	var req dto.CreateRecipientRequest
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
func (h *RecipientHandler) List(c *gin.Context) {
	page, size := pagination(c)
	items, total, err := h.service.List(c.Request.Context(), page, size)
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, gin.H{"items": items, "pagination": dto.Pagination{Page: page, PageSize: size, Total: total}})
}
func (h *RecipientHandler) Get(c *gin.Context) {
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
func (h *RecipientHandler) Update(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req dto.UpdateRecipientRequest
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
func (h *RecipientHandler) Delete(c *gin.Context) {
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
func (h *RecipientHandler) Confirm(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req dto.ConfirmRecipientRequest
	if !h.bind(c, &req) {
		return
	}
	item, err := h.service.Confirm(c.Request.Context(), id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, item)
}
