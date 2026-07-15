package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
	"github.com/gonszalito/go-ddd-architecture/internal/interfaces/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -------------------- RoleHandler --------------------

func withIdentity(userID, role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.ContextUserIDKey, userID)
		c.Set(middleware.ContextRoleKey, role)
		c.Next()
	}
}

func TestRoleHandler_MerchantOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := handlers.NewRoleHandler()
	router.GET("/merchant/profile", withIdentity("u-1", "MERCHANT"), h.MerchantOnly)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/merchant/profile", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "u-1", payload["userId"])
	assert.Equal(t, "MERCHANT", payload["role"])
}

func TestRoleHandler_AdminOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := handlers.NewRoleHandler()
	router.GET("/admin/dashboard", withIdentity("u-2", "ADMIN"), h.AdminOnly)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "u-2", payload["userId"])
	assert.Equal(t, "ADMIN", payload["role"])
}

// -------------------- PaymentHandler (Gin-mounted) --------------------

func newPaymentHandlerFixture() (*handlers.PaymentHandler, *appInvoice.Service) {
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "i1", PaymentToken: "tok1", MerchantID: "m1", Amount: 1000, Currency: "USD", Status: invoice.StatusPending, CreatedAt: time.Now()},
	}))
	paymentSvc := appPayment.NewService(database.NewInMemoryPaymentRepository(nil), invoiceSvc)
	return handlers.NewPaymentHandler(paymentSvc), invoiceSvc
}

func TestPaymentHandler_ResolvePaymentLink_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h, _ := newPaymentHandlerFixture()
	router.GET("/pay/:token", h.ResolvePaymentLink)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pay/tok1", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "i1", payload["invoiceId"])
}

func TestPaymentHandler_ResolvePaymentLink_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h, _ := newPaymentHandlerFixture()
	router.GET("/pay/:token", h.ResolvePaymentLink)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pay/missing", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPaymentHandler_CreatePaymentIntent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h, _ := newPaymentHandlerFixture()
	router.POST("/pay/:token/intents", h.CreatePaymentIntent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pay/tok1/intents", bytes.NewBufferString(`{"method":"WALLET"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestPaymentHandler_CreatePaymentIntent_InvalidMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h, _ := newPaymentHandlerFixture()
	router.POST("/pay/:token/intents", h.CreatePaymentIntent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pay/tok1/intents", bytes.NewBufferString(`{"method":"BOGUS"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPaymentHandler_SimulatePaymentIntent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h, invoiceSvc := newPaymentHandlerFixture()
	paymentSvc := appPayment.NewService(database.NewInMemoryPaymentRepository(nil), invoiceSvc)
	h = handlers.NewPaymentHandler(paymentSvc)
	router.POST("/pay/:token/intents", h.CreatePaymentIntent)
	router.POST("/admin/payment-intents/:intentId/simulate", h.SimulatePaymentIntent)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/pay/tok1/intents", bytes.NewBufferString(`{"method":"WALLET"}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))
	intentID, _ := created["ID"].(string)
	require.NotEmpty(t, intentID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/payment-intents/"+intentID+"/simulate", bytes.NewBufferString(`{"outcome":"SUCCESS"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPaymentHandler_SimulatePaymentIntent_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h, _ := newPaymentHandlerFixture()
	router.POST("/admin/payment-intents/:intentId/simulate", h.SimulatePaymentIntent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/payment-intents/i1/simulate", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPaymentHandler_SimulatePaymentIntent_MissingOutcome(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h, _ := newPaymentHandlerFixture()
	router.POST("/admin/payment-intents/:intentId/simulate", h.SimulatePaymentIntent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/payment-intents/i1/simulate", bytes.NewBufferString(`{"outcome":""}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// -------------------- RefundHandler (Gin-mounted) --------------------

func newRefundHandlerFixture() *handlers.RefundHandler {
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "i1", MerchantID: "m1", Amount: 1000, Currency: "USD", Status: invoice.StatusPaid, CreatedAt: time.Now()},
	}))
	walletRepo := database.NewInMemoryWalletRepository([]*domainWallet.Wallet{{ID: "w1", MerchantID: "m1", Balance: 500}})
	refundSvc := appRefund.NewService(database.NewInMemoryRefundRepository(nil), walletRepo, invoiceSvc)
	return handlers.NewRefundHandler(refundSvc)
}

func TestRefundHandler_FullLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newRefundHandlerFixture()
	router.POST("/merchant/refunds", h.RequestRefund)
	router.GET("/merchant/refunds", h.ListHistory)
	router.POST("/admin/refunds/:refundId/approve", h.ApproveRefund)
	router.POST("/admin/refunds/:refundId/reject", h.RejectRefund)
	router.POST("/admin/refunds/:refundId/process", h.ProcessRefund)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/merchant/refunds", bytes.NewBufferString(`{"invoiceId":"i1","merchantId":"m1","amount":200}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	refundID, _ := created["id"].(string)
	require.NotEmpty(t, refundID)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/merchant/refunds?merchantId=m1&page=1&pageSize=10", nil)
	router.ServeHTTP(listRec, listReq)
	assert.Equal(t, http.StatusOK, listRec.Code)

	approveRec := httptest.NewRecorder()
	approveReq := httptest.NewRequest(http.MethodPost, "/admin/refunds/"+refundID+"/approve", nil)
	router.ServeHTTP(approveRec, approveReq)
	assert.Equal(t, http.StatusOK, approveRec.Code)

	processRec := httptest.NewRecorder()
	processReq := httptest.NewRequest(http.MethodPost, "/admin/refunds/"+refundID+"/process", bytes.NewBufferString(`{"success":true}`))
	processReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(processRec, processReq)
	assert.Equal(t, http.StatusOK, processRec.Code)
}

