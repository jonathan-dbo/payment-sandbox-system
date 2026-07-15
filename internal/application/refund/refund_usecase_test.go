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

func TestRefundRequestHappyPath(t *testing.T) {
	svc, _ := newRefundService()
	model, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 500)
	require.NoError(t, err)
	assert.Equal(t, refund.StatusRequested, model.Status)
}

func TestRefundApproveRefuseTransitions(t *testing.T) {
	svc, _ := newRefundService()
	model, _ := svc.RequestRefund(context.Background(), "inv-1", "m-1", 500)

	approved, err := svc.Approve(context.Background(), model.ID)
	require.NoError(t, err)
	assert.Equal(t, refund.StatusApproved, approved.Status)

	model2, _ := svc.RequestRefund(context.Background(), "inv-2", "m-1", 500)
	rejected, err := svc.Reject(context.Background(), model2.ID)
	require.NoError(t, err)
	assert.Equal(t, refund.StatusRejected, rejected.Status)
}

func TestRefundProcessSuccessFailure(t *testing.T) {
	svc, _ := newRefundService()
	model, _ := svc.RequestRefund(context.Background(), "inv-1", "m-1", 500)
	_, _ = svc.Approve(context.Background(), model.ID)

	success, err := svc.Process(context.Background(), model.ID, true)
	require.NoError(t, err)
	assert.Equal(t, refund.StatusSuccess, success.Status)

	model2, _ := svc.RequestRefund(context.Background(), "inv-2", "m-1", 500)
	_, _ = svc.Approve(context.Background(), model2.ID)
	failed, err := svc.Process(context.Background(), model2.ID, false)
	require.NoError(t, err)
	assert.Equal(t, refund.StatusFailed, failed.Status)
}

func TestRefundInvalidTransitionRejection(t *testing.T) {
	svc, _ := newRefundService()
	model, _ := svc.RequestRefund(context.Background(), "inv-1", "m-1", 500)
	_, err := svc.Process(context.Background(), model.ID, true)
	require.Error(t, err)
}

func TestRefundWalletBalanceAdjustment(t *testing.T) {
	svc, walletRepo := newRefundService()
	model, _ := svc.RequestRefund(context.Background(), "inv-1", "m-1", 500)
	_, _ = svc.Approve(context.Background(), model.ID)
	_, err := svc.Process(context.Background(), model.ID, true)
	require.NoError(t, err)

	w, err := walletRepo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1500), w.Balance)
}

func TestAtomicRefundCommitSuccess(t *testing.T) {
	svc, walletRepo := newRefundService()
	model, _ := svc.RequestRefund(context.Background(), "inv-atomic-1", "m-1", 200)
	_, _ = svc.Approve(context.Background(), model.ID)
	_, err := svc.Process(context.Background(), model.ID, true)
	require.NoError(t, err)

	w, err := walletRepo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1200), w.Balance)
}

func TestAtomicRefundRollbackPreventsPartialWrite(t *testing.T) {
	baseRepo := database.NewInMemoryRefundRepository(nil)
	failRepo := &failingRefundRepository{inner: baseRepo, failOnSuccessSave: true}
	walletRepo := database.NewInMemoryWalletRepository([]*wallet.Wallet{{ID: "w-1", MerchantID: "m-1", Balance: 1000}})
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "inv-atomic-2", MerchantID: "m-1", Amount: 300, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}))
	svc := appRefund.NewService(failRepo, walletRepo, invoiceSvc)

	model, _ := svc.RequestRefund(context.Background(), "inv-atomic-2", "m-1", 300)
	_, _ = svc.Approve(context.Background(), model.ID)
	_, err := svc.Process(context.Background(), model.ID, true)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errForcedRefundSave))

	w, wErr := walletRepo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, wErr)
	assert.Equal(t, int64(1000), w.Balance)
	persisted, pErr := baseRepo.FindByID(context.Background(), model.ID)
	require.NoError(t, pErr)
	assert.Equal(t, refund.StatusApproved, persisted.Status)
}

func newRefundService() (*appRefund.Service, *database.InMemoryWalletRepository) {
	refundRepo := database.NewInMemoryRefundRepository(nil)
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository([]*invoice.Invoice{
		{ID: "inv-1", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
		{ID: "inv-2", MerchantID: "m-1", Amount: 500, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
		{ID: "inv-atomic-1", MerchantID: "m-1", Amount: 200, Status: invoice.StatusPaid, CreatedAt: time.Now().UTC()},
	}))
	walletRepo := database.NewInMemoryWalletRepository([]*wallet.Wallet{
		{ID: "w-1", MerchantID: "m-1", Balance: 1000},
	})
	return appRefund.NewService(refundRepo, walletRepo, invoiceSvc), walletRepo
}

func TestRefundRejectsAmountAboveInvoiceAmount(t *testing.T) {
	svc, _ := newRefundService()
	_, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 600)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds max refundable amount")
}

func TestRefundHistory_ReturnsMerchantScopedList(t *testing.T) {
	svc, _ := newRefundService()
	_, err := svc.RequestRefund(context.Background(), "inv-1", "m-1", 100)
	require.NoError(t, err)

	history, err := svc.History(context.Background(), "m-1")
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "m-1", history[0].MerchantID)
}

func TestRefundHistory_EmptyForUnknownMerchant(t *testing.T) {
	svc, _ := newRefundService()
	history, err := svc.History(context.Background(), "unknown-merchant")
	require.NoError(t, err)
	assert.Len(t, history, 0)
}

var errForcedRefundSave = errors.New("forced refund save failure")

type failingRefundRepository struct {
	inner             *database.InMemoryRefundRepository
	failOnSuccessSave bool
}

func (r *failingRefundRepository) Create(ctx context.Context, model *refund.Refund) error {
	return r.inner.Create(ctx, model)
}
func (r *failingRefundRepository) FindByID(ctx context.Context, id string) (*refund.Refund, error) {
	return r.inner.FindByID(ctx, id)
}
func (r *failingRefundRepository) List(ctx context.Context, filter appRefund.ListFilter) ([]*refund.Refund, error) {
	return r.inner.List(ctx, filter)
}
func (r *failingRefundRepository) ListByInvoiceID(ctx context.Context, invoiceID string) ([]*refund.Refund, error) {
	return r.inner.ListByInvoiceID(ctx, invoiceID)
}
func (r *failingRefundRepository) Save(ctx context.Context, model *refund.Refund) error {
	if r.failOnSuccessSave && model.Status == refund.StatusSuccess {
		return errForcedRefundSave
	}
	return r.inner.Save(ctx, model)
}
