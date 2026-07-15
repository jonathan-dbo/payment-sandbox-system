package paymentintent_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
)

// BenchmarkPaymentIntentMarkSuccess measures the PaymentIntent MarkSuccess
// state transition.
func BenchmarkPaymentIntentMarkSuccess(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p := &paymentintent.PaymentIntent{Status: paymentintent.StatusPending}
		if err := p.MarkSuccess(); err != nil {
			b.Fatalf("MarkSuccess failed: %v", err)
		}
	}
}

// BenchmarkPaymentIntentMarkFailed measures the PaymentIntent MarkFailed
// state transition.
func BenchmarkPaymentIntentMarkFailed(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p := &paymentintent.PaymentIntent{Status: paymentintent.StatusPending}
		if err := p.MarkFailed(); err != nil {
			b.Fatalf("MarkFailed failed: %v", err)
		}
	}
}
