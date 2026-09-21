package handler

import (
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"github.com/gin-gonic/gin"
	"strconv"
)

type SMSLogHandler struct {
	*Handler
	repo *repository.SMSLogRepository
}

func NewSMSLogHandler(base *Handler, repo *repository.SMSLogRepository) *SMSLogHandler {
	return &SMSLogHandler{Handler: base, repo: repo}
}
func (h *SMSLogHandler) List(c *gin.Context) {
	page, size := pagination(c)
	var recipientID *uint
	if raw := c.Query("recipient_id"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || id == 0 {
			c.JSON(400, dto.APIResponse{Code: 40003, Message: "invalid recipient_id"})
			return
		}
		parsed := uint(id)
		recipientID = &parsed
	}
	items, total, err := h.repo.List(c.Request.Context(), page, size, recipientID)
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, gin.H{"items": items, "pagination": dto.Pagination{Page: page, PageSize: size, Total: total}})
}
