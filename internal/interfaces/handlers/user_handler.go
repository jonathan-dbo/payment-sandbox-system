package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	domainUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
	apigen "github.com/gonszalito/go-ddd-architecture/internal/interfaces"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
)

type UserHandler struct {
	userService    *appUser.UserService
	invoiceService *appInvoice.Service
}

func NewUserHandler(userService *appUser.UserService, invoiceService *appInvoice.Service) *UserHandler {
	return &UserHandler{
		userService:    userService,
		invoiceService: invoiceService,
	}
}

func (h *UserHandler) Register(ctx context.Context, req apigen.RegisterRequestObject) (apigen.RegisterResponseObject, error) {
	if req.Body == nil {
		return apigen.Register400JSONResponse{Error: "invalid_request", Message: "request body is required"}, nil
	}

	result, err := h.userService.Register(ctx, req.Body.Name, string(req.Body.Email), req.Body.Password, roleValue(req.Body.Role))
	if err != nil {
		return registerErrorResponse(err), nil
	}

	return apigen.Register201JSONResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		Role:      result.Role,
		UserId:    result.UserID,
		Email:     result.Email,
	}, nil
}

func (h *UserHandler) Login(ctx context.Context, req apigen.LoginRequestObject) (apigen.LoginResponseObject, error) {
	if req.Body == nil {
		return apigen.Login400JSONResponse{Error: "invalid_request", Message: "request body is required"}, nil
	}

	result, err := h.userService.Login(ctx, string(req.Body.Email), req.Body.Password)
	if err != nil {
		return loginErrorResponse(err), nil
	}

	return apigen.Login200JSONResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		Role:      result.Role,
		UserId:    result.UserID,
		Email:     result.Email,
	}, nil
}

func registerErrorResponse(err error) apigen.RegisterResponseObject {
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		switch appErr.StatusCode {
		case http.StatusConflict:
			return apigen.Register409JSONResponse{Error: appErr.Code, Message: appErr.Message}
		case http.StatusUnauthorized:
			return apigen.Register401JSONResponse{Error: appErr.Code, Message: appErr.Message}
		default:
			return apigen.Register500JSONResponse{Error: appErr.Code, Message: appErr.Message}
		}
	}

	return apigen.Register500JSONResponse{Error: "internal_server_error", Message: err.Error()}
}

func loginErrorResponse(err error) apigen.LoginResponseObject {
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		switch appErr.StatusCode {
		case http.StatusUnauthorized:
			return apigen.Login401JSONResponse{Error: appErr.Code, Message: appErr.Message}
		case http.StatusConflict:
			return apigen.Login409JSONResponse{Error: appErr.Code, Message: appErr.Message}
		default:
			return apigen.Login500JSONResponse{Error: appErr.Code, Message: appErr.Message}
		}
	}

	return apigen.Login500JSONResponse{Error: "internal_server_error", Message: err.Error()}
}

func (h *UserHandler) CreateInvoice(ctx context.Context, req apigen.CreateInvoiceRequestObject) (apigen.CreateInvoiceResponseObject, error) {
	if req.Body == nil {
		return apigen.CreateInvoice400JSONResponse{Error: "invalid_request", Message: "request body is required"}, nil
	}
	if req.Body.MerchantId == "" || req.Body.Amount <= 0 {
		return apigen.CreateInvoice400JSONResponse{Error: "invalid_request", Message: "merchantId and positive amount are required"}, nil
	}

	inv, err := h.invoiceService.Create(ctx, appInvoice.CreateInvoiceInput{
		MerchantID: req.Body.MerchantId,
		Amount:     req.Body.Amount,
		Currency:   valueOrEmpty(req.Body.Currency),
	})
	if err != nil {
		return apigen.CreateInvoice500JSONResponse{Error: "internal_server_error", Message: err.Error()}, nil
	}

	return apigen.CreateInvoice201JSONResponse{
		Id:            inv.ID,
		InvoiceNumber: inv.InvoiceNumber,
		MerchantId:    inv.MerchantID,
		Amount:        inv.Amount,
		Currency:      inv.Currency,
		Status:        inv.Status,
		PaymentToken:  inv.PaymentToken,
		CreatedAt:     inv.CreatedAt,
	}, nil
}