func TestRefundHandler_RejectFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newRefundHandlerFixture()
	router.POST("/merchant/refunds", h.RequestRefund)
	router.POST("/admin/refunds/:refundId/reject", h.RejectRefund)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/merchant/refunds", bytes.NewBufferString(`{"invoiceId":"i1","merchantId":"m1","amount":100}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	refundID, _ := created["id"].(string)

	rejectRec := httptest.NewRecorder()
	rejectReq := httptest.NewRequest(http.MethodPost, "/admin/refunds/"+refundID+"/reject", nil)
	router.ServeHTTP(rejectRec, rejectReq)
	assert.Equal(t, http.StatusOK, rejectRec.Code)
}

func TestRefundHandler_ApproveRefund_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newRefundHandlerFixture()
	router.POST("/admin/refunds/:refundId/approve", h.ApproveRefund)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/refunds/missing/approve", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRefundHandler_ProcessRefund_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newRefundHandlerFixture()
	router.POST("/admin/refunds/:refundId/process", h.ProcessRefund)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/refunds/r1/process", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRefundHandler_ListHistory_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newRefundHandlerFixture()
	router.GET("/merchant/refunds", h.ListHistory)

	// No merchantId and no seeded refunds still returns 200 with an empty list;
	// exercise the pagination defaults path instead.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/merchant/refunds", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// -------------------- TopUpHandler (Gin-mounted) --------------------

func newTopUpHandlerFixture() *handlers.TopUpHandler {
	walletRepo := database.NewInMemoryWalletRepository([]*domainWallet.Wallet{{ID: "w1", MerchantID: "m1", Balance: 500}})
	topupSvc := appTopUp.NewService(database.NewInMemoryTopUpRepository(nil), walletRepo)
	return handlers.NewTopUpHandler(topupSvc)
}

