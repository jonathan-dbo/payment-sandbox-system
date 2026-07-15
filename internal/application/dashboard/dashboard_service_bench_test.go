package dashboard_test

import (
	"context"
	"testing"
	"time"

	appDashboard "github.com/gonszalito/go-ddd-architecture/internal/application/dashboard"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
)

// newBenchDashboardService seeds a moderately sized dataset (200 invoices,
// 200 payment intents, 200 refunds) so GetStats' aggregation loops have
// representative work to measure.
func newBenchDashboardService() *appDashboard.Service {
	now := time.Now().UTC()
	invoices := make([]*invoice.Invoice, 0, 200)
	intents := make([]*paymentintent.PaymentIntent, 0, 200)
	refunds := make([]*refund.Refund, 0, 200)

	for i := 0; i < 200; i++ {
		status := invoice.StatusPending
		switch i % 3 {
		case 0:
			status = invoice.StatusPaid
		case 1:
			status = invoice.StatusExpired
		}
		invoices = append(invoices, &invoice.Invoice{
			ID:         intToID("inv", i),
			MerchantID: "m-1",
			Amount:     int64(1000 + i),
			Status:     status,
			CreatedAt:  now,
		})

		intentStatus := paymentintent.StatusPending
		if i%4 == 0 {
			intentStatus = paymentintent.StatusFailed
		}
		intents = append(intents, &paymentintent.PaymentIntent{
			ID:        intToID("pi", i),
			InvoiceID: intToID("inv", i),
			Status:    intentStatus,
			CreatedAt: now,
		})

		refundStatus := refund.StatusRequested
		if i%5 == 0 {
			refundStatus = refund.StatusSuccess
		}
		refunds = append(refunds, &refund.Refund{
			ID:         intToID("rf", i),
			MerchantID: "m-1",
			Amount:     int64(100 + i),
			Status:     refundStatus,
			CreatedAt:  now,
		})
	}

	return appDashboard.NewService(
		database.NewInMemoryInvoiceRepository(invoices),
		database.NewInMemoryPaymentRepository(intents),
		database.NewInMemoryRefundRepository(refunds),
	)
}

func intToID(prefix string, i int) string {
	return prefix + "-" + string(rune('a'+i%26)) + string(rune('0'+i/26%10))
}

// BenchmarkDashboardGetStats measures the full stats aggregation path across
// invoices, payment intents, and refunds for a single merchant.
func BenchmarkDashboardGetStats(b *testing.B) {
	svc := newBenchDashboardService()
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GetStats(ctx, appDashboard.Filter{MerchantID: "m-1"}); err != nil {
			b.Fatalf("GetStats failed: %v", err)
		}
	}
}

// BenchmarkDashboardGetStats_WithDateRange measures the aggregation path
// when a start/end date filter is applied, exercising the repositories'
// date-range filtering logic in addition to the aggregation loops.
func BenchmarkDashboardGetStats_WithDateRange(b *testing.B) {
	svc := newBenchDashboardService()
	ctx := context.Background()
	start := time.Now().UTC().Add(-24 * time.Hour)
	end := time.Now().UTC().Add(24 * time.Hour)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GetStats(ctx, appDashboard.Filter{MerchantID: "m-1", StartDate: &start, EndDate: &end}); err != nil {
			b.Fatalf("GetStats failed: %v", err)
		}
	}
}
