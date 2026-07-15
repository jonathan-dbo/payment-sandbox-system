package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
)

func respondError(c *gin.Context, status int, code, message string) {
	attrs := map[string]any{
		"method":      c.Request.Method,
		"path":        c.Request.URL.Path,
		"status_code": status,
		"error_code":  code,
		"message":     message,
		"client_ip":   c.ClientIP(),
	}
	shared.LogEvent("api_error", attrs)

	if status >= http.StatusInternalServerError {
		log.Printf("internal error: method=%s path=%s status=%d code=%s message=%s", c.Request.Method, c.Request.URL.Path, status, code, message)
	}

	c.JSON(status, gin.H{
		"error":   code,
		"message": message,
	})
}

func parsePagination(c *gin.Context) (int, int) {
	page := 1
	size := 10
	if v := c.Query("page"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			size = parsed
		}
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

func paginateSlice[T any](items []T, page, size int) []T {
	start := (page - 1) * size
	if start >= len(items) {
		return []T{}
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
