package refund_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
)

// silenceLog discards shared.LogEvent output for the duration of a benchmark
// so structured logging I/O doesn't distort throughput/allocation results.
func silenceLog(b *testing.B) {
	b.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(prev) })
}

// newBenchRefundService seeds n paid invoices (one per benchmark iteration)
// so each RequestRefund call operates on an independent, unrefunded invoice.
func newBenchRefundService(n int) (*appRefund.Service, []string) {
	invoices := make([]*invoice.Invoice, 0, n)
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("bench-inv-%d", i)
		ids = append(ids, id)
		invoices = append(invoices, &invoice.Invoice{
			ID:         id,
			MerchantID: "m-1",
			Amount:     500,
			Status:     invoice.StatusPaid,
			CreatedAt:  time.Now().UTC(),
		})
	}
	refundRepo := database.NewInMemoryRefundRepository(nil)
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(invoices))
	walletRepo := database.NewInMemoryWalletRepository([]*wallet.Wallet{
		{ID: "w-1", MerchantID: "m-1", Balance: 1_000_000},
	})
	return appRefund.NewService(refundRepo, walletRepo, invoiceSvc), ids
}

// BenchmarkRefundRequestRefund measures the request-refund hot path
// (invoice lookup + refundable-amount validation + persistence).
func BenchmarkRefundRequestRefund(b *testing.B) {
	silenceLog(b)
	svc, ids := newBenchRefundService(b.N)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.RequestRefund(ctx, ids[i], "m-1", 100); err != nil {
			b.Fatalf("RequestRefund failed: %v", err)
		}
	}
}

// BenchmarkRefundApproveProcessSuccess measures the full approve -> process
// success flow, which atomically mutates both refund and wallet state.
func BenchmarkRefundApproveProcessSuccess(b *testing.B) {
	silenceLog(b)
	svc, ids := newBenchRefundService(b.N)
	ctx := context.Background()

	refundIDs := make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		model, err := svc.RequestRefund(ctx, ids[i], "m-1", 100)
		if err != nil {
			b.Fatalf("RequestRefund failed: %v", err)
		}
		refundIDs = append(refundIDs, model.ID)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Approve(ctx, refundIDs[i]); err != nil {
			b.Fatalf("Approve failed: %v", err)
		}
		if _, err := svc.Process(ctx, refundIDs[i], true); err != nil {
			b.Fatalf("Process failed: %v", err)
		}
	}
}
