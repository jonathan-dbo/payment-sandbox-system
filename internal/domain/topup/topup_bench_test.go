package topup_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
)

// BenchmarkTopUpMarkSuccess measures the TopUp MarkSuccess state transition.
func BenchmarkTopUpMarkSuccess(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tu := &topup.TopUp{Status: topup.StatusPending}
		if err := tu.MarkSuccess(); err != nil {
			b.Fatalf("MarkSuccess failed: %v", err)
		}
	}
}

// BenchmarkTopUpMarkFailed measures the TopUp MarkFailed state transition.
func BenchmarkTopUpMarkFailed(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tu := &topup.TopUp{Status: topup.StatusPending}
		if err := tu.MarkFailed(); err != nil {
			b.Fatalf("MarkFailed failed: %v", err)
		}
	}
}
