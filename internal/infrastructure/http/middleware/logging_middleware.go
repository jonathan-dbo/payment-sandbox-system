package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
)

func StructuredLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		shared.LogEvent("http_request", map[string]any{
			"method":      c.Request.Method,
			"path":        c.FullPath(),
			"status_code": c.Writer.Status(),
			"latency_ms":  time.Since(start).Milliseconds(),
			"client_ip":   c.ClientIP(),
		})
	}
}
