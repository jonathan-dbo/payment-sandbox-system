package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	appWallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	domainInvoice "github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	domainPaymentIntent "github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	domainRefund "github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	domainTopUp "github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
	domainUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -------------------- Invoice --------------------

func TestPostgresInvoiceRepository_FindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)
	now := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "invoice_number", "merchant_id", "amount", "currency", "status", "payment_token", "due_date", "created_at"}).
		AddRow("inv-1", "INV-001", "m-1", int64(1000), "USD", domainInvoice.StatusPending, "tok-1", nil, now)
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE id = \\$1").
		WithArgs("inv-1").
		WillReturnRows(rows)

	inv, err := repo.FindByID(context.Background(), "inv-1")
	require.NoError(t, err)
	assert.Equal(t, "inv-1", inv.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_FindByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)

	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE id = \\$1").
		WithArgs("missing").
		WillReturnError(errNoRows())

	_, err = repo.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestPostgresInvoiceRepository_FindByPaymentToken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)
	now := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "invoice_number", "merchant_id", "amount", "currency", "status", "payment_token", "due_date", "created_at"}).
		AddRow("inv-1", "INV-001", "m-1", int64(1000), "USD", domainInvoice.StatusPending, "tok-1", now.Add(time.Hour), now)
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE payment_token = \\$1").
		WithArgs("tok-1").
		WillReturnRows(rows)

	inv, err := repo.FindByPaymentToken(context.Background(), "tok-1")
	require.NoError(t, err)
	assert.Equal(t, "tok-1", inv.PaymentToken)
	assert.False(t, inv.DueDate.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_FindByPaymentToken_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)

	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE payment_token = \\$1").
		WithArgs("missing-token").
		WillReturnError(errNoRows())

	_, err = repo.FindByPaymentToken(context.Background(), "missing-token")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestPostgresInvoiceRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)
	now := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "invoice_number", "merchant_id", "amount", "currency", "status", "payment_token", "due_date", "created_at"}).
		AddRow("inv-1", "INV-001", "m-1", int64(1000), "USD", domainInvoice.StatusPending, "tok-1", nil, now)
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE merchant_id = \\$1 AND status = \\$2 ORDER BY created_at DESC LIMIT \\$3 OFFSET \\$4").
		WithArgs("m-1", "PENDING", 10, 0).
		WillReturnRows(rows)

	items, err := repo.List(context.Background(), appInvoice.ListFilter{MerchantID: "m-1", Status: "pending", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "inv-1", items[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_List_DefaultsPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)

	rows := sqlmock.NewRows([]string{"id", "invoice_number", "merchant_id", "amount", "currency", "status", "payment_token", "due_date", "created_at"})
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
		WithArgs(10, 0).
		WillReturnRows(rows)

	items, err := repo.List(context.Background(), appInvoice.ListFilter{Page: 0, PageSize: 0})
	require.NoError(t, err)
	assert.Len(t, items, 0)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)
	inv := &domainInvoice.Invoice{ID: "inv-1", Amount: 2000, Currency: "USD", Status: domainInvoice.StatusPaid}

	mock.ExpectExec("UPDATE invoices SET amount = \\$1, currency = \\$2, status = \\$3, due_date = \\$4 WHERE id = \\$5").
		WithArgs(inv.Amount, inv.Currency, inv.Status, nil, inv.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Save(context.Background(), inv))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_Delete(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)

	mock.ExpectExec("DELETE FROM invoices WHERE id = \\$1").
		WithArgs("inv-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Delete(context.Background(), "inv-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_Delete_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)

	mock.ExpectExec("DELETE FROM invoices WHERE id = \\$1").
		WithArgs("inv-1").
		WillReturnError(errors.New("boom"))

	err = repo.Delete(context.Background(), "inv-1")
	require.Error(t, err)
}

// -------------------- Payment Intent --------------------

