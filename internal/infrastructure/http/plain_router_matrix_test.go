package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appDashboard "github.com/gonszalito/go-ddd-architecture/internal/application/dashboard"
	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	infraHTTP "github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// plainRouterFixture mirrors endpointFixture but is built against NewRouter
// (plain Gin handler wiring: *Gin-suffixed UserHandler methods, RoleHandler,
// CORSMiddleware, StructuredLoggingMiddleware) instead of NewRouterCodegen,
// so this exercises the code paths the endpoint matrix / codegen tests don't.
type plainRouterFixture struct {
	router         http.Handler
	merchantToken  string
	adminToken     string
	userToken      string
	merchantID     string
	paidInvoiceID  string
	pendingInvoice string
	paymentToken   string
	intentID       string
}

func newPlainRouterFixture(t *testing.T) *plainRouterFixture {
	t.Helper()

	userRepo := database.NewInMemoryUserRepository(nil)
	invoiceRepo := database.NewInMemoryInvoiceRepository(nil)
	paymentRepo := database.NewInMemoryPaymentRepository(nil)
	refundRepo := database.NewInMemoryRefundRepository(nil)
	walletRepo := database.NewInMemoryWalletRepository(nil)
	topupRepo := database.NewInMemoryTopUpRepository(nil)

	userSvc := appUser.NewUserService(userRepo, "test-secret", 60)
	invoiceSvc := appInvoice.NewService(invoiceRepo)
	paymentSvc := appPayment.NewService(paymentRepo, invoiceSvc)
	refundSvc := appRefund.NewService(refundRepo, walletRepo, invoiceSvc)
	topupSvc := appTopUp.NewService(topupRepo, walletRepo)
	dashboardSvc := appDashboard.NewService(invoiceRepo, paymentRepo, refundRepo)

	merchant, err := userSvc.Register(context.Background(), "Merchant", "plain-merchant@example.com", "secret123", "MERCHANT")
	require.NoError(t, err)
	admin, err := userSvc.Register(context.Background(), "Admin", "plain-admin@example.com", "secret123", "ADMIN")
	require.NoError(t, err)
	userOnly, err := userSvc.Register(context.Background(), "User", "plain-user@example.com", "secret123", "USER")
	require.NoError(t, err)

	paidInv, err := invoiceSvc.Create(context.Background(), appInvoice.CreateInvoiceInput{
		MerchantID: merchant.UserID,
		Amount:     10000,
		Currency:   "USD",
	})
	require.NoError(t, err)
	require.NoError(t, paidInv.MarkPaid())
	require.NoError(t, invoiceSvc.Save(context.Background(), paidInv))

	pendingInv, err := invoiceSvc.Create(context.Background(), appInvoice.CreateInvoiceInput{
		MerchantID: merchant.UserID,
		Amount:     11000,
		Currency:   "USD",
		DueDate:    timePtr(time.Now().UTC().Add(2 * time.Hour)),
	})
	require.NoError(t, err)

	intent, err := paymentSvc.CreateIntent(context.Background(), pendingInv.PaymentToken, "WALLET")
	require.NoError(t, err)

	router := infraHTTP.NewRouter(infraHTTP.RouterDependencies{
		UserService:        userSvc,
		InvoiceService:     invoiceSvc,
		PaymentService:     paymentSvc,
		RefundService:      refundSvc,
		TopUpService:       topupSvc,
		Dashboard:          dashboardSvc,
		JWTSecret:          "test-secret",
		CORSAllowedOrigins: []string{"https://allowed.example.com"},
	})

	return &plainRouterFixture{
		router:         router,
		merchantToken:  merchant.Token,
		adminToken:     admin.Token,
		userToken:      userOnly.Token,
		merchantID:     merchant.UserID,
		paidInvoiceID:  paidInv.ID,
		pendingInvoice: pendingInv.ID,
		paymentToken:   pendingInv.PaymentToken,
		intentID:       intent.ID,
	}
}

