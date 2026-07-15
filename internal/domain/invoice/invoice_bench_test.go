package invoice_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
)

// BenchmarkInvoiceMarkPaid measures the invoice MarkPaid state transition.
func BenchmarkInvoiceMarkPaid(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		inv := &invoice.Invoice{Status: invoice.StatusPending}
		if err := inv.MarkPaid(); err != nil {
			b.Fatalf("MarkPaid failed: %v", err)
		}
	}
}

// BenchmarkInvoiceMarkExpired measures the invoice MarkExpired state transition.
func BenchmarkInvoiceMarkExpired(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		inv := &invoice.Invoice{Status: invoice.StatusPending}
		if err := inv.MarkExpired(); err != nil {
			b.Fatalf("MarkExpired failed: %v", err)
		}
	}
}
