package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
)

type RefundHandler struct {
	service *appRefund.Service
}

func NewRefundHandler(service *appRefund.Service) *RefundHandler {
	return &RefundHandler{service: service}
}

func (h *RefundHandler) RequestRefund(c *gin.Context) {
	var req struct {
		InvoiceID  string `json:"invoiceId"`
		MerchantID string `json:"merchantId"`
		Amount     int64  `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(req.InvoiceID) == "" || strings.TrimSpace(req.MerchantID) == "" || req.Amount <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "invoiceId, merchantId, and positive amount are required")
		return
	}
	model, err := h.service.RequestRefund(c.Request.Context(), req.InvoiceID, req.MerchantID, req.Amount)
	if err != nil {
		respondError(c, http.StatusBadRequest, "refund_error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":         model.ID,
		"invoiceId":  model.InvoiceID,
		"merchantId": model.MerchantID,
		"amount":     model.Amount,
		"status":     model.Status,
		"history":    model.History,
		"createdAt":  model.CreatedAt,
	})
}

func (h *RefundHandler) ApproveRefund(c *gin.Context) {
	model, err := h.service.Approve(c.Request.Context(), c.Param("refundId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "refund_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         model.ID,
		"invoiceId":  model.InvoiceID,
		"merchantId": model.MerchantID,
		"amount":     model.Amount,
		"status":     model.Status,
		"history":    model.History,
		"createdAt":  model.CreatedAt,
	})
}

func (h *RefundHandler) RejectRefund(c *gin.Context) {
	model, err := h.service.Reject(c.Request.Context(), c.Param("refundId"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "refund_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         model.ID,
		"invoiceId":  model.InvoiceID,
		"merchantId": model.MerchantID,
		"amount":     model.Amount,
		"status":     model.Status,
		"history":    model.History,
		"createdAt":  model.CreatedAt,
	})
}

func (h *RefundHandler) ProcessRefund(c *gin.Context) {
	var req struct {
		Success bool `json:"success"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	model, err := h.service.Process(c.Request.Context(), c.Param("refundId"), req.Success)
	if err != nil {
		respondError(c, http.StatusBadRequest, "refund_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":         model.ID,
		"invoiceId":  model.InvoiceID,
		"merchantId": model.MerchantID,
		"amount":     model.Amount,
		"status":     model.Status,
		"history":    model.History,
		"createdAt":  model.CreatedAt,
	})
}

func (h *RefundHandler) ListHistory(c *gin.Context) {
	merchantID := c.Query("merchantId")
	page, size := parsePagination(c)
	items, err := h.service.History(c.Request.Context(), merchantID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "refund_error", err.Error())
		return
	}
	paged := paginateSlice(items, page, size)
	respItems := make([]gin.H, 0, len(paged))
	for _, item := range paged {
		respItems = append(respItems, gin.H{
			"id":         item.ID,
			"invoiceId":  item.InvoiceID,
			"merchantId": item.MerchantID,
			"amount":     item.Amount,
			"status":     item.Status,
			"history":    item.History,
			"createdAt":  item.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": respItems, "page": page, "size": size, "total": len(items)})
}