func TestPostgresPaymentRepository_CreateFindSaveList(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresPaymentRepository(db)
	now := time.Now().UTC()
	intent := &domainPaymentIntent.PaymentIntent{
		ID: "pi-1", InvoiceID: "inv-1", Method: "WALLET", Status: domainPaymentIntent.StatusPending,
		DueAt: now.Add(time.Hour), CreatedAt: now,
	}

	mock.ExpectExec("INSERT INTO payment_intents").
		WithArgs(intent.ID, intent.InvoiceID, intent.Method, intent.Status, intent.DueAt, intent.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.Create(context.Background(), intent))

	rows := sqlmock.NewRows([]string{"id", "invoice_id", "method", "status", "due_at", "created_at"}).
		AddRow(intent.ID, intent.InvoiceID, intent.Method, intent.Status, intent.DueAt, intent.CreatedAt)
	mock.ExpectQuery("SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents WHERE id = \\$1").
		WithArgs("pi-1").
		WillReturnRows(rows)
	found, err := repo.FindByID(context.Background(), "pi-1")
	require.NoError(t, err)
	assert.Equal(t, "pi-1", found.ID)

	mock.ExpectExec("UPDATE payment_intents SET status = \\$1 WHERE id = \\$2").
		WithArgs(domainPaymentIntent.StatusSuccess, "pi-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	intent.Status = domainPaymentIntent.StatusSuccess
	require.NoError(t, repo.Save(context.Background(), intent))

	listRows := sqlmock.NewRows([]string{"id", "invoice_id", "method", "status", "due_at", "created_at"}).
		AddRow(intent.ID, intent.InvoiceID, intent.Method, intent.Status, intent.DueAt, intent.CreatedAt)
	mock.ExpectQuery("SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents WHERE invoice_id = \\$1 ORDER BY created_at ASC").
		WithArgs("inv-1").
		WillReturnRows(listRows)
	items, err := repo.List(context.Background(), appPayment.ListFilter{InvoiceID: "inv-1"})
	require.NoError(t, err)
	require.Len(t, items, 1)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresPaymentRepository_FindByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresPaymentRepository(db)

	mock.ExpectQuery("SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents WHERE id = \\$1").
		WithArgs("missing").
		WillReturnError(errNoRows())

	_, err = repo.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appPayment.ErrNotFound))
}

func TestPostgresPaymentRepository_List_DateRangeFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresPaymentRepository(db)
	start := time.Now().UTC().Add(-time.Hour)
	end := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "invoice_id", "method", "status", "due_at", "created_at"})
	mock.ExpectQuery("SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents WHERE created_at >= \\$1 AND created_at <= \\$2 ORDER BY created_at ASC").
		WithArgs(start, end).
		WillReturnRows(rows)

	items, err := repo.List(context.Background(), appPayment.ListFilter{StartDate: &start, EndDate: &end})
	require.NoError(t, err)
	assert.Len(t, items, 0)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresPaymentRepository_List_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresPaymentRepository(db)

	mock.ExpectQuery("SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents ORDER BY created_at ASC").
		WillReturnError(errors.New("boom"))

	_, err = repo.List(context.Background(), appPayment.ListFilter{})
	require.Error(t, err)
}

// -------------------- Refund --------------------

func TestPostgresRefundRepository_CreateFindSave(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)
	now := time.Now().UTC()
	rf := &domainRefund.Refund{ID: "rf-1", InvoiceID: "inv-1", MerchantID: "m-1", Amount: 500, Status: domainRefund.StatusRequested, CreatedAt: now}

	mock.ExpectExec("INSERT INTO refunds").
		WithArgs(rf.ID, rf.InvoiceID, rf.MerchantID, rf.Amount, rf.Status, rf.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.Create(context.Background(), rf))

	rows := sqlmock.NewRows([]string{"id", "invoice_id", "merchant_id", "amount", "status", "created_at"}).
		AddRow(rf.ID, rf.InvoiceID, rf.MerchantID, rf.Amount, rf.Status, rf.CreatedAt)
	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE id = \\$1").
		WithArgs("rf-1").
		WillReturnRows(rows)
	found, err := repo.FindByID(context.Background(), "rf-1")
	require.NoError(t, err)
	assert.Equal(t, "rf-1", found.ID)

	mock.ExpectExec("UPDATE refunds SET status = \\$1 WHERE id = \\$2").
		WithArgs(domainRefund.StatusApproved, "rf-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	rf.Status = domainRefund.StatusApproved
	require.NoError(t, repo.Save(context.Background(), rf))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRefundRepository_FindByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)

	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE id = \\$1").
		WithArgs("missing").
		WillReturnError(errNoRows())

	_, err = repo.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appRefund.ErrNotFound))
}

