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

// ==================== In-memory: TopUp ====================

func TestInMemoryTopUpRepository_Create_PersistsAndIsFindable(t *testing.T) {
	repo := NewInMemoryTopUpRepository(nil)
	tu := &domainTopUp.TopUp{ID: "tu-1", MerchantID: "m-1", Amount: 500, Status: domainTopUp.StatusPending, RequestKey: "req-1"}
	require.NoError(t, repo.Create(context.Background(), tu))

	found, err := repo.FindByID(context.Background(), "tu-1")
	require.NoError(t, err)
	assert.Equal(t, "m-1", found.MerchantID)
	assert.Equal(t, int64(500), found.Amount)
}

func TestInMemoryTopUpRepository_Save_UpdatesExisting(t *testing.T) {
	seed := []*domainTopUp.TopUp{{ID: "tu-1", MerchantID: "m-1", Amount: 100, Status: domainTopUp.StatusPending}}
	repo := NewInMemoryTopUpRepository(seed)

	tu, err := repo.FindByID(context.Background(), "tu-1")
	require.NoError(t, err)
	tu.Status = domainTopUp.StatusSuccess
	tu.Amount = 999
	require.NoError(t, repo.Save(context.Background(), tu))

	updated, err := repo.FindByID(context.Background(), "tu-1")
	require.NoError(t, err)
	assert.Equal(t, domainTopUp.StatusSuccess, updated.Status)
	assert.Equal(t, int64(999), updated.Amount)
}

func TestInMemoryTopUpRepository_FindByID_Found(t *testing.T) {
	seed := []*domainTopUp.TopUp{{ID: "tu-1", MerchantID: "m-1", Amount: 250}}
	repo := NewInMemoryTopUpRepository(seed)

	found, err := repo.FindByID(context.Background(), "tu-1")
	require.NoError(t, err)
	assert.Equal(t, "tu-1", found.ID)
	assert.Equal(t, int64(250), found.Amount)
}

func TestInMemoryTopUpRepository_FindByRequestKey_Found(t *testing.T) {
	seed := []*domainTopUp.TopUp{
		{ID: "tu-1", MerchantID: "m-1", RequestKey: "req-1"},
		{ID: "tu-2", MerchantID: "m-1", RequestKey: "req-2"},
	}
	repo := NewInMemoryTopUpRepository(seed)

	found, err := repo.FindByRequestKey(context.Background(), "m-1", "req-2")
	require.NoError(t, err)
	assert.Equal(t, "tu-2", found.ID)
}

func TestInMemoryTopUpRepository_FindByRequestKey_WrongMerchantNoMatch(t *testing.T) {
	seed := []*domainTopUp.TopUp{{ID: "tu-1", MerchantID: "m-1", RequestKey: "req-1"}}
	repo := NewInMemoryTopUpRepository(seed)

	_, err := repo.FindByRequestKey(context.Background(), "m-2", "req-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, appTopUp.ErrNotFound)
}

func TestInMemoryTopUpRepository_ListByMerchantID_FiltersByMerchant(t *testing.T) {
	seed := []*domainTopUp.TopUp{
		{ID: "tu-1", MerchantID: "m-1"},
		{ID: "tu-2", MerchantID: "m-2"},
		{ID: "tu-3", MerchantID: "m-1"},
	}
	repo := NewInMemoryTopUpRepository(seed)

	items, err := repo.ListByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "tu-1", items[0].ID)
	assert.Equal(t, "tu-3", items[1].ID)
}

// ==================== In-memory: User ====================

func TestInMemoryUserRepository_Create_PersistsAndIsFindable(t *testing.T) {
	repo := NewInMemoryUserRepository(nil)
	u := &domainUser.User{ID: "u-1", Name: "Jane", Email: "jane@example.com", Role: domainUser.RoleUser}
	require.NoError(t, repo.Create(u))

	found, err := repo.FindByID("u-1")
	require.NoError(t, err)
	assert.Equal(t, "jane@example.com", found.Email)
}

