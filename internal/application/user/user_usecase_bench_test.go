package user_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"testing"

	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
)

// silenceLog discards shared.LogEvent/log.Println output for the duration of
// a benchmark so structured logging I/O doesn't distort timing/allocations.
func silenceLog(b *testing.B) {
	b.Helper()
	prev := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() { log.SetOutput(prev) })
}

// BenchmarkUserRegister measures registration cost, which is dominated by
// bcrypt password hashing (bcrypt.DefaultCost) by design.
func BenchmarkUserRegister(b *testing.B) {
	silenceLog(b)
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "bench-secret", 60)
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		email := fmt.Sprintf("bench-register-%d@example.com", i)
		if _, err := svc.Register(ctx, "Bench User", email, "secret123", "MERCHANT"); err != nil {
			b.Fatalf("Register failed: %v", err)
		}
	}
}

// BenchmarkUserLogin measures login cost end-to-end: repo lookup + bcrypt
// comparison + JWT issuance. Also bcrypt-dominated by design.
func BenchmarkUserLogin(b *testing.B) {
	silenceLog(b)
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "bench-secret", 60)
	ctx := context.Background()
	_, err := svc.Register(ctx, "Bench User", "bench-login@example.com", "secret123", "MERCHANT")
	if err != nil {
		b.Fatalf("setup Register failed: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Login(ctx, "bench-login@example.com", "secret123"); err != nil {
			b.Fatalf("Login failed: %v", err)
		}
	}
}

// BenchmarkUserRefresh measures the token-refresh hot path, which performs a
// user lookup by ID and re-issues a JWT (no bcrypt work involved).
func BenchmarkUserRefresh(b *testing.B) {
	silenceLog(b)
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "bench-secret", 60)
	ctx := context.Background()
	result, err := svc.Register(ctx, "Bench User", "bench-refresh@example.com", "secret123", "MERCHANT")
	if err != nil {
		b.Fatalf("setup Register failed: %v", err)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := svc.Refresh(ctx, result.UserID); err != nil {
			b.Fatalf("Refresh failed: %v", err)
		}
	}
}
