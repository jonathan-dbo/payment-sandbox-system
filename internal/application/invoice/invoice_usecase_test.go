package invoice_test

import (
	"context"
	"testing"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceCreateHappyPath(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	inv, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{
		MerchantID: "m1",
		Amount:     1000,
		Currency:   "usd",
	})
	require.NoError(t, err)
	assert.Equal(t, "m1", inv.MerchantID)
	assert.Equal(t, "USD", inv.Currency)
	assert.Equal(t, invoice.StatusPending, inv.Status)
	assert.NotEmpty(t, inv.InvoiceNumber)
	assert.NotEmpty(t, inv.PaymentToken)
}

func TestInvoiceNumberUniqueness(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	first, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)
	second, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)
	assert.NotEqual(t, first.InvoiceNumber, second.InvoiceNumber)
}

func TestInvoiceListWithFilters(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	_, _ = svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	_, _ = svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m2", Amount: 2000, Currency: "USD"})

	list, err := svc.List(context.Background(), appInvoice.ListFilter{
		MerchantID: "m1",
		Page:       1,
		PageSize:   10,
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "m1", list[0].MerchantID)
}

func TestInvoicePaginationBoundaries(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	for i := range 3 {
		_, _ = svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: int64(1000 + i), Currency: "USD"})
	}

	page1, err := svc.List(context.Background(), appInvoice.ListFilter{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := svc.List(context.Background(), appInvoice.ListFilter{Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, page2, 1)

	page3, err := svc.List(context.Background(), appInvoice.ListFilter{Page: 3, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, page3, 0)
}

func TestInvoicePaymentTokenUniqueness(t *testing.T) {
	svc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	first, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)
	second, err := svc.Create(context.Background(), appInvoice.CreateInvoiceInput{MerchantID: "m1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)
	assert.NotEqual(t, first.PaymentToken, second.PaymentToken)
}
