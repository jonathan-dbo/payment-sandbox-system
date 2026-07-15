package invoice_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceUpdate_HappyPath(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)

	updated, err := svc.Update(context.Background(), inv.ID, appInvoice.UpdateInvoiceInput{Amount: 2000, Currency: "eur"})
	require.NoError(t, err)
	assert.Equal(t, int64(2000), updated.Amount)
	assert.Equal(t, "EUR", updated.Currency)
}

func TestInvoiceUpdate_DefaultsCurrencyWhenEmpty(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)

	updated, err := svc.Update(context.Background(), inv.ID, appInvoice.UpdateInvoiceInput{Amount: 500, Currency: "  "})
	require.NoError(t, err)
	assert.Equal(t, "USD", updated.Currency)
}

func TestInvoiceUpdate_SetsDueDate(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)

	due := time.Now().Add(48 * time.Hour)
	updated, err := svc.Update(context.Background(), inv.ID, appInvoice.UpdateInvoiceInput{Amount: 1000, Currency: "USD", DueDate: &due})
	require.NoError(t, err)
	assert.WithinDuration(t, due.UTC(), updated.DueDate, time.Second)
}

func TestInvoiceUpdate_RejectsNonPendingInvoice(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)
	require.NoError(t, inv.MarkPaid())
	require.NoError(t, svc.Save(context.Background(), inv))

	_, err = svc.Update(context.Background(), inv.ID, appInvoice.UpdateInvoiceInput{Amount: 500, Currency: "USD"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only pending invoice can be updated")
}

func TestInvoiceUpdate_NotFound(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	_, err := svc.Update(context.Background(), "missing-id", appInvoice.UpdateInvoiceInput{Amount: 500, Currency: "USD"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestInvoiceDelete_HappyPath(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)

	require.NoError(t, svc.Delete(context.Background(), inv.ID))

	_, err = svc.GetByID(context.Background(), inv.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestInvoiceDelete_RejectsNonPendingInvoice(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)
	require.NoError(t, inv.MarkPaid())
	require.NoError(t, svc.Save(context.Background(), inv))

	err = svc.Delete(context.Background(), inv.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only pending invoice can be deleted")
}

func TestInvoiceDelete_NotFound(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	err := svc.Delete(context.Background(), "missing-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestInvoiceGetByID_NotFound(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	_, err := svc.GetByID(context.Background(), "missing-id")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestInvoiceResolvePaymentToken_NotFound(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	_, err := svc.ResolvePaymentToken(context.Background(), "missing-token")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestInvoiceRefreshExpiry_MarksExpiredWhenPastDueDate(t *testing.T) {
	repo := database.NewInMemoryInvoiceRepository(nil)
	svc := appInvoice.NewService(repo)
	past := time.Now().UTC().Add(-1 * time.Hour)
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD", DueDate: &past})
	require.NoError(t, err)

	fetched, err := svc.GetByID(context.Background(), inv.ID)
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusExpired, fetched.Status)

	// verify persisted status was saved, not just the returned copy
	persisted, err := repo.FindByID(context.Background(), inv.ID)
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusExpired, persisted.Status)
}

func TestInvoiceRefreshExpiry_NoOpWhenDueDateInFuture(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	future := time.Now().UTC().Add(24 * time.Hour)
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD", DueDate: &future})
	require.NoError(t, err)

	fetched, err := svc.GetByID(context.Background(), inv.ID)
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPending, fetched.Status)
}

func TestInvoiceRefreshExpiry_NoOpWhenNoDueDate(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)

	fetched, err := svc.GetByID(context.Background(), inv.ID)
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPending, fetched.Status)
}

func TestInvoiceRefreshExpiry_NoOpWhenAlreadyPaid(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	past := time.Now().UTC().Add(-1 * time.Hour)
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD", DueDate: &past})
	require.NoError(t, err)
	require.NoError(t, inv.MarkPaid())
	require.NoError(t, svc.Save(context.Background(), inv))

	fetched, err := svc.GetByID(context.Background(), inv.ID)
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPaid, fetched.Status)
}

func TestInvoiceListApplyingExpiryDuringPagination(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	past := time.Now().UTC().Add(-1 * time.Hour)
	_, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD", DueDate: &past})
	require.NoError(t, err)

	list, err := svc.List(context.Background(), appInvoice.ListFilter{MerchantID: "m1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, invoice.StatusExpired, list[0].Status)
}

func TestInvoiceCreate_DefaultsCurrencyWhenEmpty(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: ""})
	require.NoError(t, err)
	assert.Equal(t, "USD", inv.Currency)
}

func TestInvoiceCreate_SetsDueDate(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	due := time.Now().Add(72 * time.Hour)
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD", DueDate: &due})
	require.NoError(t, err)
	assert.WithinDuration(t, due.UTC(), inv.DueDate, time.Second)
}
