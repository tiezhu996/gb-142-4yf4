package handler

import (
	"errors"
	"github.com/blueship581/gbcarenotify/internal/dto"
	"github.com/blueship581/gbcarenotify/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"net/http"
	"strconv"
)

type Handler struct{ validator *validator.Validate }

func NewBaseHandler() *Handler { return &Handler{validator: validator.New()} }
func (h *Handler) bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Code: 40001, Message: "invalid JSON: " + err.Error()})
		return false
	}
	if err := h.validator.Struct(target); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Code: 40002, Message: "validation failed: " + err.Error()})
		return false
	}
	return true
}
func respondError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		c.JSON(http.StatusNotFound, dto.APIResponse{Code: 40401, Message: "resource not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, dto.APIResponse{Code: 50000, Message: "internal server error"})
}
func parseID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || value == 0 {
		c.JSON(http.StatusBadRequest, dto.APIResponse{Code: 40003, Message: "invalid id"})
		return 0, false
	}
	return uint(value), true
}
func pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, dto.APIResponse{Code: 0, Message: "ok", Data: data})
}
func created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, dto.APIResponse{Code: 0, Message: "ok", Data: data})
}
