package refund_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appWallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errForcedRefundSaveErrors2 = errors.New("forced refund save failure (errors2)")

// saveFailingRefundRepository fails Save unconditionally, to exercise
// Approve/Reject's repo.Save error-propagation branches.
type saveFailingRefundRepository struct {
	inner *database.InMemoryRefundRepository
}

func (r *saveFailingRefundRepository) Create(ctx context.Context, model *refund.Refund) error {
	return r.inner.Create(ctx, model)
}
func (r *saveFailingRefundRepository) FindByID(ctx context.Context, id string) (*refund.Refund, error) {
	return r.inner.FindByID(ctx, id)
}
func (r *saveFailingRefundRepository) List(ctx context.Context, filter appRefund.ListFilter) ([]*refund.Refund, error) {
	return r.inner.List(ctx, filter)
}
func (r *saveFailingRefundRepository) ListByInvoiceID(ctx context.Context, invoiceID string) ([]*refund.Refund, error) {
	return r.inner.ListByInvoiceID(ctx, invoiceID)
}
func (r *saveFailingRefundRepository) Save(ctx context.Context, model *refund.Refund) error {
	return errForcedRefundSaveErrors2
}

func newRefundServiceWithInvoices(t *testing.T, invoices []*invoice.Invoice, wallets []*wallet.Wallet) (*appRefund.Service, *database.InMemoryRefundRepository, *database.InMemoryWalletRepository) {
	t.Helper()
	refundRepo := database.NewInMemoryRefundRepository(nil)
	walletRepo := database.NewInMemoryWalletRepository(wallets)
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(invoices))
	svc := appRefund.NewService(refundRepo, walletRepo, invoiceSvc)
	return svc, refundRepo, walletRepo
}

func TestRefundApprove_RepoSaveErrorPropagates(t *testing.T) {
	baseRepo := database.NewInMemoryRefundRepository([]*refund.Refund{
		{ID: "rf-1", InvoiceID: "inv-1", MerchantID: "m-1", Amount: 100, Status: refund.StatusRequested},
	})
	failRepo := &saveFailingRefundRepository{inner: baseRepo}
	walletRepo := database.NewInMemoryWalletRepository(nil)
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	svc := appRefund.NewService(failRepo, walletRepo, invoiceSvc)

	_, err := svc.Approve(context.Background(), "rf-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errForcedRefundSaveErrors2))
}

func TestRefundReject_RepoSaveErrorPropagates(t *testing.T) {
	baseRepo := database.NewInMemoryRefundRepository([]*refund.Refund{
		{ID: "rf-1", InvoiceID: "inv-1", MerchantID: "m-1", Amount: 100, Status: refund.StatusRequested},
	})
	failRepo := &saveFailingRefundRepository{inner: baseRepo}
	walletRepo := database.NewInMemoryWalletRepository(nil)
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	svc := appRefund.NewService(failRepo, walletRepo, invoiceSvc)

	_, err := svc.Reject(context.Background(), "rf-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errForcedRefundSaveErrors2))
}

func TestRefundApprove_AlreadyApprovedIsIdempotent(t *testing.T) {
	svc, _, _ := newRefundServiceWithInvoices(t, []*invoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}, []*wallet.Wallet{{ID: "w-1", MerchantID: "m-1", Balance: 0}})
	model, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 100)
	require.NoError(t, err)
	_, err = svc.Approve(context.Background(), model.ID)
	require.NoError(t, err)

	// approving again is idempotent per the domain's Approve() semantics.
	again, err := svc.Approve(context.Background(), model.ID)
	require.NoError(t, err)
	assert.Equal(t, refund.StatusApproved, again.Status)
}

