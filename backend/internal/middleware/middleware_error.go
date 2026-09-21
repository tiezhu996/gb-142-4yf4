package middleware

import (
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/gin-gonic/gin"
	"net/http"
)

func ErrorResponder() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 && !c.IsAborted() {
			c.JSON(http.StatusInternalServerError, dto.APIResponse{Code: 50000, Message: "internal server error"})
		}
	}
}
