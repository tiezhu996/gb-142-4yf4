package handler

import (
	"github.com/blueship581/gbcarenotify/internal/service"
	"github.com/gin-gonic/gin"
)

type AlertHandler struct {
	*Handler
	notifications *service.NotificationService
	alerts        *service.AlertService
}

func NewAlertHandler(base *Handler, notifications *service.NotificationService, alerts *service.AlertService) *AlertHandler {
	return &AlertHandler{Handler: base, notifications: notifications, alerts: alerts}
}
func (h *AlertHandler) RunGreetings(c *gin.Context) {
	count, err := h.notifications.SendDueGreetings(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, gin.H{"sent": count})
}
func (h *AlertHandler) RunAlerts(c *gin.Context) {
	count, err := h.alerts.SendOverdueAlerts(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	ok(c, gin.H{"sent": count})
}
