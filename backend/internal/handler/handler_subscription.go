package handler

import (
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/service"
	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	*Handler
	service *service.SubscriptionService
}

func NewSubscriptionHandler(base *Handler, service *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{Handler: base, service: service}
}
func (h *SubscriptionHandler) Create(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	var req dto.CreateSubscriptionRequest
	if !h.bind(c, &req) {
		return
	}
	item, err := h.service.Create(c.Request.Context(), id, req)
	if err != nil {
		respondError(c, err)
		return
	}
	created(c, item)
}
func (h *SubscriptionHandler) List(c *gin.Context) {
	id, valid := parseID(c)
	if !valid {
		return
	}
	items, err := h.service.List(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, items)
}
