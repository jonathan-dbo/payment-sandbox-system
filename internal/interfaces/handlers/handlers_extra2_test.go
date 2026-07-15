package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appDashboard "github.com/gonszalito/go-ddd-architecture/internal/application/dashboard"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/gonszalito/go-ddd-architecture/internal/interfaces/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -------------------- writeRegisterError / writeLoginError AppError paths --------------------

func TestUserHandler_RegisterGin_DuplicateEmailAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/auth/register", h.RegisterGin)

	body, _ := json.Marshal(map[string]any{"name": "A", "email": "dup@example.com", "password": "secret123", "role": "MERCHANT"})
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusCreated, rec1.Code)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusConflict, rec2.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &payload))
	assert.Equal(t, "conflict_error", payload["error"])
}

func TestUserHandler_LoginGin_InvalidCredentialsAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	registerAndLoginGin(t, router, h, "login-apperr@example.com", "MERCHANT")

	router.POST("/auth/login", h.LoginGin)
	body, _ := json.Marshal(map[string]any{"email": "login-apperr@example.com", "password": "wrong-password"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "auth_error", payload["error"])
}

func TestUserHandler_RefreshGin_InvalidCredentialsAppError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/auth/refresh", withIdentity("missing-user-id", "MERCHANT"), h.RefreshGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// -------------------- DashboardHandler.Stats --------------------

func TestDashboardHandler_Stats_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := handlers.NewDashboardHandler(appDashboard.NewService(
		database.NewInMemoryInvoiceRepository(nil),
		database.NewInMemoryPaymentRepository(nil),
		database.NewInMemoryRefundRepository([]*refund.Refund{}),
	))
	router.GET("/admin/dashboard/stats", h.Stats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats?merchantId=m1", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDashboardHandler_Stats_WithValidDateRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := handlers.NewDashboardHandler(appDashboard.NewService(
		database.NewInMemoryInvoiceRepository(nil),
		database.NewInMemoryPaymentRepository(nil),
		database.NewInMemoryRefundRepository([]*refund.Refund{}),
	))
	router.GET("/admin/dashboard/stats", h.Stats)

	start := url.QueryEscape(time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339))
	end := url.QueryEscape(time.Now().UTC().Format(time.RFC3339))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats?startDate="+start+"&endDate="+end, nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestDashboardHandler_Stats_InvalidEndDate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := handlers.NewDashboardHandler(appDashboard.NewService(
		database.NewInMemoryInvoiceRepository(nil),
		database.NewInMemoryPaymentRepository(nil),
		database.NewInMemoryRefundRepository([]*refund.Refund{}),
	))
	router.GET("/admin/dashboard/stats", h.Stats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats?endDate=not-a-date", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// -------------------- UpdateInvoiceGin / DeleteInvoiceGin happy paths & forbidden --------------------

func TestUserHandler_UpdateInvoiceGin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)
	router.PUT("/invoices/:id", withIdentity("u-1", "MERCHANT"), h.UpdateInvoiceGin)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"amount":500,"currency":"USD"}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	invoiceID, _ := created["id"].(string)
	require.NotEmpty(t, invoiceID)

	due := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invoices/"+invoiceID, bytes.NewBufferString(`{"amount":900,"currency":"EUR","dueDate":"`+due+`"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_UpdateInvoiceGin_ForbiddenForOtherMerchant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)
	router.PUT("/invoices/:id", withIdentity("u-2", "MERCHANT"), h.UpdateInvoiceGin)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"amount":500,"currency":"USD"}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	invoiceID, _ := created["id"].(string)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invoices/"+invoiceID, bytes.NewBufferString(`{"amount":900,"currency":"EUR"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUserHandler_DeleteInvoiceGin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)
	router.DELETE("/invoices/:id", withIdentity("u-1", "MERCHANT"), h.DeleteInvoiceGin)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"amount":500,"currency":"USD"}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	invoiceID, _ := created["id"].(string)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/invoices/"+invoiceID, nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestUserHandler_DeleteInvoiceGin_ForbiddenForOtherMerchant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)
	router.DELETE("/invoices/:id", withIdentity("u-2", "MERCHANT"), h.DeleteInvoiceGin)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"amount":500,"currency":"USD"}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	invoiceID, _ := created["id"].(string)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/invoices/"+invoiceID, nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUserHandler_GetInvoiceGin_ForbiddenForOtherMerchant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)
	router.GET("/invoices/:id", withIdentity("u-2", "MERCHANT"), h.GetInvoiceGin)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"amount":500,"currency":"USD"}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	invoiceID, _ := created["id"].(string)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices/"+invoiceID, nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// -------------------- RefundHandler.RejectRefund not-found path --------------------

func TestRefundHandler_RejectRefund_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newRefundHandlerFixture()
	router.POST("/admin/refunds/:refundId/reject", h.RejectRefund)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/refunds/missing/reject", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRefundHandler_ProcessRefund_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newRefundHandlerFixture()
	router.POST("/admin/refunds/:refundId/process", h.ProcessRefund)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/refunds/missing/process", bytes.NewBufferString(`{"success":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
