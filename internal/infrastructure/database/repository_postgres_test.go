package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/stretchr/testify/require"
)

func TestRepositoryPostgresAdapterBehavior(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresInvoiceRepository(db)
	now := time.Now().UTC()
	model := &invoice.Invoice{
		ID:            "inv-1",
		InvoiceNumber: "INV-001",
		MerchantID:    "m-1",
		Amount:        1000,
		Currency:      "USD",
		Status:        invoice.StatusPending,
		PaymentToken:  "tok-1",
		CreatedAt:     now,
	}

	mock.ExpectExec("INSERT INTO invoices").
		WithArgs(model.ID, model.InvoiceNumber, model.MerchantID, model.Amount, model.Currency, model.Status, model.PaymentToken, nil, model.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.Create(context.Background(), model))

	rows := sqlmock.NewRows([]string{"id", "invoice_number", "merchant_id", "amount", "currency", "status", "payment_token", "due_date", "created_at"}).
		AddRow(model.ID, model.InvoiceNumber, model.MerchantID, model.Amount, model.Currency, model.Status, model.PaymentToken, nil, model.CreatedAt)
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE invoice_number = \\$1").
		WithArgs(model.InvoiceNumber).
		WillReturnRows(rows)
	_, err = repo.FindByInvoiceNumber(context.Background(), model.InvoiceNumber)
	require.NoError(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRepositoryTransactionContextPropagation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPostgresInvoiceRepository(db)
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), &sql.TxOptions{})
	require.NoError(t, err)

	now := time.Now().UTC()
	model := &invoice.Invoice{
		ID:            "inv-2",
		InvoiceNumber: "INV-002",
		MerchantID:    "m-2",
		Amount:        2000,
		Currency:      "USD",
		Status:        invoice.StatusPending,
		PaymentToken:  "tok-2",
		CreatedAt:     now,
	}
	mock.ExpectExec("INSERT INTO invoices").
		WithArgs(model.ID, model.InvoiceNumber, model.MerchantID, model.Amount, model.Currency, model.Status, model.PaymentToken, nil, model.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ctx := WithTx(context.Background(), tx)
	require.NoError(t, repo.Create(ctx, model))
	mock.ExpectCommit()
	require.NoError(t, tx.Commit())
	require.NoError(t, mock.ExpectationsWereMet())
}
