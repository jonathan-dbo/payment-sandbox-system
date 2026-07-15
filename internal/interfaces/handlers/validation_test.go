package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	appDashboard "github.com/gonszalito/go-ddd-architecture/internal/application/dashboard"
	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/gonszalito/go-ddd-architecture/internal/interfaces/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationErrorResponsePaymentIntentBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "i1", PaymentToken: "tok1", MerchantID: "m1", Amount: 1000, Currency: "USD", Status: invoice.StatusPending, CreatedAt: time.Now()},
	}))
	paymentH := handlers.NewPaymentHandler(appPayment.NewService(database.NewInMemoryPaymentRepository(nil), invoiceSvc))
	router.POST("/pay/:token/intents", paymentH.CreatePaymentIntent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pay/tok1/intents", bytes.NewBufferString(`{"method":""}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assertErrorSchema(t, rec.Body.Bytes())
}

func TestValidationErrorResponsePaymentIntentEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "i1", PaymentToken: "tok1", MerchantID: "m1", Amount: 1000, Currency: "USD", Status: invoice.StatusPending, CreatedAt: time.Now()},
	}))
	paymentH := handlers.NewPaymentHandler(appPayment.NewService(database.NewInMemoryPaymentRepository(nil), invoiceSvc))
	router.POST("/pay/:token/intents", paymentH.CreatePaymentIntent)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pay/tok1/intents", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "invalid_request", payload["error"])
	assert.Equal(t, "request body is required", payload["message"])
}

func TestValidationErrorResponseRefundBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	refundH := handlers.NewRefundHandler(appRefund.NewService(database.NewInMemoryRefundRepository(nil), database.NewInMemoryWalletRepository(nil), invoiceSvc))
	router.POST("/merchant/refunds", refundH.RequestRefund)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/merchant/refunds", bytes.NewBufferString(`{"invoiceId":"i1","merchantId":"m1","amount":0}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assertErrorSchema(t, rec.Body.Bytes())
}

func TestValidationErrorResponseTopUpBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	topupH := handlers.NewTopUpHandler(appTopUp.NewService(database.NewInMemoryTopUpRepository(nil), database.NewInMemoryWalletRepository(nil)))
	router.POST("/merchant/topups", topupH.RequestTopUp)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/merchant/topups", bytes.NewBufferString(`{"merchantId":"","amount":10}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assertErrorSchema(t, rec.Body.Bytes())
}

func TestValidationErrorResponseDashboardDateBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	dashboardH := handlers.NewDashboardHandler(appDashboard.NewService(
		database.NewInMemoryInvoiceRepository(nil),
		database.NewInMemoryPaymentRepository(nil),
		database.NewInMemoryRefundRepository([]*refund.Refund{}),
	))
	router.GET("/admin/dashboard/stats", dashboardH.Stats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats?startDate=bad-date", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assertErrorSchema(t, rec.Body.Bytes())
}

func assertErrorSchema(t *testing.T, body []byte) {
	t.Helper()
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	_, hasError := payload["error"]
	_, hasMessage := payload["message"]
	assert.True(t, hasError)
	assert.True(t, hasMessage)
}
