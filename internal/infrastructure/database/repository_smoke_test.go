package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	domainInvoice "github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestRepositoryLocalDatabaseSmoke(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skip local DB smoke test")
	}

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, db.Ping())

	repo := NewPostgresInvoiceRepository(db)
	suffix := time.Now().UTC().Format("20060102150405")
	id := "smoke-inv-" + suffix
	model := &domainInvoice.Invoice{
		ID:            id,
		InvoiceNumber: fmt.Sprintf("INV-SMOKE-%s", suffix),
		MerchantID:    "merchant-smoke",
		Amount:        1000,
		Currency:      "USD",
		Status:        domainInvoice.StatusPending,
		PaymentToken:  fmt.Sprintf("smoke-token-%s", suffix),
		CreatedAt:     time.Now().UTC(),
	}

	ctx := context.Background()
	require.NoError(t, repo.Create(ctx, model))
	found, err := repo.FindByInvoiceNumber(ctx, model.InvoiceNumber)
	require.NoError(t, err)
	require.Equal(t, model.ID, found.ID)

	_, err = db.ExecContext(ctx, "DELETE FROM invoices WHERE id = $1", id)
	require.NoError(t, err)
}
