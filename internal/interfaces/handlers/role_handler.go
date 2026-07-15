// Package handlers contains HTTP handler implementations.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
)

type RoleHandler struct{}

func NewRoleHandler() *RoleHandler {
	return &RoleHandler{}
}

func (h *RoleHandler) MerchantOnly(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserIDKey)
	role, _ := c.Get(middleware.ContextRoleKey)
	c.JSON(http.StatusOK, gin.H{
		"userId":     userID,
		"merchant_id": userID,
		"role":       role,
	})
}

func (h *RoleHandler) AdminOnly(c *gin.Context) {
	userID, _ := c.Get(middleware.ContextUserIDKey)
	role, _ := c.Get(middleware.ContextRoleKey)
	c.JSON(http.StatusOK, gin.H{
		"userId":  userID,
		"role":    role,
	})
}
