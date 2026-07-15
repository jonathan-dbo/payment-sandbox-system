package payment

import (
	"context"
	"io"
	"log"
	"testing"
)

// silenceLog discards shared.LogEvent output for the duration of a benchmark
// so structured logging I/O doesn't distort throughput/allocation results.
// It restores the previous log output on cleanup.
func silenceLog(b *testing.B) {
	b.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(prev) })
}

// BenchmarkCreateIntent measures throughput/allocations of the payment intent
// creation hot path (public token resolve + intent persistence).
func BenchmarkCreateIntent(b *testing.B) {
	silenceLog(b)
	svc, _, _ := newPaymentServiceWithInvoice(&testing.T{})
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.CreateIntent(ctx, "pay-token-1", MethodWallet); err != nil {
			b.Fatalf("CreateIntent failed: %v", err)
		}
	}
}

// BenchmarkSimulateAdminOutcome_Success measures the atomic success path that
// mutates both the payment intent and the invoice.
func BenchmarkSimulateAdminOutcome_Success(b *testing.B) {
	silenceLog(b)
	svc, _, invoiceRepo := newPaymentServiceWithInvoice(&testing.T{})
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Reset the invoice back to pending so each iteration exercises the
		// full success transition rather than failing on a repeated MarkPaid.
		inv, err := invoiceRepo.FindByID(ctx, "inv-1")
		if err != nil {
			b.Fatalf("FindByID failed: %v", err)
		}
		inv.Status = "PENDING"
		if err := invoiceRepo.Save(ctx, inv); err != nil {
			b.Fatalf("Save failed: %v", err)
		}

		intent, err := svc.CreateIntent(ctx, "pay-token-1", MethodWallet)
		if err != nil {
			b.Fatalf("CreateIntent failed: %v", err)
		}
		if _, err := svc.SimulateAdminOutcome(ctx, intent.ID, OutcomeSuccess); err != nil {
			b.Fatalf("SimulateAdminOutcome failed: %v", err)
		}
	}
}

// BenchmarkSimulateAdminOutcome_Failed measures the failure path, which only
// mutates the payment intent.
func BenchmarkSimulateAdminOutcome_Failed(b *testing.B) {
	silenceLog(b)
	svc, _, _ := newPaymentServiceWithInvoice(&testing.T{})
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		intent, err := svc.CreateIntent(ctx, "pay-token-1", MethodVADummy)
		if err != nil {
			b.Fatalf("CreateIntent failed: %v", err)
		}
		if _, err := svc.SimulateAdminOutcome(ctx, intent.ID, OutcomeFailed); err != nil {
			b.Fatalf("SimulateAdminOutcome failed: %v", err)
		}
	}
}

// BenchmarkResolvePublicToken measures the read-only public token resolution
// path used by the unauthenticated /pay/:token endpoint.
func BenchmarkResolvePublicToken(b *testing.B) {
	silenceLog(b)
	svc, _, _ := newPaymentServiceWithInvoice(&testing.T{})
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.ResolvePublicToken(ctx, "pay-token-1"); err != nil {
			b.Fatalf("ResolvePublicToken failed: %v", err)
		}
	}
}