func TestPostgresRefundRepository_List_WithDateFilters(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)
	now := time.Now().UTC()
	inRange := now
	outOfRange := now.Add(-48 * time.Hour)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	rows := sqlmock.NewRows([]string{"id", "invoice_id", "merchant_id", "amount", "status", "created_at"}).
		AddRow("rf-1", "inv-1", "m-1", int64(100), domainRefund.StatusRequested, inRange).
		AddRow("rf-2", "inv-1", "m-1", int64(200), domainRefund.StatusRequested, outOfRange)
	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE merchant_id = \\$1 ORDER BY id ASC").
		WithArgs("m-1").
		WillReturnRows(rows)

	items, err := repo.List(context.Background(), appRefund.ListFilter{MerchantID: "m-1", StartDate: &start, EndDate: &end})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "rf-1", items[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRefundRepository_List_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)

	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE merchant_id = \\$1 ORDER BY id ASC").
		WithArgs("m-1").
		WillReturnError(errors.New("boom"))

	_, err = repo.List(context.Background(), appRefund.ListFilter{MerchantID: "m-1"})
	require.Error(t, err)
}

func TestPostgresRefundRepository_ListByInvoiceID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)
	now := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "invoice_id", "merchant_id", "amount", "status", "created_at"}).
		AddRow("rf-1", "inv-1", "m-1", int64(100), domainRefund.StatusRequested, now)
	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE invoice_id = \\$1 ORDER BY id ASC").
		WithArgs("inv-1").
		WillReturnRows(rows)

	items, err := repo.ListByInvoiceID(context.Background(), "inv-1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRefundRepository_ListByInvoiceID_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)

	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE invoice_id = \\$1 ORDER BY id ASC").
		WithArgs("inv-1").
		WillReturnError(errors.New("boom"))

	_, err = repo.ListByInvoiceID(context.Background(), "inv-1")
	require.Error(t, err)
}

// -------------------- TopUp --------------------

func TestPostgresTopUpRepository_CreateFindSave(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)
	tu := &domainTopUp.TopUp{ID: "tu-1", MerchantID: "m-1", Amount: 500, Status: domainTopUp.StatusPending, RequestKey: "req-1"}

	mock.ExpectExec("INSERT INTO top_ups").
		WithArgs(tu.ID, tu.MerchantID, tu.Amount, tu.Status, tu.RequestKey).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, repo.Create(context.Background(), tu))

	rows := sqlmock.NewRows([]string{"id", "merchant_id", "amount", "status", "request_key"}).
		AddRow(tu.ID, tu.MerchantID, tu.Amount, tu.Status, tu.RequestKey)
	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE id = \\$1").
		WithArgs("tu-1").
		WillReturnRows(rows)
	found, err := repo.FindByID(context.Background(), "tu-1")
	require.NoError(t, err)
	assert.Equal(t, "tu-1", found.ID)

	mock.ExpectExec("UPDATE top_ups SET status = \\$1 WHERE id = \\$2").
		WithArgs(domainTopUp.StatusSuccess, "tu-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	tu.Status = domainTopUp.StatusSuccess
	require.NoError(t, repo.Save(context.Background(), tu))

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTopUpRepository_FindByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)

	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE id = \\$1").
		WithArgs("missing").
		WillReturnError(errNoRows())

	_, err = repo.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appTopUp.ErrNotFound))
}