func TestPlainRouterHappyPaths(t *testing.T) {
	fx := newPlainRouterFixture(t)

	resp := doJSON(t, fx.router, http.MethodGet, "/health", "", nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// CORS preflight and allowed-origin behavior via the router's registered middleware.
	preflight := httptest.NewRequest(http.MethodOptions, "/health", nil)
	preflight.Header.Set("Origin", "https://allowed.example.com")
	preflightRec := httptest.NewRecorder()
	fx.router.ServeHTTP(preflightRec, preflight)
	assert.Equal(t, http.StatusNoContent, preflightRec.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/auth/register", "", map[string]any{
		"name":     "New Merchant",
		"email":    "new-plain-merchant@example.com",
		"password": "secret123",
		"role":     "MERCHANT",
	})
	assert.Equal(t, http.StatusCreated, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/auth/login", "", map[string]any{
		"email":    "plain-merchant@example.com",
		"password": "secret123",
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/auth/refresh", fx.merchantToken, map[string]any{})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/pay/"+fx.paymentToken, "", nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/pay/"+fx.paymentToken+"/intents", "", map[string]any{
		"method": "WALLET",
	})
	assert.Equal(t, http.StatusCreated, resp.Code)

	// Invoice CRUD via *Gin-suffixed handlers (plain Gin mode).
	resp = doJSON(t, fx.router, http.MethodPost, "/invoices", fx.merchantToken, map[string]any{
		"merchantId": fx.merchantID,
		"amount":     4000,
		"currency":   "USD",
	})
	require.Equal(t, http.StatusCreated, resp.Code)
	createdID := mustIDFromBody(t, resp.Body.Bytes())

	resp = doJSON(t, fx.router, http.MethodGet, "/invoices?page=1&pageSize=10", fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/invoices/"+createdID, fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPut, "/invoices/"+createdID, fx.merchantToken, map[string]any{
		"amount":   9000,
		"currency": "EUR",
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodDelete, "/invoices/"+createdID, fx.merchantToken, nil)
	assert.Equal(t, http.StatusNoContent, resp.Code)

	// Admin invoice access as non-owning role should still succeed (admin bypasses ownership check).
	resp = doJSON(t, fx.router, http.MethodGet, "/invoices/"+fx.paidInvoiceID, fx.adminToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Role-only endpoints wired via RoleHandler.
	resp = doJSON(t, fx.router, http.MethodGet, "/merchant/profile", fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
	resp = doJSON(t, fx.router, http.MethodGet, "/admin/dashboard", fx.adminToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	// Refund flow.
	resp = doJSON(t, fx.router, http.MethodPost, "/merchant/refunds", fx.merchantToken, map[string]any{
		"invoiceId":  fx.paidInvoiceID,
		"merchantId": fx.merchantID,
		"amount":     1500,
	})
	require.Equal(t, http.StatusCreated, resp.Code)
	refundID := mustIDFromBody(t, resp.Body.Bytes())

	resp = doJSON(t, fx.router, http.MethodGet, "/merchant/refunds?merchantId="+fx.merchantID, fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/admin/refunds/"+refundID+"/approve", fx.adminToken, map[string]any{})
	assert.Equal(t, http.StatusOK, resp.Code)
	resp = doJSON(t, fx.router, http.MethodPost, "/admin/refunds/"+refundID+"/process", fx.adminToken, map[string]any{
		"success": true,
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/merchant/refunds", fx.merchantToken, map[string]any{
		"invoiceId":  fx.paidInvoiceID,
		"merchantId": fx.merchantID,
		"amount":     500,
	})
	require.Equal(t, http.StatusCreated, resp.Code)
	rejectID := mustIDFromBody(t, resp.Body.Bytes())
	resp = doJSON(t, fx.router, http.MethodPost, "/admin/refunds/"+rejectID+"/reject", fx.adminToken, map[string]any{})
	assert.Equal(t, http.StatusOK, resp.Code)

	// TopUp flow.
	resp = doJSON(t, fx.router, http.MethodPost, "/merchant/topups", fx.merchantToken, map[string]any{
		"merchantId": fx.merchantID,
		"amount":     700,
	})
	require.Equal(t, http.StatusCreated, resp.Code)
	topupID := mustIDFromBody(t, resp.Body.Bytes())

	resp = doJSON(t, fx.router, http.MethodGet, "/merchant/topups?merchantId="+fx.merchantID, fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/admin/topups/"+topupID+"/status", fx.adminToken, map[string]any{
		"success": true,
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	// Payment intent admin simulation.
	resp = doJSON(t, fx.router, http.MethodPost, "/admin/payment-intents/"+fx.intentID+"/simulate", fx.adminToken, map[string]any{
		"outcome": "FAILED",
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/admin/dashboard/stats", fx.adminToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestPlainRouterErrorPaths(t *testing.T) {
	fx := newPlainRouterFixture(t)

	// Login failures.
	resp := doJSON(t, fx.router, http.MethodPost, "/auth/login", "", map[string]any{
		"email":    "plain-merchant@example.com",
		"password": "wrong-password",
	})
	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/auth/login", "", map[string]any{
		"email": "not-json-parseable",
	})
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// Register with duplicate email.
	resp = doJSON(t, fx.router, http.MethodPost, "/auth/register", "", map[string]any{
		"name":     "Dup",
		"email":    "plain-merchant@example.com",
		"password": "secret123",
		"role":     "MERCHANT",
	})
	assert.Equal(t, http.StatusConflict, resp.Code)

	// Register with invalid JSON body.
	badReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("not-json")))
	badReq.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	fx.router.ServeHTTP(badRec, badReq)
	assert.Equal(t, http.StatusBadRequest, badRec.Code)

	// Refresh without auth context.
	resp = doJSON(t, fx.router, http.MethodPost, "/auth/refresh", "", map[string]any{})
	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	// Invoice creation validation error (amount <= 0).
	resp = doJSON(t, fx.router, http.MethodPost, "/invoices", fx.merchantToken, map[string]any{
		"merchantId": fx.merchantID,
		"amount":     0,
		"currency":   "USD",
	})
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// Merchant attempting to create invoice for a different merchant ID.
	resp = doJSON(t, fx.router, http.MethodPost, "/invoices", fx.merchantToken, map[string]any{
		"merchantId": "someone-else",
		"amount":     1000,
		"currency":   "USD",
	})
	assert.Equal(t, http.StatusForbidden, resp.Code)

	// Merchant role forbidden from accessing another merchant's invoice.
	otherMerchant := doJSON(t, fx.router, http.MethodPost, "/auth/register", "", map[string]any{
		"name":     "Other Merchant",
		"email":    "other-plain-merchant@example.com",
		"password": "secret123",
		"role":     "MERCHANT",
	})
	require.Equal(t, http.StatusCreated, otherMerchant.Code)
	var otherBody map[string]any
	require.NoError(t, json.Unmarshal(otherMerchant.Body.Bytes(), &otherBody))
	otherToken, _ := otherBody["token"].(string)
	require.NotEmpty(t, otherToken)

	resp = doJSON(t, fx.router, http.MethodGet, "/invoices/"+fx.paidInvoiceID, otherToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPut, "/invoices/"+fx.paidInvoiceID, otherToken, map[string]any{
		"amount":   100,
		"currency": "USD",
	})
	assert.Equal(t, http.StatusForbidden, resp.Code)

	resp = doJSON(t, fx.router, http.MethodDelete, "/invoices/"+fx.paidInvoiceID, otherToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	// Update invoice not found.
	resp = doJSON(t, fx.router, http.MethodPut, "/invoices/missing-id", fx.merchantToken, map[string]any{
		"amount":   100,
		"currency": "USD",
	})
	assert.Equal(t, http.StatusNotFound, resp.Code)

	// Update invoice with invalid amount.
	resp = doJSON(t, fx.router, http.MethodPut, "/invoices/"+fx.pendingInvoice, fx.merchantToken, map[string]any{
		"amount":   0,
		"currency": "USD",
	})
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// Update invoice with invalid dueDate format.
	resp = doJSON(t, fx.router, http.MethodPut, "/invoices/"+fx.pendingInvoice, fx.merchantToken, map[string]any{
		"amount":   100,
		"currency": "USD",
		"dueDate":  "not-a-date",
	})
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// Get invoice not found.
	resp = doJSON(t, fx.router, http.MethodGet, "/invoices/missing-id", fx.merchantToken, nil)
	assert.Equal(t, http.StatusNotFound, resp.Code)

	// Delete invoice not found.
	resp = doJSON(t, fx.router, http.MethodDelete, "/invoices/missing-id", fx.merchantToken, nil)
	assert.Equal(t, http.StatusNotFound, resp.Code)

	// User role forbidden from invoice routes (RequireRole rejects USER).
	resp = doJSON(t, fx.router, http.MethodGet, "/invoices", fx.userToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	// Payment link resolve with unknown token.
	resp = doJSON(t, fx.router, http.MethodGet, "/pay/unknown-token", "", nil)
	assert.Equal(t, http.StatusNotFound, resp.Code)

	// Create payment intent with missing body.
	emptyBodyReq := httptest.NewRequest(http.MethodPost, "/pay/"+fx.paymentToken+"/intents", nil)
	emptyBodyRec := httptest.NewRecorder()
	fx.router.ServeHTTP(emptyBodyRec, emptyBodyReq)
	assert.Equal(t, http.StatusBadRequest, emptyBodyRec.Code)

	// Refund request validation error.
	resp = doJSON(t, fx.router, http.MethodPost, "/merchant/refunds", fx.merchantToken, map[string]any{
		"invoiceId":  "",
		"merchantId": fx.merchantID,
		"amount":     100,
	})
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// TopUp request validation error.
	resp = doJSON(t, fx.router, http.MethodPost, "/merchant/topups", fx.merchantToken, map[string]any{
		"merchantId": fx.merchantID,
		"amount":     0,
	})
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	// Admin dashboard forbidden for merchant.
	resp = doJSON(t, fx.router, http.MethodGet, "/admin/dashboard", fx.merchantToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	// Merchant profile forbidden for admin.
	resp = doJSON(t, fx.router, http.MethodGet, "/merchant/profile", fx.adminToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}
