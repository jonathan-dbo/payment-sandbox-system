package refund_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundRequestRefund_NilInvoiceServiceRejected(t *testing.T) {
	refundRepo := database.NewInMemoryRefundRepository(nil)
	walletRepo := database.NewInMemoryWalletRepository(nil)
	svc := appRefund.NewService(refundRepo, walletRepo, nil)

	_, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invoice service is required")
}

func TestRefundRequestRefund_InvoiceNotFound(t *testing.T) {
	svc, _ := newRefundService()
	_, err := svc.RequestRefund(context.Background(), "missing-invoice", "m-1", 100)
	require.Error(t, err)
}

func TestRefundRequestRefund_MerchantOwnershipMismatch(t *testing.T) {
	svc, _ := newRefundService()
	_, err := svc.RequestRefund(context.Background(), "inv-1", "not-the-owner", 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "merchant does not own this invoice")
}

func TestRefundRequestRefund_InvoiceNotPaidRejected(t *testing.T) {
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "inv-pending", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPending, CreatedAt: time.Now().UTC()},
	}))
	walletRepo := database.NewInMemoryWalletRepository([]*wallet.Wallet{{ID: "w-1", MerchantID: "m-1", Balance: 1000}})
	svc := appRefund.NewService(database.NewInMemoryRefundRepository(nil), walletRepo, invoiceSvc)

	_, err := svc.RequestRefund(context.Background(), "inv-pending", "m-1", 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refund allowed only for paid invoice")
}

func TestRefundRequestRefund_FullyRefundedRejected(t *testing.T) {
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "inv-full", MerchantID: "m-1", Amount: 300, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}))
	refundRepo := database.NewInMemoryRefundRepository([]*refund.Refund{
		{ID: "rf-existing", InvoiceID: "inv-full", MerchantID: "m-1", Amount: 300, Status: refund.StatusSuccess},
	})
	walletRepo := database.NewInMemoryWalletRepository([]*wallet.Wallet{{ID: "w-1", MerchantID: "m-1", Balance: 1000}})
	svc := appRefund.NewService(refundRepo, walletRepo, invoiceSvc)

	_, err := svc.RequestRefund(context.Background(), "inv-full", "m-1", 50)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invoice is fully refunded")
}

func TestRefundRequestRefund_ListByInvoiceIDError(t *testing.T) {
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "inv-err", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}))
	walletRepo := database.NewInMemoryWalletRepository(nil)
	failRepo := &listErrorRefundRepository{inner: database.NewInMemoryRefundRepository(nil), failWith: errors.New("list boom")}
	svc := appRefund.NewService(failRepo, walletRepo, invoiceSvc)

	_, err := svc.RequestRefund(context.Background(), "inv-err", "m-1", 100)
	require.Error(t, err)
	assert.Equal(t, "list boom", err.Error())
}

func TestRefundProcess_AutoCreatesWalletWhenMissing(t *testing.T) {
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "inv-nowallet", MerchantID: "m-nowallet", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}))
	walletRepo := database.NewInMemoryWalletRepository(nil) // no wallet seeded for m-nowallet
	refundRepo := database.NewInMemoryRefundRepository(nil)
	svc := appRefund.NewService(refundRepo, walletRepo, invoiceSvc)

	model, err := svc.RequestRefund(context.Background(), "inv-nowallet", "m-nowallet", 200)
	require.NoError(t, err)
	_, err = svc.Approve(context.Background(), model.ID)
	require.NoError(t, err)

	processed, err := svc.Process(context.Background(), model.ID, true)
	require.NoError(t, err)
	assert.Equal(t, refund.StatusSuccess, processed.Status)

	w, err := walletRepo.FindByMerchantID(context.Background(), "m-nowallet")
	require.NoError(t, err)
	assert.Equal(t, int64(200), w.Balance)
}

func TestRefundApprove_NotFound(t *testing.T) {
	svc, _ := newRefundService()
	_, err := svc.Approve(context.Background(), "missing-refund")
	require.Error(t, err)
}

func TestRefundReject_NotFound(t *testing.T) {
	svc, _ := newRefundService()
	_, err := svc.Reject(context.Background(), "missing-refund")
	require.Error(t, err)
}

func TestRefundProcess_NotFound(t *testing.T) {
	svc, _ := newRefundService()
	_, err := svc.Process(context.Background(), "missing-refund", true)
	require.Error(t, err)
}

// listErrorRefundRepository wraps an in-memory refund repo but forces
// ListByInvoiceID to fail, to exercise RequestRefund's error propagation path.
type listErrorRefundRepository struct {
	inner    *database.InMemoryRefundRepository
	failWith error
}

func (r *listErrorRefundRepository) Create(ctx context.Context, model *refund.Refund) error {
	return r.inner.Create(ctx, model)
}
func (r *listErrorRefundRepository) FindByID(ctx context.Context, id string) (*refund.Refund, error) {
	return r.inner.FindByID(ctx, id)
}
func (r *listErrorRefundRepository) List(ctx context.Context, filter appRefund.ListFilter) ([]*refund.Refund, error) {
	return r.inner.List(ctx, filter)
}
func (r *listErrorRefundRepository) ListByInvoiceID(ctx context.Context, invoiceID string) ([]*refund.Refund, error) {
	return nil, r.failWith
}
func (r *listErrorRefundRepository) Save(ctx context.Context, model *refund.Refund) error {
	return r.inner.Save(ctx, model)
}
