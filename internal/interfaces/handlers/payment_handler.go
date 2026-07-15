package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
)

type PaymentHandler struct {
	service *appPayment.Service
}

func NewPaymentHandler(service *appPayment.Service) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) ResolvePaymentLink(c *gin.Context) {
	token := c.Param("token")
	if strings.TrimSpace(token) == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "payment token is required")
		return
	}
	inv, err := h.service.ResolvePublicToken(c.Request.Context(), token)
	if err != nil {
		respondError(c, http.StatusNotFound, "not_found", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"invoiceId":     inv.ID,
		"invoiceNumber": inv.InvoiceNumber,
		"amount":        inv.Amount,
		"currency":      inv.Currency,
		"status":        inv.Status,
	})
}

func (h *PaymentHandler) CreatePaymentIntent(c *gin.Context) {
	token := c.Param("token")
	var req struct {
		Method string `json:"method"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		if errors.Is(err, io.EOF) {
			respondError(c, http.StatusBadRequest, "invalid_request", "request body is required")
			return
		}
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(token) == "" || strings.TrimSpace(req.Method) == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "token and method are required")
		return
	}
	intent, err := h.service.CreateIntent(c.Request.Context(), token, req.Method)
	if err != nil {
		respondError(c, http.StatusBadRequest, "payment_intent_error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, intent)
}

func (h *PaymentHandler) SimulatePaymentIntent(c *gin.Context) {
	intentID := c.Param("intentId")
	var req struct {
		Outcome string `json:"outcome"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if strings.TrimSpace(intentID) == "" || strings.TrimSpace(req.Outcome) == "" {
		respondError(c, http.StatusBadRequest, "invalid_request", "intentId and outcome are required")
		return
	}
	intent, err := h.service.SimulateAdminOutcome(c.Request.Context(), intentID, req.Outcome)
	if err != nil {
		respondError(c, http.StatusBadRequest, "simulation_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, intent)
}
