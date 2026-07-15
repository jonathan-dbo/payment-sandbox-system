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

type endpointFixture struct {
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

func TestEndpointMatrixHappyPaths(t *testing.T) {
	fx := newEndpointFixture(t)

	resp := doJSON(t, fx.router, http.MethodGet, "/health", "", nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/auth/login", "", map[string]any{
		"email":    "merchant@example.com",
		"password": "secret123",
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/auth/refresh", fx.merchantToken, map[string]any{})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/merchant/profile", fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/admin/dashboard", fx.adminToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/pay/"+fx.paymentToken, "", nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/pay/"+fx.paymentToken+"/intents", "", map[string]any{
		"method": "WALLET",
	})
	assert.Equal(t, http.StatusCreated, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/invoices?page=1&pageSize=10", fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/invoices/"+fx.pendingInvoice, fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPut, "/invoices/"+fx.pendingInvoice, fx.merchantToken, map[string]any{
		"amount":   12000,
		"currency": "USD",
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/merchant/refunds", fx.merchantToken, map[string]any{
		"invoiceId":  fx.paidInvoiceID,
		"merchantId": fx.merchantID,
		"amount":     2000,
	})
	assert.Equal(t, http.StatusCreated, resp.Code)
	refundID := mustIDFromBody(t, resp.Body.Bytes())

	resp = doJSON(t, fx.router, http.MethodGet, "/merchant/refunds?merchantId="+fx.merchantID, fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/merchant/topups", fx.merchantToken, map[string]any{
		"merchantId": fx.merchantID,
		"amount":     1000,
	})
	assert.Equal(t, http.StatusCreated, resp.Code)
	topupID := mustIDFromBody(t, resp.Body.Bytes())

	resp = doJSON(t, fx.router, http.MethodGet, "/merchant/topups?merchantId="+fx.merchantID, fx.merchantToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/admin/payment-intents/"+fx.intentID+"/simulate", fx.adminToken, map[string]any{
		"outcome": "FAILED",
	})
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
		"amount":     1000,
	})
	assert.Equal(t, http.StatusCreated, resp.Code)
	rejectID := mustIDFromBody(t, resp.Body.Bytes())
	resp = doJSON(t, fx.router, http.MethodPost, "/admin/refunds/"+rejectID+"/reject", fx.adminToken, map[string]any{})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/admin/topups/"+topupID+"/status", fx.adminToken, map[string]any{
		"success": true,
	})
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/admin/dashboard/stats", fx.adminToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/invoices", fx.merchantToken, map[string]any{
		"merchantId": fx.merchantID,
		"amount":     5000,
		"currency":   "USD",
	})
	assert.Equal(t, http.StatusCreated, resp.Code)
	deleteID := mustIDFromBody(t, resp.Body.Bytes())
	resp = doJSON(t, fx.router, http.MethodDelete, "/invoices/"+deleteID, fx.merchantToken, nil)
	assert.Equal(t, http.StatusNoContent, resp.Code)
}

func TestEndpointMatrixErrorPaths(t *testing.T) {
	fx := newEndpointFixture(t)

	resp := doJSON(t, fx.router, http.MethodPost, "/auth/refresh", "", map[string]any{})
	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/invoices", fx.userToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/merchant/profile", fx.adminToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	resp = doJSON(t, fx.router, http.MethodGet, "/admin/dashboard", fx.merchantToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/pay/"+fx.paymentToken+"/intents", "", nil)
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/admin/payment-intents/"+fx.intentID+"/simulate", fx.adminToken, map[string]any{
		"outcome": "INVALID",
	})
	assert.Equal(t, http.StatusBadRequest, resp.Code)

	resp = doJSON(t, fx.router, http.MethodPost, "/merchant/refunds", fx.merchantToken, map[string]any{
		"invoiceId":  fx.paidInvoiceID,
		"merchantId": fx.merchantID,
		"amount":     999999999,
	})
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func newEndpointFixture(t *testing.T) *endpointFixture {
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

	merchant, err := userSvc.Register(context.Background(), "Merchant", "merchant@example.com", "secret123", "MERCHANT")
	require.NoError(t, err)
	admin, err := userSvc.Register(context.Background(), "Admin", "admin@example.com", "secret123", "ADMIN")
	require.NoError(t, err)
	userOnly, err := userSvc.Register(context.Background(), "User", "user@example.com", "secret123", "USER")
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

	router := infraHTTP.NewRouterCodegen(infraHTTP.RouterDependencies{
		UserService:    userSvc,
		InvoiceService: invoiceSvc,
		PaymentService: paymentSvc,
		RefundService:  refundSvc,
		TopUpService:   topupSvc,
		Dashboard:      dashboardSvc,
		JWTSecret:      "test-secret",
	})

	return &endpointFixture{
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

func doJSON(t *testing.T, r http.Handler, method, path, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body []byte
	if payload != nil {
		b, err := json.Marshal(payload)
		require.NoError(t, err)
		body = b
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func mustIDFromBody(t *testing.T, body []byte) string {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	id, _ := payload["id"].(string)
	require.NotEmpty(t, id)
	return id
}

func timePtr(v time.Time) *time.Time { return &v }
