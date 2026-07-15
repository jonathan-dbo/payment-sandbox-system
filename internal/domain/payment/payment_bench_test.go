package payment

import (
	"context"
	"testing"
)

// BenchmarkPaymentAuthorizeCaptureRefund measures the full happy-path state
// transition chain for a Payment aggregate (Authorize -> Capture -> Refund).
func BenchmarkPaymentAuthorizeCaptureRefund(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p, err := NewPayment("m-1", 1000, "USD")
		if err != nil {
			b.Fatalf("NewPayment failed: %v", err)
		}
		if err := p.Authorize(ctx); err != nil {
			b.Fatalf("Authorize failed: %v", err)
		}
		if err := p.Capture(ctx); err != nil {
			b.Fatalf("Capture failed: %v", err)
		}
		if err := p.Refund(ctx, 200); err != nil {
			b.Fatalf("Refund failed: %v", err)
		}
	}
}

// BenchmarkPaymentStatusCanTransitionTo measures the pure value-object
// transition-check hot path used throughout the state machine.
func BenchmarkPaymentStatusCanTransitionTo(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = StatusPending.CanTransitionTo(StatusAuthorized)
		_ = StatusAuthorized.CanTransitionTo(StatusCaptured)
		_ = StatusCaptured.CanTransitionTo(StatusRefunded)
	}
}