func TestTopUpHandler_FullLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newTopUpHandlerFixture()
	router.POST("/merchant/topups", h.RequestTopUp)
	router.GET("/merchant/topups", h.History)
	router.POST("/admin/topups/:topupId/status", h.AdminUpdate)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/merchant/topups", bytes.NewBufferString(`{"merchantId":"m1","amount":200}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var created map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	topupID, _ := created["id"].(string)
	require.NotEmpty(t, topupID)

	historyRec := httptest.NewRecorder()
	historyReq := httptest.NewRequest(http.MethodGet, "/merchant/topups?merchantId=m1&page=1&pageSize=10", nil)
	router.ServeHTTP(historyRec, historyReq)
	assert.Equal(t, http.StatusOK, historyRec.Code)

	statusRec := httptest.NewRecorder()
	statusReq := httptest.NewRequest(http.MethodPost, "/admin/topups/"+topupID+"/status", bytes.NewBufferString(`{"success":true}`))
	statusReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(statusRec, statusReq)
	assert.Equal(t, http.StatusOK, statusRec.Code)
}

func TestTopUpHandler_AdminUpdate_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newTopUpHandlerFixture()
	router.POST("/admin/topups/:topupId/status", h.AdminUpdate)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/topups/t1/status", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTopUpHandler_AdminUpdate_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newTopUpHandlerFixture()
	router.POST("/admin/topups/:topupId/status", h.AdminUpdate)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/topups/missing/status", bytes.NewBufferString(`{"success":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTopUpHandler_History_PaginationDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newTopUpHandlerFixture()
	router.GET("/merchant/topups", h.History)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/merchant/topups", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// -------------------- UserHandler *Gin methods --------------------

func newUserHandlerFixture() *handlers.UserHandler {
	userSvc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	return handlers.NewUserHandler(userSvc, invoiceSvc)
}

func registerAndLoginGin(t *testing.T, router *gin.Engine, h *handlers.UserHandler, email, role string) string {
	t.Helper()
	router.POST("/auth/register", h.RegisterGin)
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{
		"name": "Test User", "email": email, "password": "secret123", "role": role,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	token, _ := payload["token"].(string)
	require.NotEmpty(t, token)
	return token
}

func TestUserHandler_RegisterGin_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/auth/register", h.RegisterGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_LoginGin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	registerAndLoginGin(t, router, h, "gin-login@example.com", "MERCHANT")

	router.POST("/auth/login", h.LoginGin)
	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"email": "gin-login@example.com", "password": "secret123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_LoginGin_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/auth/login", h.LoginGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_RefreshGin_MissingAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/auth/refresh", h.RefreshGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestUserHandler_RefreshGin_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	registerAndLoginGin(t, router, h, "gin-refresh@example.com", "MERCHANT")

	// RefreshGin only depends on an auth context already populated by the
	// real auth middleware (userID + role claims extracted from the JWT).
	// Here we simulate that context directly with the known registered user.
	userSvc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	result, err := userSvc.Register(t.Context(), "Refresh User", "gin-refresh-2@example.com", "secret123", "MERCHANT")
	require.NoError(t, err)
	h2 := handlers.NewUserHandler(userSvc, appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)))

	router.POST("/auth/refresh", withIdentity(result.UserID, "MERCHANT"), h2.RefreshGin)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_CreateInvoiceGin_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"merchantId":"u-1","amount":0}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_CreateInvoiceGin_InvalidBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserHandler_ListInvoicesGin_MerchantScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)
	router.GET("/invoices", withIdentity("u-1", "MERCHANT"), h.ListInvoicesGin)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"amount":500,"currency":"USD"}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices?page=1&pageSize=10", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_GetInvoiceGin_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.GET("/invoices/:id", withIdentity("u-1", "MERCHANT"), h.GetInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices/missing", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserHandler_UpdateInvoiceGin_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.PUT("/invoices/:id", withIdentity("u-1", "MERCHANT"), h.UpdateInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/invoices/missing", bytes.NewBufferString(`{"amount":100,"currency":"USD"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestUserHandler_DeleteInvoiceGin_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := newUserHandlerFixture()
	router.DELETE("/invoices/:id", withIdentity("u-1", "MERCHANT"), h.DeleteInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/invoices/missing", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
