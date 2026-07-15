package database

import (
	"context"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	domainInvoice "github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	domainPaymentIntent "github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	domainRefund "github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	domainTopUp "github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
	domainUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -------------------- Invoice memory repo edge cases --------------------

func TestInMemoryInvoiceRepository_FindByID_NotFound(t *testing.T) {
	repo := NewInMemoryInvoiceRepository(nil)
	_, err := repo.FindByID(context.Background(), "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, appInvoice.ErrNotFound)
}

func TestInMemoryInvoiceRepository_FindByPaymentToken_NotFound(t *testing.T) {
	repo := NewInMemoryInvoiceRepository(nil)
	_, err := repo.FindByPaymentToken(context.Background(), "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, appInvoice.ErrNotFound)
}

func TestInMemoryInvoiceRepository_Delete_NotFound(t *testing.T) {
	repo := NewInMemoryInvoiceRepository(nil)
	err := repo.Delete(context.Background(), "missing")
	require.Error(t, err)
}

func TestInMemoryInvoiceRepository_Save_UpdatesExisting(t *testing.T) {
	seed := []*domainInvoice.Invoice{{ID: "inv-1", MerchantID: "m-1", Amount: 100, Status: domainInvoice.StatusPending, CreatedAt: time.Now().UTC()}}
	repo := NewInMemoryInvoiceRepository(seed)
	inv, err := repo.FindByID(context.Background(), "inv-1")
	require.NoError(t, err)
	inv.Amount = 999
	require.NoError(t, repo.Save(context.Background(), inv))
	updated, err := repo.FindByID(context.Background(), "inv-1")
	require.NoError(t, err)
	assert.Equal(t, int64(999), updated.Amount)
}

func TestInMemoryInvoiceRepository_List_StatusFilterCaseInsensitive(t *testing.T) {
	seed := []*domainInvoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", Status: domainInvoice.StatusPaid, CreatedAt: time.Now().UTC()},
		{ID: "inv-2", MerchantID: "m-1", Status: domainInvoice.StatusPending, CreatedAt: time.Now().UTC()},
	}
	repo := NewInMemoryInvoiceRepository(seed)
	items, err := repo.List(context.Background(), appInvoice.ListFilter{Status: "paid", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "inv-1", items[0].ID)
}

// -------------------- Payment intent memory repo edge cases --------------------

func TestInMemoryPaymentRepository_List_Empty(t *testing.T) {
	repo := NewInMemoryPaymentRepository(nil)
	items, err := repo.List(context.Background(), appPayment.ListFilter{})
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestInMemoryPaymentRepository_List_FiltersByInvoiceAndDateRange(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainPaymentIntent.PaymentIntent{
		{ID: "pi-1", InvoiceID: "inv-1", Status: domainPaymentIntent.StatusPending, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "pi-2", InvoiceID: "inv-1", Status: domainPaymentIntent.StatusPending, CreatedAt: now},
		{ID: "pi-3", InvoiceID: "inv-2", Status: domainPaymentIntent.StatusPending, CreatedAt: now},
	}
	repo := NewInMemoryPaymentRepository(seed)
	start := now.Add(-time.Hour)
	items, err := repo.List(context.Background(), appPayment.ListFilter{InvoiceID: "inv-1", StartDate: &start})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "pi-2", items[0].ID)
}

func TestInMemoryPaymentRepository_FindByID_NotFound(t *testing.T) {
	repo := NewInMemoryPaymentRepository(nil)
	_, err := repo.FindByID(context.Background(), "missing")
	require.Error(t, err)
}

// -------------------- Refund memory repo edge cases --------------------

func TestInMemoryRefundRepository_List_FiltersByMerchantAndDateRange(t *testing.T) {
	now := time.Now().UTC()
	seed := []*domainRefund.Refund{
		{ID: "rf-1", MerchantID: "m-1", CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "rf-2", MerchantID: "m-1", CreatedAt: now},
		{ID: "rf-3", MerchantID: "m-2", CreatedAt: now},
	}
	repo := NewInMemoryRefundRepository(seed)
	start := now.Add(-time.Hour)
	items, err := repo.List(context.Background(), appRefund.ListFilter{MerchantID: "m-1", StartDate: &start})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "rf-2", items[0].ID)
}

func TestInMemoryRefundRepository_ListByInvoiceID_Empty(t *testing.T) {
	repo := NewInMemoryRefundRepository(nil)
	items, err := repo.ListByInvoiceID(context.Background(), "missing")
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestInMemoryRefundRepository_FindByID_NotFound(t *testing.T) {
	repo := NewInMemoryRefundRepository(nil)
	_, err := repo.FindByID(context.Background(), "missing")
	require.Error(t, err)
}

// -------------------- TopUp memory repo edge cases --------------------

func TestInMemoryTopUpRepository_FindByRequestKey_EmptyKeyNeverMatches(t *testing.T) {
	seed := []*domainTopUp.TopUp{{ID: "tu-1", MerchantID: "m-1", RequestKey: ""}}
	repo := NewInMemoryTopUpRepository(seed)
	_, err := repo.FindByRequestKey(context.Background(), "m-1", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, appTopUp.ErrNotFound)
}

func TestInMemoryTopUpRepository_FindByRequestKey_NotFound(t *testing.T) {
	repo := NewInMemoryTopUpRepository(nil)
	_, err := repo.FindByRequestKey(context.Background(), "m-1", "req-1")
	require.Error(t, err)
}

func TestInMemoryTopUpRepository_ListByMerchantID_AllWhenEmptyFilter(t *testing.T) {
	seed := []*domainTopUp.TopUp{
		{ID: "tu-1", MerchantID: "m-1"},
		{ID: "tu-2", MerchantID: "m-2"},
	}
	repo := NewInMemoryTopUpRepository(seed)
	items, err := repo.ListByMerchantID(context.Background(), "")
	require.NoError(t, err)
	assert.Len(t, items, 2)
}

func TestInMemoryTopUpRepository_FindByID_NotFound(t *testing.T) {
	repo := NewInMemoryTopUpRepository(nil)
	_, err := repo.FindByID(context.Background(), "missing")
	require.Error(t, err)
}

// -------------------- User memory repo edge cases --------------------

func TestInMemoryUserRepository_SeedPopulatesMerchantIndex(t *testing.T) {
	seed := []*domainUser.User{
		{ID: "u-1", Email: "merchant@example.com", Role: domainUser.RoleMerchant},
		{ID: "u-2", Email: "user@example.com", Role: domainUser.RoleUser},
	}
	repo := NewInMemoryUserRepository(seed)
	assert.True(t, repo.MerchantExistsForUser("u-1"))
	assert.False(t, repo.MerchantExistsForUser("u-2"))
	assert.False(t, repo.MerchantExistsForUser("missing"))
}

func TestInMemoryUserRepository_FindByID_NotFound(t *testing.T) {
	repo := NewInMemoryUserRepository(nil)
	_, err := repo.FindByID("missing")
	require.Error(t, err)
}

func TestInMemoryUserRepository_FindByEmail_CaseInsensitive(t *testing.T) {
	seed := []*domainUser.User{{ID: "u-1", Email: "Jane@Example.com", Role: domainUser.RoleUser}}
	repo := NewInMemoryUserRepository(seed)
	u, err := repo.FindByEmail("jane@example.com")
	require.NoError(t, err)
	assert.Equal(t, "u-1", u.ID)
}

func TestInMemoryUserRepository_FindByEmail_NotFound(t *testing.T) {
	repo := NewInMemoryUserRepository(nil)
	_, err := repo.FindByEmail("missing@example.com")
	require.Error(t, err)
}

func TestInMemoryUserRepository_CreateWithMerchant_NonMerchantRoleNotIndexed(t *testing.T) {
	repo := NewInMemoryUserRepository(nil)
	u := &domainUser.User{ID: "u-1", Email: "plain@example.com", Role: domainUser.RoleUser}
	require.NoError(t, repo.CreateWithMerchant(u))
	assert.False(t, repo.MerchantExistsForUser("u-1"))
}
