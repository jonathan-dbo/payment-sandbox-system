package topup_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"
)

// silenceLog discards shared.LogEvent output for the duration of a benchmark
// so structured logging I/O doesn't distort throughput/allocation results.
func silenceLog(b *testing.B) {
	b.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(prev) })
}

// BenchmarkTopUpRequest measures the top-up request hot path, including the
// idempotency lookup against RequestKey.
func BenchmarkTopUpRequest(b *testing.B) {
	silenceLog(b)
	svc, _ := newTopUpService()
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-req-%d", i)
		if _, err := svc.Request(ctx, "m-1", 500, key); err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkTopUpAdminUpdateSuccess measures the atomic admin-update success
// path that mutates both the top-up record and the merchant wallet balance.
func BenchmarkTopUpAdminUpdateSuccess(b *testing.B) {
	silenceLog(b)
	svc, _ := newTopUpService()
	ctx := context.Background()

	ids := make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench-admin-req-%d", i)
		model, err := svc.Request(ctx, "m-1", 500, key)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
		ids = append(ids, model.ID)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.AdminUpdate(ctx, ids[i], true); err != nil {
			b.Fatalf("AdminUpdate failed: %v", err)
		}
	}
}
