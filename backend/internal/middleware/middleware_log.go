package middleware

import (
	"github.com/gin-gonic/gin"
	"log/slog"
	"time"
)

func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmtRequestID()
		}
		c.Header("X-Request-ID", requestID)
		c.Next()
		logger.Info("http request", "request_id", requestID, "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "latency_ms", time.Since(started).Milliseconds())
	}
}
func fmtRequestID() string { return time.Now().UTC().Format("20060102T150405.000000000") }
