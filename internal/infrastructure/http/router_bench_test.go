package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// benchFixture is a minimal wrapper so benchmark helpers can reuse the richer
// endpointFixture (tokens, seeded invoices) built for the endpoint matrix
// tests without pulling in *testing.T-only assertions.
type benchFixture = endpointFixture

// newBenchEndpointFixture adapts newEndpointFixture (which requires *testing.T)
// for use inside Benchmark* functions by feeding it a throwaway *testing.T.
func newBenchEndpointFixture(b *testing.B) *benchFixture {
	b.Helper()
	return newEndpointFixture(&testing.T{})
}

// silenceGinAndLog puts gin in release mode and discards structured log
// output (both the standard library log package and gin's own request
// logger) for the duration of a benchmark, so per-request logging I/O
// doesn't distort HTTP throughput/allocation measurements.
func silenceGinAndLog(b *testing.B) {
	b.Helper()
	prevGinMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	prevGinWriter := gin.DefaultWriter
	gin.DefaultWriter = io.Discard
	prevLogOut := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() {
		gin.SetMode(prevGinMode)
		gin.DefaultWriter = prevGinWriter
		log.SetOutput(prevLogOut)
	})
}

// BenchmarkHTTPHealth measures the unauthenticated /health endpoint, the
// simplest request in the router (no middleware-heavy auth/role checks).
func BenchmarkHTTPHealth(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

// BenchmarkHTTPLogin measures the /auth/login endpoint end-to-end through
// gin routing, JSON binding, password verification, and JWT issuance.
func BenchmarkHTTPLogin(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)
	body, err := json.Marshal(map[string]any{
		"email":    "merchant@example.com",
		"password": "secret123",
	})
	if err != nil {
		b.Fatalf("marshal failed: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

// BenchmarkHTTPCreateInvoice measures the authenticated POST /invoices
// endpoint, exercising auth middleware, role checks, and invoice creation
// (including unique invoice-number/payment-token generation).
func BenchmarkHTTPCreateInvoice(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body, err := json.Marshal(map[string]any{
			"merchantId": fx.merchantID,
			"amount":     5000 + i,
			"currency":   "USD",
		})
		if err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+fx.merchantToken)
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			b.Fatalf("unexpected status: %d, body: %s", rec.Code, rec.Body.String())
		}
	}
}

// BenchmarkHTTPResolvePaymentLink measures the unauthenticated public
// GET /pay/:token endpoint used by end customers to view an invoice.
func BenchmarkHTTPResolvePaymentLink(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/pay/%s", fx.paymentToken), nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

// BenchmarkHTTPRegister measures the /auth/register endpoint end-to-end:
// JSON binding, duplicate-email lookup, bcrypt hashing, and JWT issuance.
// Like login, this is bcrypt-dominated by design.
func BenchmarkHTTPRegister(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body, err := json.Marshal(map[string]any{
			"name":     "Bench Merchant",
			"email":    fmt.Sprintf("bench-register-%d@example.com", i),
			"password": "secret123",
			"role":     "MERCHANT",
		})
		if err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			b.Fatalf("unexpected status: %d, body: %s", rec.Code, rec.Body.String())
		}
	}
}

// BenchmarkHTTPRequestRefund measures the authenticated POST
// /merchant/refunds endpoint against a paid invoice with ample refundable
// balance, so every iteration succeeds independently.
func BenchmarkHTTPRequestRefund(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body, err := json.Marshal(map[string]any{
			"invoiceId":  fx.paidInvoiceID,
			"merchantId": fx.merchantID,
			"amount":     1,
		})
		if err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/merchant/refunds", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+fx.merchantToken)
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			b.Fatalf("unexpected status: %d, body: %s", rec.Code, rec.Body.String())
		}
	}
}

// BenchmarkHTTPRequestTopUp measures the authenticated POST
// /merchant/topups endpoint.
func BenchmarkHTTPRequestTopUp(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		body, err := json.Marshal(map[string]any{
			"merchantId": fx.merchantID,
			"amount":     100,
			"requestKey": fmt.Sprintf("bench-topup-key-%d", i),
		})
		if err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/merchant/topups", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+fx.merchantToken)
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			b.Fatalf("unexpected status: %d, body: %s", rec.Code, rec.Body.String())
		}
	}
}

// BenchmarkHTTPDashboardStats measures the authenticated GET
// /admin/dashboard/stats endpoint (aggregation across invoices, payment
// intents, and refunds).
func BenchmarkHTTPDashboardStats(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/stats?merchantId="+fx.merchantID, nil)
	req.Header.Set("Authorization", "Bearer "+fx.adminToken)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d, body: %s", rec.Code, rec.Body.String())
		}
	}
}

// BenchmarkHTTPSimulatePaymentIntent measures the authenticated admin
// POST /admin/payment-intents/:intentId/simulate endpoint using the FAILED
// outcome, which only mutates the payment intent (safe to repeat across
// iterations without needing a fresh invoice/intent each time).
func BenchmarkHTTPSimulatePaymentIntent(b *testing.B) {
	silenceGinAndLog(b)
	fx := newBenchEndpointFixture(b)
	body, err := json.Marshal(map[string]any{"outcome": "FAILED"})
	if err != nil {
		b.Fatalf("marshal failed: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/admin/payment-intents/"+fx.intentID+"/simulate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+fx.adminToken)
		rec := httptest.NewRecorder()
		fx.router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d, body: %s", rec.Code, rec.Body.String())
		}
	}
}