func TestInMemoryUserRepository_FindByID_Found(t *testing.T) {
	seed := []*domainUser.User{{ID: "u-1", Name: "Jane", Email: "jane@example.com", Role: domainUser.RoleUser}}
	repo := NewInMemoryUserRepository(seed)

	found, err := repo.FindByID("u-1")
	require.NoError(t, err)
	assert.Equal(t, "Jane", found.Name)
}

func TestInMemoryUserRepository_CreateWithMerchant_MerchantRoleIndexed(t *testing.T) {
	repo := NewInMemoryUserRepository(nil)
	u := &domainUser.User{ID: "u-1", Name: "Merchant Owner", Email: "owner@example.com", Role: domainUser.RoleMerchant}
	require.NoError(t, repo.CreateWithMerchant(u))

	assert.True(t, repo.MerchantExistsForUser("u-1"))
	found, err := repo.FindByID("u-1")
	require.NoError(t, err)
	assert.Equal(t, domainUser.RoleMerchant, found.Role)
}

func TestInMemoryUserRepository_CreateWithMerchant_AdminRoleNotIndexed(t *testing.T) {
	repo := NewInMemoryUserRepository(nil)
	u := &domainUser.User{ID: "u-2", Name: "Admin", Email: "admin@example.com", Role: domainUser.RoleAdmin}
	require.NoError(t, repo.CreateWithMerchant(u))

	assert.False(t, repo.MerchantExistsForUser("u-2"))
	found, err := repo.FindByID("u-2")
	require.NoError(t, err)
	assert.Equal(t, domainUser.RoleAdmin, found.Role)
}

// ==================== In-memory: Invoice ====================

func TestInMemoryInvoiceRepository_FindByPaymentToken_Found(t *testing.T) {
	seed := []*domainInvoice.Invoice{
		{ID: "inv-1", PaymentToken: "tok-1", MerchantID: "m-1", CreatedAt: time.Now().UTC()},
		{ID: "inv-2", PaymentToken: "tok-2", MerchantID: "m-1", CreatedAt: time.Now().UTC()},
	}
	repo := NewInMemoryInvoiceRepository(seed)

	found, err := repo.FindByPaymentToken(context.Background(), "tok-2")
	require.NoError(t, err)
	assert.Equal(t, "inv-2", found.ID)
}

