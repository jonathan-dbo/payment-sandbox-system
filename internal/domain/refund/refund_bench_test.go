package refund_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
)

// BenchmarkRefundFullLifecycle measures the full Requested -> Approved ->
// Success state transition chain for a Refund aggregate.
func BenchmarkRefundFullLifecycle(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := &refund.Refund{Status: refund.StatusRequested}
		if err := r.Approve(); err != nil {
			b.Fatalf("Approve failed: %v", err)
		}
		if err := r.MarkSuccess(); err != nil {
			b.Fatalf("MarkSuccess failed: %v", err)
		}
	}
}

// BenchmarkRefundReject measures the Requested -> Rejected transition.
func BenchmarkRefundReject(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r := &refund.Refund{Status: refund.StatusRequested}
		if err := r.Reject(); err != nil {
			b.Fatalf("Reject failed: %v", err)
		}
	}
}
