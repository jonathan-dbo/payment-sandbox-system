package database

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositoryInMemoryAdaptersBehavior(t *testing.T) {
	ctx := context.Background()

	invoiceRepo := NewInMemoryInvoiceRepository(nil)
	inv := &invoice.Invoice{
		ID:            "inv-1",
		InvoiceNumber: "INV-001",
		MerchantID:    "m-1",
		Amount:        1000,
		Currency:      "USD",
		Status:        invoice.StatusPending,
		PaymentToken:  "tok-1",
		CreatedAt:     time.Now().UTC(),
	}
	require.NoError(t, invoiceRepo.Create(ctx, inv))
	foundInv, err := invoiceRepo.FindByInvoiceNumber(ctx, "INV-001")
	require.NoError(t, err)
	assert.Equal(t, "m-1", foundInv.MerchantID)
	list, err := invoiceRepo.List(ctx, appInvoice.ListFilter{MerchantID: "m-1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, list, 1)

	paymentRepo := NewInMemoryPaymentRepository(nil)
	intent := &paymentintent.PaymentIntent{ID: "pi-1", Status: paymentintent.StatusPending}
	require.NoError(t, paymentRepo.Create(ctx, intent))
	intent.Status = paymentintent.StatusSuccess
	require.NoError(t, paymentRepo.Save(ctx, intent))
	foundPayment, err := paymentRepo.FindByID(ctx, "pi-1")
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusSuccess, foundPayment.Status)

	refundRepo := NewInMemoryRefundRepository(nil)
	rf := &refund.Refund{ID: "rf-1", Status: refund.StatusRequested}
	require.NoError(t, refundRepo.Create(ctx, rf))
	rf.Status = refund.StatusApproved
	require.NoError(t, refundRepo.Save(ctx, rf))
	foundRefund, err := refundRepo.FindByID(ctx, "rf-1")
	require.NoError(t, err)
	assert.Equal(t, refund.StatusApproved, foundRefund.Status)

	walletRepo := NewInMemoryWalletRepository(nil)
	w := &wallet.Wallet{ID: "w-1", MerchantID: "m-1", Balance: 5000}
	require.NoError(t, walletRepo.Create(ctx, w))
	w.Balance = 9000
	require.NoError(t, walletRepo.Save(ctx, w))
	foundWallet, err := walletRepo.FindByMerchantID(ctx, "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(9000), foundWallet.Balance)
}

func TestRepositoryInMemoryConcurrencySafety(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryInvoiceRepository(nil)
	const workers = 50
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := range workers {
		go func(i int) {
			defer wg.Done()
			inv := &invoice.Invoice{
				ID:            fmt.Sprintf("inv-%d", i),
				InvoiceNumber: fmt.Sprintf("INV-%03d", i),
				MerchantID:    "m-1",
				Amount:        int64(1000 + i),
				Currency:      "USD",
				Status:        invoice.StatusPending,
				PaymentToken:  fmt.Sprintf("tok-%d", i),
				CreatedAt:     time.Now().UTC().Add(time.Duration(i) * time.Millisecond),
			}
			_ = repo.Create(ctx, inv)
		}(i)
	}

	wg.Wait()
	items, err := repo.List(ctx, appInvoice.ListFilter{MerchantID: "m-1", Page: 1, PageSize: workers})
	require.NoError(t, err)
	assert.Len(t, items, workers)
}
