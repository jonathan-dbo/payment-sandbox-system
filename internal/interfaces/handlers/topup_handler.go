package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
)

type TopUpHandler struct {
	service *appTopUp.Service
}

func NewTopUpHandler(service *appTopUp.Service) *TopUpHandler {
	return &TopUpHandler{service: service}
}

func (h *TopUpHandler) RequestTopUp(c *gin.Context) {
	var req struct {
		MerchantID string `json:"merchantId"`
		Amount     int64  `json:"amount"`
		RequestKey string `json:"requestKey"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(req.MerchantID) == "" || req.Amount <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "merchantId and positive amount are required")
		return
	}
	model, err := h.service.Request(c.Request.Context(), req.MerchantID, req.Amount, req.RequestKey)
	if err != nil {
		respondError(c, http.StatusBadRequest, "topup_error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         model.ID,
		"merchantId": model.MerchantID,
		"amount":     model.Amount,
		"status":     model.Status,
		"requestKey": model.RequestKey,
	})
}

func (h *TopUpHandler) AdminUpdate(c *gin.Context) {
	var req struct {
		Success bool `json:"success"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	model, err := h.service.AdminUpdate(c.Request.Context(), c.Param("topupId"), req.Success)
	if err != nil {
		respondError(c, http.StatusBadRequest, "topup_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         model.ID,
		"merchantId": model.MerchantID,
		"amount":     model.Amount,
		"status":     model.Status,
		"requestKey": model.RequestKey,
	})
}

func (h *TopUpHandler) History(c *gin.Context) {
	merchantID := c.Query("merchantId")
	page, size := parsePagination(c)
	items, err := h.service.History(c.Request.Context(), merchantID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "topup_error", err.Error())
		return
	}
	paged := paginateSlice(items, page, size)
	respItems := make([]gin.H, 0, len(paged))
	for _, item := range paged {
		respItems = append(respItems, gin.H{
			"id":         item.ID,
			"merchantId": item.MerchantID,
			"amount":     item.Amount,
			"status":     item.Status,
			"requestKey": item.RequestKey,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": respItems, "page": page, "size": size, "total": len(items)})
}