func TestInMemoryInvoiceRepository_Delete_Success(t *testing.T) {
	seed := []*domainInvoice.Invoice{{ID: "inv-1", MerchantID: "m-1", CreatedAt: time.Now().UTC()}}
	repo := NewInMemoryInvoiceRepository(seed)

	require.NoError(t, repo.Delete(context.Background(), "inv-1"))

	_, err := repo.FindByID(context.Background(), "inv-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, appInvoice.ErrNotFound)
}

func TestInMemoryInvoiceRepository_FindByInvoiceNumber_NotFound(t *testing.T) {
	seed := []*domainInvoice.Invoice{{ID: "inv-1", InvoiceNumber: "INV-001", MerchantID: "m-1", CreatedAt: time.Now().UTC()}}
	repo := NewInMemoryInvoiceRepository(seed)

	_, err := repo.FindByInvoiceNumber(context.Background(), "INV-999")
	require.Error(t, err)
	assert.ErrorIs(t, err, appInvoice.ErrNotFound)
}

func TestInMemoryInvoiceRepository_List_DateRangeFiltersAndPagination(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainInvoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", Status: domainInvoice.StatusPending, CreatedAt: now.Add(-3 * time.Hour)},
		{ID: "inv-2", MerchantID: "m-1", Status: domainInvoice.StatusPending, CreatedAt: now.Add(-time.Hour)},
		{ID: "inv-3", MerchantID: "m-1", Status: domainInvoice.StatusPending, CreatedAt: now},
	}
	repo := NewInMemoryInvoiceRepository(seed)
	start := now.Add(-2 * time.Hour)
	end := now.Add(-30 * time.Minute)

	items, err := repo.List(context.Background(), appInvoice.ListFilter{MerchantID: "m-1", StartDate: &start, EndDate: &end, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "inv-2", items[0].ID)
}

func TestInMemoryInvoiceRepository_List_MerchantIDFilterExcludesOthers(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainInvoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", CreatedAt: now},
		{ID: "inv-2", MerchantID: "m-2", CreatedAt: now},
	}
	repo := NewInMemoryInvoiceRepository(seed)

	items, err := repo.List(context.Background(), appInvoice.ListFilter{MerchantID: "m-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "inv-1", items[0].ID)
}

func TestInMemoryInvoiceRepository_List_PageBeyondRangeReturnsEmpty(t *testing.T) {
	seed := []*domainInvoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", CreatedAt: time.Now().UTC()},
	}
	repo := NewInMemoryInvoiceRepository(seed)

	items, err := repo.List(context.Background(), appInvoice.ListFilter{MerchantID: "m-1", Page: 5, PageSize: 10})
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestInMemoryInvoiceRepository_List_EndClampedToLength(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainInvoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", CreatedAt: now.Add(-time.Minute)},
		{ID: "inv-2", MerchantID: "m-1", CreatedAt: now},
	}
	repo := NewInMemoryInvoiceRepository(seed)

	// PageSize (10) exceeds available items (2): exercises the "end > len(items)" clamp branch.
	items, err := repo.List(context.Background(), appInvoice.ListFilter{MerchantID: "m-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, items, 2)
	// newest first
	assert.Equal(t, "inv-2", items[0].ID)
}

func TestInMemoryInvoiceRepository_List_EndWithinLengthNoClamp(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainInvoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", CreatedAt: now.Add(-2 * time.Minute)},
		{ID: "inv-2", MerchantID: "m-1", CreatedAt: now.Add(-time.Minute)},
		{ID: "inv-3", MerchantID: "m-1", CreatedAt: now},
	}
	repo := NewInMemoryInvoiceRepository(seed)

	// PageSize (2) is smaller than available items (3): end stays within bounds, no clamp needed.
	items, err := repo.List(context.Background(), appInvoice.ListFilter{MerchantID: "m-1", Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "inv-3", items[0].ID)
	assert.Equal(t, "inv-2", items[1].ID)
}

// ==================== In-memory: Refund ====================

func TestInMemoryRefundRepository_ListByInvoiceID_FiltersMatches(t *testing.T) {
	seed := []*domainRefund.Refund{
		{ID: "rf-1", InvoiceID: "inv-1"},
		{ID: "rf-2", InvoiceID: "inv-2"},
		{ID: "rf-3", InvoiceID: "inv-1"},
	}
	repo := NewInMemoryRefundRepository(seed)

	items, err := repo.ListByInvoiceID(context.Background(), "inv-1")
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "rf-1", items[0].ID)
	assert.Equal(t, "rf-3", items[1].ID)
}

func TestInMemoryRefundRepository_List_NoFilterReturnsAll(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainRefund.Refund{
		{ID: "rf-1", MerchantID: "m-1", CreatedAt: now},
		{ID: "rf-2", MerchantID: "m-2", CreatedAt: now},
	}
	repo := NewInMemoryRefundRepository(seed)

	items, err := repo.List(context.Background(), appRefund.ListFilter{})
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestInMemoryRefundRepository_List_EndDateFilterExcludesLater(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainRefund.Refund{
		{ID: "rf-1", MerchantID: "m-1", CreatedAt: now.Add(-time.Hour)},
		{ID: "rf-2", MerchantID: "m-1", CreatedAt: now.Add(time.Hour)},
	}
	repo := NewInMemoryRefundRepository(seed)
	end := now

	items, err := repo.List(context.Background(), appRefund.ListFilter{MerchantID: "m-1", EndDate: &end})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "rf-1", items[0].ID)
}

// ==================== In-memory: Wallet ====================

func TestNewInMemoryWalletRepository_SeedsMultipleWallets(t *testing.T) {
	seed := []*domainWallet.Wallet{
		{ID: "w-1", MerchantID: "m-1", Balance: 1000},
		{ID: "w-2", MerchantID: "m-2", Balance: 2000},
	}
	repo := NewInMemoryWalletRepository(seed)

	w1, err := repo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1000), w1.Balance)

	w2, err := repo.FindByMerchantID(context.Background(), "m-2")
	require.NoError(t, err)
	assert.Equal(t, int64(2000), w2.Balance)
}