func TestPostgresTopUpRepository_FindByRequestKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)

	rows := sqlmock.NewRows([]string{"id", "merchant_id", "amount", "status", "request_key"}).
		AddRow("tu-1", "m-1", int64(500), domainTopUp.StatusPending, "req-1")
	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE merchant_id = \\$1 AND request_key = \\$2").
		WithArgs("m-1", "req-1").
		WillReturnRows(rows)

	found, err := repo.FindByRequestKey(context.Background(), "m-1", "req-1")
	require.NoError(t, err)
	assert.Equal(t, "tu-1", found.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTopUpRepository_FindByRequestKey_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)

	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE merchant_id = \\$1 AND request_key = \\$2").
		WithArgs("m-1", "missing-key").
		WillReturnError(errNoRows())

	_, err = repo.FindByRequestKey(context.Background(), "m-1", "missing-key")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appTopUp.ErrNotFound))
}

func TestPostgresTopUpRepository_ListByMerchantID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)

	rows := sqlmock.NewRows([]string{"id", "merchant_id", "amount", "status", "request_key"}).
		AddRow("tu-1", "m-1", int64(500), domainTopUp.StatusPending, "req-1")
	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE merchant_id = \\$1 ORDER BY id ASC").
		WithArgs("m-1").
		WillReturnRows(rows)

	items, err := repo.ListByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresTopUpRepository_ListByMerchantID_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)

	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE merchant_id = \\$1 ORDER BY id ASC").
		WithArgs("m-1").
		WillReturnError(errors.New("boom"))

	_, err = repo.ListByMerchantID(context.Background(), "m-1")
	require.Error(t, err)
}

// -------------------- User --------------------

func TestPostgresUserRepository_FindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "role"}).
		AddRow("u-1", "Jane", "jane@example.com", "hash", domainUser.RoleMerchant)
	mock.ExpectQuery("SELECT id, name, email, password_hash, role FROM users WHERE id = \\$1").
		WithArgs("u-1").
		WillReturnRows(rows)

	u, err := repo.FindByID("u-1")
	require.NoError(t, err)
	assert.Equal(t, "jane@example.com", u.Email)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserRepository_FindByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)

	mock.ExpectQuery("SELECT id, name, email, password_hash, role FROM users WHERE id = \\$1").
		WithArgs("missing").
		WillReturnError(errNoRows())

	_, err = repo.FindByID("missing")
	require.Error(t, err)
}

func TestPostgresUserRepository_FindByEmail(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)

	rows := sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "role"}).
		AddRow("u-1", "Jane", "jane@example.com", "hash", domainUser.RoleMerchant)
	mock.ExpectQuery("SELECT id, name, email, password_hash, role FROM users WHERE email = \\$1").
		WithArgs("jane@example.com").
		WillReturnRows(rows)

	u, err := repo.FindByEmail("jane@example.com")
	require.NoError(t, err)
	assert.Equal(t, "u-1", u.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserRepository_FindByEmail_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)

	mock.ExpectQuery("SELECT id, name, email, password_hash, role FROM users WHERE email = \\$1").
		WithArgs("missing@example.com").
		WillReturnError(errNoRows())

	_, err = repo.FindByEmail("missing@example.com")
	require.Error(t, err)
}

func TestPostgresUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-1", Name: "Jane", Email: "jane@example.com", PasswordHash: "hash", Role: domainUser.RoleUser}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name, u.Email, u.PasswordHash, u.Role).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(u))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserRepository_Create_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-1", Name: "Jane", Email: "jane@example.com", PasswordHash: "hash", Role: domainUser.RoleUser}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name, u.Email, u.PasswordHash, u.Role).
		WillReturnError(errors.New("duplicate key"))

	err = repo.Create(u)
	require.Error(t, err)
}