func TestRefundReject_FromTerminalStateReturnsError(t *testing.T) {
	svc, _, _ := newRefundServiceWithInvoices(t, []*invoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}, []*wallet.Wallet{{ID: "w-1", MerchantID: "m-1", Balance: 0}})
	model, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 100)
	require.NoError(t, err)
	_, err = svc.Approve(context.Background(), model.ID)
	require.NoError(t, err)
	_, err = svc.Process(context.Background(), model.ID, true)
	require.NoError(t, err)

	// SUCCESS is terminal; rejecting should surface the domain invalid-transition error.
	_, err = svc.Reject(context.Background(), model.ID)
	require.Error(t, err)
}

func TestRefundProcess_WalletLookupNonNotFoundErrorPropagates(t *testing.T) {
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}))
	refundRepo := database.NewInMemoryRefundRepository(nil)
	walletRepo := &walletLookupErrorRepository{failWith: errors.New("wallet lookup boom")}
	svc := appRefund.NewService(refundRepo, walletRepo, invoiceSvc)

	model, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 100)
	require.NoError(t, err)
	_, err = svc.Approve(context.Background(), model.ID)
	require.NoError(t, err)

	_, err = svc.Process(context.Background(), model.ID, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wallet lookup boom")
}

func TestRefundProcess_WalletCreateErrorPropagates(t *testing.T) {
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}))
	refundRepo := database.NewInMemoryRefundRepository(nil)
	walletRepo := &walletCreateErrorRepository{failWith: errors.New("wallet create boom")}
	svc := appRefund.NewService(refundRepo, walletRepo, invoiceSvc)

	model, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 100)
	require.NoError(t, err)
	_, err = svc.Approve(context.Background(), model.ID)
	require.NoError(t, err)

	_, err = svc.Process(context.Background(), model.ID, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wallet create boom")
}

func TestRefundProcess_RejectedFailure_RepoSaveErrorPropagates(t *testing.T) {
	baseRepo := database.NewInMemoryRefundRepository([]*refund.Refund{
		{ID: "rf-1", InvoiceID: "inv-1", MerchantID: "m-1", Amount: 100, Status: refund.StatusApproved},
	})
	failRepo := &saveFailingRefundRepository{inner: baseRepo}
	walletRepo := database.NewInMemoryWalletRepository(nil)
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	svc := appRefund.NewService(failRepo, walletRepo, invoiceSvc)

	_, err := svc.Process(context.Background(), "rf-1", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errForcedRefundSaveErrors2))
}

func TestRefundProcess_MarkFailedInvalidTransitionRejected(t *testing.T) {
	svc, _, _ := newRefundServiceWithInvoices(t, []*invoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}, []*wallet.Wallet{{ID: "w-1", MerchantID: "m-1", Balance: 0}})
	model, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 100)
	require.NoError(t, err)
	// requested (not approved) refund cannot MarkFailed directly.
	_, err = svc.Process(context.Background(), model.ID, false)
	require.Error(t, err)
}

// walletLookupErrorRepository fails FindByMerchantID with a non-ErrNotFound
// error, to exercise Process's `else if err != nil` propagation branch.
type walletLookupErrorRepository struct {
	failWith error
}

func (r *walletLookupErrorRepository) Create(ctx context.Context, model *wallet.Wallet) error { return nil }
func (r *walletLookupErrorRepository) FindByMerchantID(ctx context.Context, merchantID string) (*wallet.Wallet, error) {
	return nil, r.failWith
}
func (r *walletLookupErrorRepository) Save(ctx context.Context, model *wallet.Wallet) error { return nil }

// walletCreateErrorRepository reports ErrNotFound on lookup (so the service
// attempts auto-creation) but fails wallet creation.
type walletCreateErrorRepository struct {
	failWith error
}

func (r *walletCreateErrorRepository) Create(ctx context.Context, model *wallet.Wallet) error {
	return r.failWith
}
func (r *walletCreateErrorRepository) FindByMerchantID(ctx context.Context, merchantID string) (*wallet.Wallet, error) {
	return nil, appWallet.ErrNotFound
}
func (r *walletCreateErrorRepository) Save(ctx context.Context, model *wallet.Wallet) error { return nil }
