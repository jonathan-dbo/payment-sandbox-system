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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardAggregateCalculations(t *testing.T) {
	svc := newDashboardService(t)
	stats, err := svc.GetStats(context.Background(), appDashboard.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 4, stats.TotalInvoices)
	assert.Equal(t, 2, stats.PaidCount)
	assert.Equal(t, 1, stats.FailedCount)
	assert.Equal(t, 1, stats.ExpiredCount)
	assert.Equal(t, int64(3000), stats.TotalTransactionAmount)
	assert.Equal(t, int64(500), stats.TotalRefundAmount)
}

func TestDashboardFilterCombination(t *testing.T) {
	svc := newDashboardService(t)
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 3, 23, 59, 59, 0, time.UTC)
	stats, err := svc.GetStats(context.Background(), appDashboard.Filter{
		MerchantID: "m-1",
		StartDate:  &start,
		EndDate:    &end,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, stats.TotalInvoices)
	assert.Equal(t, 1, stats.PaidCount)
	assert.Equal(t, 1, stats.FailedCount)
	assert.Equal(t, 1, stats.ExpiredCount)
	assert.Equal(t, int64(2000), stats.TotalTransactionAmount)
	assert.Equal(t, int64(500), stats.TotalRefundAmount)
}

func TestDashboardEmptyDataSetBehavior(t *testing.T) {
	svc := appDashboard.NewService(
		database.NewInMemoryInvoiceRepository(nil),
		database.NewInMemoryPaymentRepository(nil),
		database.NewInMemoryRefundRepository(nil),
	)
	stats, err := svc.GetStats(context.Background(), appDashboard.Filter{})
	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalInvoices)
	assert.Equal(t, 0, stats.PaidCount)
	assert.Equal(t, 0, stats.FailedCount)
	assert.Equal(t, 0, stats.ExpiredCount)
	assert.Equal(t, int64(0), stats.TotalTransactionAmount)
	assert.Equal(t, int64(0), stats.TotalRefundAmount)
}

func newDashboardService(t *testing.T) *appDashboard.Service {
	t.Helper()
	invoiceRepo := database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "i1", MerchantID: "m-1", Status: invoice.StatusPaid, Amount: 1000, CreatedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)},
		{ID: "i2", MerchantID: "m-1", Status: invoice.StatusPaid, Amount: 2000, CreatedAt: time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)},
		{ID: "i3", MerchantID: "m-1", Status: invoice.StatusExpired, Amount: 3000, CreatedAt: time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)},
		{ID: "i4", MerchantID: "m-2", Status: invoice.StatusPending, Amount: 500, CreatedAt: time.Date(2026, 1, 4, 10, 0, 0, 0, time.UTC)},
	})
	refundRepo := database.NewInMemoryRefundRepository([]*refund.Refund{
		{ID: "r1", MerchantID: "m-1", Amount: 500, Status: refund.StatusSuccess, CreatedAt: time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)},
		{ID: "r2", MerchantID: "m-2", Amount: 300, Status: refund.StatusFailed, CreatedAt: time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC)},
	})
	paymentRepo := database.NewInMemoryPaymentRepository([]*paymentintent.PaymentIntent{
		{ID: "p1", InvoiceID: "i1", Status: paymentintent.StatusSuccess, CreatedAt: time.Date(2026, 1, 1, 10, 5, 0, 0, time.UTC)},
		{ID: "p2", InvoiceID: "i2", Status: paymentintent.StatusFailed, CreatedAt: time.Date(2026, 1, 2, 10, 5, 0, 0, time.UTC)},
	})
	return appDashboard.NewService(invoiceRepo, paymentRepo, refundRepo)
}