func TestPostgresUserRepository_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-1", Email: "new@example.com", PasswordHash: "hash2", Role: domainUser.RoleAdmin}

	mock.ExpectExec("UPDATE users SET email = \\$1, password_hash = \\$2, role = \\$3 WHERE id = \\$4").
		WithArgs(u.Email, u.PasswordHash, u.Role, u.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Save(u))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserRepository_CreateWithMerchant_MerchantRole(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-1", Name: "Jane", Email: "jane@example.com", PasswordHash: "hash", Role: domainUser.RoleMerchant}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name, u.Email, u.PasswordHash, u.Role).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO merchants").
		WithArgs(u.ID, u.ID, u.Name, u.Email).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.CreateWithMerchant(u))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserRepository_CreateWithMerchant_NonMerchantRoleSkipsMerchantInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-2", Name: "Bob", Email: "bob@example.com", PasswordHash: "hash", Role: domainUser.RoleUser}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name, u.Email, u.PasswordHash, u.Role).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, repo.CreateWithMerchant(u))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserRepository_CreateWithMerchant_UserInsertFailsRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-3", Name: "Eve", Email: "eve@example.com", PasswordHash: "hash", Role: domainUser.RoleMerchant}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name, u.Email, u.PasswordHash, u.Role).
		WillReturnError(errors.New("duplicate key"))
	mock.ExpectRollback()

	err = repo.CreateWithMerchant(u)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserRepository_CreateWithMerchant_MerchantInsertFailsRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-4", Name: "Sam", Email: "sam@example.com", PasswordHash: "hash", Role: domainUser.RoleMerchant}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name, u.Email, u.PasswordHash, u.Role).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO merchants").
		WithArgs(u.ID, u.ID, u.Name, u.Email).
		WillReturnError(errors.New("duplicate key"))
	mock.ExpectRollback()

	err = repo.CreateWithMerchant(u)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUserRepository_CreateWithMerchant_CommitFailsRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-5", Name: "Amy", Email: "amy@example.com", PasswordHash: "hash", Role: domainUser.RoleUser}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO users").
		WithArgs(u.ID, u.Name, u.Email, u.PasswordHash, u.Role).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
	// Note: database/sql marks a *sql.Tx as done once Commit() returns,
	// even on error, so the subsequent tx.Rollback() call in the repository
	// short-circuits with sql.ErrTxDone and never reaches the driver/mock.
	// No ExpectRollback() is registered here for that reason.

	err = repo.CreateWithMerchant(u)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// -------------------- Wallet --------------------

func TestPostgresWalletRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresWalletRepository(db)
	w := &domainWallet.Wallet{ID: "w-1", MerchantID: "m-1", Balance: 1000}

	mock.ExpectExec("INSERT INTO wallets").
		WithArgs(w.ID, w.MerchantID, w.Balance).
		WillReturnResult(sqlmock.NewResult(1, 1))

	require.NoError(t, repo.Create(context.Background(), w))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresWalletRepository_FindByMerchantID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresWalletRepository(db)

	rows := sqlmock.NewRows([]string{"id", "merchant_id", "balance"}).
		AddRow("w-1", "m-1", int64(1000))
	mock.ExpectQuery("SELECT id, merchant_id, balance FROM wallets WHERE merchant_id = \\$1").
		WithArgs("m-1").
		WillReturnRows(rows)

	w, err := repo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1000), w.Balance)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresWalletRepository_FindByMerchantID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresWalletRepository(db)

	mock.ExpectQuery("SELECT id, merchant_id, balance FROM wallets WHERE merchant_id = \\$1").
		WithArgs("missing").
		WillReturnError(errNoRows())

	_, err = repo.FindByMerchantID(context.Background(), "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appWallet.ErrNotFound))
}

func TestPostgresWalletRepository_Save(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresWalletRepository(db)
	w := &domainWallet.Wallet{ID: "w-1", MerchantID: "m-1", Balance: 2000}

	mock.ExpectExec("UPDATE wallets SET balance = \\$1 WHERE merchant_id = \\$2").
		WithArgs(w.Balance, w.MerchantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Save(context.Background(), w))
	require.NoError(t, mock.ExpectationsWereMet())
}

// errNoRows returns sql.ErrNoRows, used to simulate "not found" query results
// via sqlmock.WillReturnError.
func errNoRows() error {
	return sql.ErrNoRows
}
