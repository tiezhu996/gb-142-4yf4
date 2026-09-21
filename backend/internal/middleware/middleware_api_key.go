package middleware

import (
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/gin-gonic/gin"
	"net/http"
)

func APIKey(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" || c.GetHeader("X-API-Key") != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, dto.APIResponse{Code: 40101, Message: "invalid API key"})
			return
		}
		c.Next()
	}
}
