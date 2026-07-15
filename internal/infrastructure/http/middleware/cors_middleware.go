package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware applies simple CORS handling for frontend-to-backend browser calls.
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	origins := normalizeOrigins(allowedOrigins)
	allowAny := len(origins) == 0

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if allowAny {
				c.Header("Access-Control-Allow-Origin", "*")
			} else if isOriginAllowed(origin, origins) {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept,Origin")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func normalizeOrigins(input []string) []string {
	out := make([]string, 0, len(input))
	for _, origin := range input {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func isOriginAllowed(origin string, allowed []string) bool {
	for _, item := range allowed {
		if item == origin {
			return true
		}
	}
	return false
}