func (h *UserHandler) ListInvoices(ctx context.Context, req apigen.ListInvoicesRequestObject) (apigen.ListInvoicesResponseObject, error) {
	page := 1
	pageSize := 10
	if req.Params.Page != nil {
		if p, err := strconv.Atoi(*req.Params.Page); err == nil {
			page = p
		}
	}
	if req.Params.PageSize != nil {
		if ps, err := strconv.Atoi(*req.Params.PageSize); err == nil {
			pageSize = ps
		}
	}

	items, err := h.invoiceService.List(ctx, appInvoice.ListFilter{
		MerchantID: valueOrEmpty(req.Params.MerchantId),
		Status:     valueOrEmpty(req.Params.Status),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		return apigen.ListInvoices500JSONResponse{Error: "internal_server_error", Message: err.Error()}, nil
	}

	respItems := make([]apigen.InvoiceResponse, 0, len(items))
	for _, inv := range items {
		respItems = append(respItems, apigen.InvoiceResponse{
			Id:            inv.ID,
			InvoiceNumber: inv.InvoiceNumber,
			MerchantId:    inv.MerchantID,
			Amount:        inv.Amount,
			Currency:      inv.Currency,
			Status:        inv.Status,
			PaymentToken:  inv.PaymentToken,
			CreatedAt:     inv.CreatedAt,
		})
	}

	return apigen.ListInvoices200JSONResponse{
		Items: respItems,
		Page:  page,
		Size:  pageSize,
	}, nil
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func roleValue(v *apigen.RegisterRequestRole) string {
	if v == nil {
		return ""
	}
	return string(*v)
}

// RegisterGin handles POST /auth/register in plain Gin mode.
func (h *UserHandler) RegisterGin(c *gin.Context) {
	var body apigen.RegisterRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.userService.Register(c.Request.Context(), body.Name, string(body.Email), body.Password, roleValue(body.Role))
	if err != nil {
		writeRegisterError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"token":     result.Token,
		"expiresAt": result.ExpiresAt,
		"userId":    result.UserID,
		"email":     result.Email,
		"role":      result.Role,
	})
}

// LoginGin handles POST /auth/login in plain Gin mode.
func (h *UserHandler) LoginGin(c *gin.Context) {
	var body apigen.LoginRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.userService.Login(c.Request.Context(), string(body.Email), body.Password)
	if err != nil {
		writeLoginError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":     result.Token,
		"expiresAt": result.ExpiresAt,
		"userId":    result.UserID,
		"email":     result.Email,
		"role":      result.Role,
	})
}

// RefreshGin handles POST /auth/refresh in plain Gin mode.
func (h *UserHandler) RefreshGin(c *gin.Context) {
	userIDRaw, exists := c.Get(middleware.ContextUserIDKey)
	if !exists {
		respondError(c, http.StatusUnauthorized, "auth_error", "missing auth context")
		return
	}
	userID, ok := userIDRaw.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		respondError(c, http.StatusUnauthorized, "auth_error", "invalid auth context")
		return
	}
	result, err := h.userService.Refresh(c.Request.Context(), userID)
	if err != nil {
		writeLoginError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":     result.Token,
		"expiresAt": result.ExpiresAt,
		"userId":    result.UserID,
		"email":     result.Email,
		"role":      result.Role,
	})
}

// CreateInvoiceGin handles POST /invoices in plain Gin mode.
func (h *UserHandler) CreateInvoiceGin(c *gin.Context) {
	var body apigen.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if body.Amount <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "amount must be positive")
		return
	}
	userID, role, ok := authIdentity(c)
	if !ok {
		return
	}
	merchantID := body.MerchantId
	if role == domainUser.RoleMerchant {
		if merchantID == "" {
			merchantID = userID
		}
		if merchantID != userID {
			respondError(c, http.StatusForbidden, "forbidden_error", "merchant can only create own invoice")
			return
		}
	}
	inv, err := h.invoiceService.Create(c.Request.Context(), appInvoice.CreateInvoiceInput{
		MerchantID: merchantID,
		Amount:     body.Amount,
		Currency:   valueOrEmpty(body.Currency),
		DueDate:    body.DueDate,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"id":            inv.ID,
		"invoiceNumber": inv.InvoiceNumber,
		"merchantId":    inv.MerchantID,
		"amount":        inv.Amount,
		"currency":      inv.Currency,
		"status":        inv.Status,
		"paymentToken":  inv.PaymentToken,
		"dueDate":       inv.DueDate,
		"createdAt":     inv.CreatedAt,
	})
}