func TestInMemoryWalletRepository_FindByMerchantID_NotFound(t *testing.T) {
	repo := NewInMemoryWalletRepository(nil)
	_, err := repo.FindByMerchantID(context.Background(), "missing")
	require.Error(t, err)
}

// ==================== In-memory: Payment ====================

func TestInMemoryPaymentRepository_List_EndDateFilterExcludesLater(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainPaymentIntent.PaymentIntent{
		{ID: "pi-1", InvoiceID: "inv-1", CreatedAt: now.Add(-time.Hour)},
		{ID: "pi-2", InvoiceID: "inv-1", CreatedAt: now.Add(time.Hour)},
	}
	repo := NewInMemoryPaymentRepository(seed)
	end := now

	items, err := repo.List(context.Background(), appPayment.ListFilter{EndDate: &end})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "pi-1", items[0].ID)
}

func TestInMemoryPaymentRepository_List_StartDateFilterExcludesEarlier(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainPaymentIntent.PaymentIntent{
		{ID: "pi-1", InvoiceID: "inv-1", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "pi-2", InvoiceID: "inv-1", CreatedAt: now},
	}
	repo := NewInMemoryPaymentRepository(seed)
	start := now.Add(-time.Hour)

	items, err := repo.List(context.Background(), appPayment.ListFilter{StartDate: &start})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "pi-2", items[0].ID)
}

func TestInMemoryPaymentRepository_List_SortsMultipleItemsAscendingByCreatedAt(t *testing.T) {
	now := time.Now().UTC()
	// Seed intentionally out of chronological order so the sort.Slice comparator
	// is exercised for both a "less" (true) and a "not less" (false) comparison.
	seed := []*domainPaymentIntent.PaymentIntent{
		{ID: "pi-later", InvoiceID: "inv-1", CreatedAt: now},
		{ID: "pi-earlier", InvoiceID: "inv-1", CreatedAt: now.Add(-time.Hour)},
		{ID: "pi-middle", InvoiceID: "inv-1", CreatedAt: now.Add(-30 * time.Minute)},
	}
	repo := NewInMemoryPaymentRepository(seed)

	items, err := repo.List(context.Background(), appPayment.ListFilter{InvoiceID: "inv-1"})
	require.NoError(t, err)
	require.Len(t, items, 3)
	assert.Equal(t, "pi-earlier", items[0].ID)
	assert.Equal(t, "pi-middle", items[1].ID)
	assert.Equal(t, "pi-later", items[2].ID)
}

// ==================== Postgres: invoice_repository_pg.go helpers ====================

func TestNullableTime_NonZeroReturnsValue(t *testing.T) {
	ts := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	got := nullableTime(ts)
	assert.Equal(t, ts, got)
}

func TestNullableTime_ZeroReturnsNil(t *testing.T) {
	got := nullableTime(time.Time{})
	assert.Nil(t, got)
}

func TestMapInvoiceErr_PassesThroughOtherErrors(t *testing.T) {
	boom := errors.New("boom")
	got := mapInvoiceErr(boom)
	assert.Equal(t, boom, got)
}

func TestMapInvoiceErr_NilReturnsNil(t *testing.T) {
	assert.Nil(t, mapInvoiceErr(nil))
}

func TestMapInvoiceErr_NotFoundMapped(t *testing.T) {
	got := mapInvoiceErr(errors.New("not_found"))
	assert.ErrorIs(t, got, appInvoice.ErrNotFound)
}

// ==================== Postgres: invoice_repository_pg.go FindByInvoiceNumber / FindByID / List ====================

