// Package middleware provides HTTP middleware for the Gin framework
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware handles errors and formats them as JSON responses
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_server_error",
				"message": err,
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_server_error",
				"message": "An unexpected error occurred",
			})
		}
		c.Abort()
	})
}
