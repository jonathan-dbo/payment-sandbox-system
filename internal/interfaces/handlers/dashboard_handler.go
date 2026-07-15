package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	appDashboard "github.com/gonszalito/go-ddd-architecture/internal/application/dashboard"
)

type DashboardHandler struct {
	service *appDashboard.Service
}

func NewDashboardHandler(service *appDashboard.Service) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) Stats(c *gin.Context) {
	filter := appDashboard.Filter{MerchantID: c.Query("merchantId")}
	if v := c.Query("startDate"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid_request", "startDate must be RFC3339")
			return
		}
		filter.StartDate = &t
	}
	if v := c.Query("endDate"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid_request", "endDate must be RFC3339")
			return
		}
		filter.EndDate = &t
	}
	stats, err := h.service.GetStats(c.Request.Context(), filter)
	if err != nil {
		respondError(c, http.StatusBadRequest, "dashboard_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, stats)
}