func TestPostgresInvoiceRepository_FindByInvoiceNumber_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)

	// Row has wrong number/type of columns to force a Scan error distinct from ErrNoRows.
	rows := sqlmock.NewRows([]string{"id"}).AddRow("inv-1")
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE invoice_number = \\$1").
		WithArgs("INV-001").
		WillReturnRows(rows)

	_, err = repo.FindByInvoiceNumber(context.Background(), "INV-001")
	require.Error(t, err)
	assert.False(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestPostgresInvoiceRepository_FindByInvoiceNumber_WithDueDate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)
	now := time.Now().UTC()
	due := now.Add(72 * time.Hour)

	rows := sqlmock.NewRows([]string{"id", "invoice_number", "merchant_id", "amount", "currency", "status", "payment_token", "due_date", "created_at"}).
		AddRow("inv-1", "INV-001", "m-1", int64(1000), "USD", domainInvoice.StatusPending, "tok-1", due, now)
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE invoice_number = \\$1").
		WithArgs("INV-001").
		WillReturnRows(rows)

	inv, err := repo.FindByInvoiceNumber(context.Background(), "INV-001")
	require.NoError(t, err)
	assert.False(t, inv.DueDate.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_FindByID_WithDueDate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)
	now := time.Now().UTC()
	due := now.Add(48 * time.Hour)

	rows := sqlmock.NewRows([]string{"id", "invoice_number", "merchant_id", "amount", "currency", "status", "payment_token", "due_date", "created_at"}).
		AddRow("inv-1", "INV-001", "m-1", int64(1000), "USD", domainInvoice.StatusPending, "tok-1", due, now)
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE id = \\$1").
		WithArgs("inv-1").
		WillReturnRows(rows)

	inv, err := repo.FindByID(context.Background(), "inv-1")
	require.NoError(t, err)
	assert.False(t, inv.DueDate.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_List_WithDueDateRow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)
	now := time.Now().UTC()
	due := now.Add(24 * time.Hour)

	rows := sqlmock.NewRows([]string{"id", "invoice_number", "merchant_id", "amount", "currency", "status", "payment_token", "due_date", "created_at"}).
		AddRow("inv-1", "INV-001", "m-1", int64(1000), "USD", domainInvoice.StatusPending, "tok-1", due, now)
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices WHERE merchant_id = \\$1 ORDER BY created_at DESC LIMIT \\$2 OFFSET \\$3").
		WithArgs("m-1", 10, 0).
		WillReturnRows(rows)

	items, err := repo.List(context.Background(), appInvoice.ListFilter{MerchantID: "m-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.False(t, items[0].DueDate.IsZero())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresInvoiceRepository_List_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)

	rows := sqlmock.NewRows([]string{"id"}).AddRow("inv-1")
	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
		WithArgs(10, 0).
		WillReturnRows(rows)

	_, err = repo.List(context.Background(), appInvoice.ListFilter{Page: 1, PageSize: 10})
	require.Error(t, err)
}

func TestPostgresInvoiceRepository_List_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresInvoiceRepository(db)

	mock.ExpectQuery("SELECT id, invoice_number, merchant_id, amount, currency, status, payment_token, due_date, created_at FROM invoices ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").
		WithArgs(10, 0).
		WillReturnError(errors.New("boom"))

	_, err = repo.List(context.Background(), appInvoice.ListFilter{Page: 1, PageSize: 10})
	require.Error(t, err)
}

// ==================== Postgres: payment_repository_pg.go ====================

func TestPostgresPaymentRepository_FindByID_ScanErrorNonNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresPaymentRepository(db)

	mock.ExpectQuery("SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents WHERE id = \\$1").
		WithArgs("pi-1").
		WillReturnError(errors.New("connection reset"))

	_, err = repo.FindByID(context.Background(), "pi-1")
	require.Error(t, err)
	assert.False(t, errors.Is(err, appPayment.ErrNotFound))
}

func TestPostgresPaymentRepository_List_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresPaymentRepository(db)

	rows := sqlmock.NewRows([]string{"id"}).AddRow("pi-1")
	mock.ExpectQuery("SELECT id, invoice_id, method, status, due_at, created_at FROM payment_intents ORDER BY created_at ASC").
		WillReturnRows(rows)

	_, err = repo.List(context.Background(), appPayment.ListFilter{})
	require.Error(t, err)
}

// ==================== Postgres: refund_repository_pg.go ====================

func TestPostgresRefundRepository_FindByID_ScanErrorNonNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)

	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE id = \\$1").
		WithArgs("rf-1").
		WillReturnError(errors.New("connection reset"))

	_, err = repo.FindByID(context.Background(), "rf-1")
	require.Error(t, err)
	assert.False(t, errors.Is(err, appRefund.ErrNotFound))
}