// ListInvoicesGin handles GET /invoices in plain Gin mode.
func (h *UserHandler) ListInvoicesGin(c *gin.Context) {
	page := 1
	pageSize := 10
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			page = p
		}
	}
	if v := c.Query("pageSize"); v != "" {
		if ps, err := strconv.Atoi(v); err == nil {
			pageSize = ps
		}
	}
	userID, role, ok := authIdentity(c)
	if !ok {
		return
	}
	merchantID := c.Query("merchantId")
	if role == domainUser.RoleMerchant {
		merchantID = userID
	}
	items, err := h.invoiceService.List(c.Request.Context(), appInvoice.ListFilter{
		MerchantID: merchantID,
		Status:     c.Query("status"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
		return
	}
	respItems := make([]gin.H, 0, len(items))
	for _, inv := range items {
		respItems = append(respItems, gin.H{
			"id":            inv.ID,
			"invoiceNumber": inv.InvoiceNumber,
			"merchantId":    inv.MerchantID,
			"amount":        inv.Amount,
			"currency":      inv.Currency,
			"status":        inv.Status,
			"paymentToken":  inv.PaymentToken,
			"dueDate":       inv.DueDate,
			"createdAt":     inv.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"items": respItems,
		"page":  page,
		"size":  pageSize,
		"total": len(respItems),
	})
}

func (h *UserHandler) GetInvoiceGin(c *gin.Context) {
	userID, role, ok := authIdentity(c)
	if !ok {
		return
	}
	inv, err := h.invoiceService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "not_found", err.Error())
		return
	}
	if role == domainUser.RoleMerchant && inv.MerchantID != userID {
		respondError(c, http.StatusForbidden, "forbidden_error", "merchant can only access own invoice")
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *UserHandler) UpdateInvoiceGin(c *gin.Context) {
	userID, role, ok := authIdentity(c)
	if !ok {
		return
	}
	existing, err := h.invoiceService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "not_found", err.Error())
		return
	}
	if role == domainUser.RoleMerchant && existing.MerchantID != userID {
		respondError(c, http.StatusForbidden, "forbidden_error", "merchant can only update own invoice")
		return
	}
	var req struct {
		Amount   int64   `json:"amount"`
		Currency *string `json:"currency"`
		DueDate  *string `json:"dueDate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.Amount <= 0 {
		respondError(c, http.StatusBadRequest, "invalid_request", "amount must be > 0")
		return
	}
	currency := existing.Currency
	if req.Currency != nil && strings.TrimSpace(*req.Currency) != "" {
		currency = *req.Currency
	}
	dueDate := existing.DueDate
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			respondError(c, http.StatusBadRequest, "invalid_request", "dueDate must be RFC3339")
			return
		}
		dueDate = t
	}
	inv, err := h.invoiceService.Update(c.Request.Context(), existing.ID, appInvoice.UpdateInvoiceInput{
		Amount:   req.Amount,
		Currency: currency,
		DueDate:  &dueDate,
	})
	if err != nil {
		respondError(c, http.StatusBadRequest, "invoice_error", err.Error())
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *UserHandler) DeleteInvoiceGin(c *gin.Context) {
	userID, role, ok := authIdentity(c)
	if !ok {
		return
	}
	existing, err := h.invoiceService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		respondError(c, http.StatusNotFound, "not_found", err.Error())
		return
	}
	if role == domainUser.RoleMerchant && existing.MerchantID != userID {
		respondError(c, http.StatusForbidden, "forbidden_error", "merchant can only delete own invoice")
		return
	}
	if err := h.invoiceService.Delete(c.Request.Context(), existing.ID); err != nil {
		respondError(c, http.StatusBadRequest, "invoice_error", err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func authIdentity(c *gin.Context) (string, string, bool) {
	userIDRaw, ok := c.Get(middleware.ContextUserIDKey)
	if !ok {
		respondError(c, http.StatusUnauthorized, "auth_error", "missing auth context")
		return "", "", false
	}
	roleRaw, ok := c.Get(middleware.ContextRoleKey)
	if !ok {
		respondError(c, http.StatusUnauthorized, "auth_error", "missing auth context")
		return "", "", false
	}
	userID, ok := userIDRaw.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		respondError(c, http.StatusUnauthorized, "auth_error", "invalid auth context")
		return "", "", false
	}
	role, ok := roleRaw.(string)
	if !ok || strings.TrimSpace(role) == "" {
		respondError(c, http.StatusUnauthorized, "auth_error", "invalid auth context")
		return "", "", false
	}
	return userID, role, true
}

func writeRegisterError(c *gin.Context, err error) {
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		respondError(c, appErr.StatusCode, appErr.Code, appErr.Message)
		return
	}
	respondError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
}

func writeLoginError(c *gin.Context, err error) {
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		respondError(c, appErr.StatusCode, appErr.Code, appErr.Message)
		return
	}
	respondError(c, http.StatusInternalServerError, "internal_server_error", err.Error())
}
