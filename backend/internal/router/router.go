package router

import (
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/handler"
	"github.com/blueship581/gbcarenotify/internal/middleware"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"os"
)

type Handlers struct {
	Recipients    *handler.RecipientHandler
	Subscriptions *handler.SubscriptionHandler
	Templates     *handler.TemplateHandler
	SMSLogs       *handler.SMSLogHandler
	Alerts        *handler.AlertHandler
	Admin         *handler.AdminHandler
}

func New(apiKey string, handlers Handlers) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery(), middleware.CORS(), middleware.RequestLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil))), middleware.ErrorResponder())
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, dto.APIResponse{Code: 0, Message: "ok", Data: gin.H{"status": "healthy"}})
	})
	api := engine.Group("/api/v1")
	api.POST("/recipients", handlers.Recipients.Create)
	api.GET("/recipients", handlers.Recipients.List)
	api.GET("/recipients/:id", handlers.Recipients.Get)
	api.PUT("/recipients/:id", handlers.Recipients.Update)
	api.DELETE("/recipients/:id", handlers.Recipients.Delete)
	api.POST("/recipients/:id/confirmations", handlers.Recipients.Confirm)
	api.POST("/recipients/:id/subscriptions", handlers.Subscriptions.Create)
	api.GET("/recipients/:id/subscriptions", handlers.Subscriptions.List)
	api.POST("/templates", handlers.Templates.Create)
	api.GET("/templates", handlers.Templates.List)
	api.GET("/templates/:id", handlers.Templates.Get)
	api.PUT("/templates/:id", handlers.Templates.Update)
	api.DELETE("/templates/:id", handlers.Templates.Delete)
	api.GET("/sms-logs", handlers.SMSLogs.List)
	api.GET("/admin/statuses", handlers.Admin.Statuses)
	api.POST("/admin/jobs/greetings", handlers.Alerts.RunGreetings)
	api.POST("/admin/jobs/alerts", handlers.Alerts.RunAlerts)
	external := api.Group("/external", middleware.APIKey(apiKey))
	external.GET("/sms-logs", handlers.SMSLogs.List)
	return engine
}