func TestPostgresRefundRepository_List_EndDateFilterExcludesLater(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)
	now := time.Now().UTC()
	end := now

	rows := sqlmock.NewRows([]string{"id", "invoice_id", "merchant_id", "amount", "status", "created_at"}).
		AddRow("rf-1", "inv-1", "m-1", int64(100), domainRefund.StatusRequested, now.Add(-time.Hour)).
		AddRow("rf-2", "inv-1", "m-1", int64(200), domainRefund.StatusRequested, now.Add(time.Hour))
	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE merchant_id = \\$1 ORDER BY id ASC").
		WithArgs("m-1").
		WillReturnRows(rows)

	items, err := repo.List(context.Background(), appRefund.ListFilter{MerchantID: "m-1", EndDate: &end})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "rf-1", items[0].ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRefundRepository_List_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)

	rows := sqlmock.NewRows([]string{"id"}).AddRow("rf-1")
	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE merchant_id = \\$1 ORDER BY id ASC").
		WithArgs("m-1").
		WillReturnRows(rows)

	_, err = repo.List(context.Background(), appRefund.ListFilter{MerchantID: "m-1"})
	require.Error(t, err)
}

func TestPostgresRefundRepository_ListByInvoiceID_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresRefundRepository(db)

	rows := sqlmock.NewRows([]string{"id"}).AddRow("rf-1")
	mock.ExpectQuery("SELECT id, invoice_id, merchant_id, amount, status, created_at FROM refunds WHERE invoice_id = \\$1 ORDER BY id ASC").
		WithArgs("inv-1").
		WillReturnRows(rows)

	_, err = repo.ListByInvoiceID(context.Background(), "inv-1")
	require.Error(t, err)
}

// ==================== Postgres: topup_repository_pg.go ====================

func TestPostgresTopUpRepository_FindByID_ScanErrorNonNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)

	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE id = \\$1").
		WithArgs("tu-1").
		WillReturnError(errors.New("connection reset"))

	_, err = repo.FindByID(context.Background(), "tu-1")
	require.Error(t, err)
	assert.False(t, errors.Is(err, appTopUp.ErrNotFound))
}

func TestPostgresTopUpRepository_FindByRequestKey_ScanErrorNonNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)

	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE merchant_id = \\$1 AND request_key = \\$2").
		WithArgs("m-1", "req-1").
		WillReturnError(errors.New("connection reset"))

	_, err = repo.FindByRequestKey(context.Background(), "m-1", "req-1")
	require.Error(t, err)
	assert.False(t, errors.Is(err, appTopUp.ErrNotFound))
}

func TestPostgresTopUpRepository_ListByMerchantID_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresTopUpRepository(db)

	rows := sqlmock.NewRows([]string{"id"}).AddRow("tu-1")
	mock.ExpectQuery("SELECT id, merchant_id, amount, status, request_key FROM top_ups WHERE merchant_id = \\$1 ORDER BY id ASC").
		WithArgs("m-1").
		WillReturnRows(rows)

	_, err = repo.ListByMerchantID(context.Background(), "m-1")
	require.Error(t, err)
}

// ==================== Postgres: wallet_repository_pg.go ====================

func TestPostgresWalletRepository_FindByMerchantID_ScanErrorNonNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresWalletRepository(db)

	mock.ExpectQuery("SELECT id, merchant_id, balance FROM wallets WHERE merchant_id = \\$1").
		WithArgs("m-1").
		WillReturnError(errors.New("connection reset"))

	_, err = repo.FindByMerchantID(context.Background(), "m-1")
	require.Error(t, err)
	assert.False(t, errors.Is(err, appWallet.ErrNotFound))
}

// ==================== Postgres: user_repository_pg.go ====================

func TestPostgresUserRepository_CreateWithMerchant_BeginError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	repo := NewPostgresUserRepository(db)
	u := &domainUser.User{ID: "u-1", Name: "Jane", Email: "jane@example.com", PasswordHash: "hash", Role: domainUser.RoleMerchant}

	mock.ExpectBegin().WillReturnError(errors.New("connection reset"))

	err = repo.CreateWithMerchant(u)
	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// sanity check that sql import is used (kept for parity with other test files
// that construct sql.NullTime / sql.ErrNoRows style helpers).
var _ = sql.ErrNoRows
