package invoice_test

import (
	"context"
	"io"
	"log"
	"testing"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
)

// silenceLog discards shared.LogEvent output for the duration of a benchmark
// so structured logging I/O doesn't distort throughput/allocation numbers.
func silenceLog(b *testing.B) {
	b.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(prev) })
}

// BenchmarkInvoiceCreate measures invoice creation, including unique
// invoice-number and payment-token generation (crypto/rand + repo lookups).
func BenchmarkInvoiceCreate(b *testing.B) {
	silenceLog(b)
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Create(ctx, appInvoice.CreateInvoiceInput{MerchantID: "m-1", Amount: 1000, Currency: "USD"}); err != nil {
			b.Fatalf("Create failed: %v", err)
		}
	}
}

// BenchmarkInvoiceGetByID measures the read path including the
// refresh-expiry check performed on every fetch.
func BenchmarkInvoiceGetByID(b *testing.B) {
	silenceLog(b)
	repo := database.NewInMemoryInvoiceRepository(nil)
	svc := appInvoice.NewService(repo)
	ctx := context.Background()
	inv, err := svc.Create(ctx, appInvoice.CreateInvoiceInput{MerchantID: "m-1", Amount: 1000, Currency: "USD"})
	if err != nil {
		b.Fatalf("setup Create failed: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GetByID(ctx, inv.ID); err != nil {
			b.Fatalf("GetByID failed: %v", err)
		}
	}
}

// BenchmarkInvoiceList measures paginated listing over a moderately sized
// dataset, exercising the in-memory repo's filter+sort+paginate path.
func BenchmarkInvoiceList(b *testing.B) {
	silenceLog(b)
	repo := database.NewInMemoryInvoiceRepository(nil)
	svc := appInvoice.NewService(repo)
	ctx := context.Background()
	for i := 0; i < 200; i++ {
		if _, err := svc.Create(ctx, appInvoice.CreateInvoiceInput{MerchantID: "m-1", Amount: int64(1000 + i), Currency: "USD"}); err != nil {
			b.Fatalf("setup Create failed: %v", err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.List(ctx, appInvoice.ListFilter{MerchantID: "m-1", Page: 1, PageSize: 20}); err != nil {
			b.Fatalf("List failed: %v", err)
		}
	}
}

// BenchmarkInvoiceResolvePaymentToken measures the public payment-token
// resolution path used by the unauthenticated /pay/:token endpoint.
func BenchmarkInvoiceResolvePaymentToken(b *testing.B) {
	silenceLog(b)
	repo := database.NewInMemoryInvoiceRepository(nil)
	svc := appInvoice.NewService(repo)
	ctx := context.Background()
	inv, err := svc.Create(ctx, appInvoice.CreateInvoiceInput{MerchantID: "m-1", Amount: 1000, Currency: "USD"})
	if err != nil {
		b.Fatalf("setup Create failed: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ResolvePaymentToken(ctx, inv.PaymentToken); err != nil {
			b.Fatalf("ResolvePaymentToken failed: %v", err)
		}
	}
}

// BenchmarkInvoiceUpdate measures the update path (fetch + validate + save).
func BenchmarkInvoiceUpdate(b *testing.B) {
	silenceLog(b)
	repo := database.NewInMemoryInvoiceRepository(nil)
	svc := appInvoice.NewService(repo)
	ctx := context.Background()

	ids := make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		inv, err := svc.Create(ctx, appInvoice.CreateInvoiceInput{MerchantID: "m-1", Amount: 1000, Currency: "USD"})
		if err != nil {
			b.Fatalf("setup Create failed: %v", err)
		}
		ids = append(ids, inv.ID)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		amount := int64(2000 + i)
		if _, err := svc.Update(ctx, ids[i], appInvoice.UpdateInvoiceInput{Amount: amount, Currency: "USD"}); err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}
