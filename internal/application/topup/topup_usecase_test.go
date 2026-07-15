package topup_test

import (
	"context"
	"errors"
	"testing"

	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopUpRequest(t *testing.T) {
	svc, _ := newTopUpService()
	model, err := svc.Request(context.Background(), "m-1", 500, "req-1")
	require.NoError(t, err)
	assert.Equal(t, topup.StatusPending, model.Status)
}

func TestTopUpHistory_ReturnsMerchantScopedList(t *testing.T) {
	svc, _ := newTopUpService()
	_, err := svc.Request(context.Background(), "m-1", 500, "req-history-1")
	require.NoError(t, err)

	history, err := svc.History(context.Background(), "m-1")
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "m-1", history[0].MerchantID)
}

func TestTopUpHistory_EmptyForUnknownMerchant(t *testing.T) {
	svc, _ := newTopUpService()
	history, err := svc.History(context.Background(), "unknown-merchant")
	require.NoError(t, err)
	assert.Len(t, history, 0)
}

func TestTopUpAdminSuccessFailureStatus(t *testing.T) {
	svc, _ := newTopUpService()
	model, _ := svc.Request(context.Background(), "m-1", 500, "req-1")
	updated, err := svc.AdminUpdate(context.Background(), model.ID, true)
	require.NoError(t, err)
	assert.Equal(t, topup.StatusSuccess, updated.Status)

	model2, _ := svc.Request(context.Background(), "m-1", 700, "req-2")
	updated2, err := svc.AdminUpdate(context.Background(), model2.ID, false)
	require.NoError(t, err)
	assert.Equal(t, topup.StatusFailed, updated2.Status)
}

func TestTopUpWalletBalanceIncrement(t *testing.T) {
	svc, walletRepo := newTopUpService()
	model, _ := svc.Request(context.Background(), "m-1", 500, "req-1")
	_, err := svc.AdminUpdate(context.Background(), model.ID, true)
	require.NoError(t, err)
	w, err := walletRepo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1500), w.Balance)
}

func TestTopUpFailedNoBalanceChange(t *testing.T) {
	svc, walletRepo := newTopUpService()
	model, _ := svc.Request(context.Background(), "m-1", 500, "req-1")
	_, err := svc.AdminUpdate(context.Background(), model.ID, false)
	require.NoError(t, err)
	w, err := walletRepo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1000), w.Balance)
}

func TestTopUpDuplicateRetryHandling(t *testing.T) {
	svc, _ := newTopUpService()
	first, err := svc.Request(context.Background(), "m-1", 500, "idempotency-key-1")
	require.NoError(t, err)
	second, err := svc.Request(context.Background(), "m-1", 500, "idempotency-key-1")
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
}

func TestAtomicTopUpCommitSuccess(t *testing.T) {
	svc, walletRepo := newTopUpService()
	model, _ := svc.Request(context.Background(), "m-1", 200, "atomic-req-1")
	_, err := svc.AdminUpdate(context.Background(), model.ID, true)
	require.NoError(t, err)
	w, err := walletRepo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(1200), w.Balance)
}

func TestAtomicTopUpRollbackPreventsPartialWrite(t *testing.T) {
	baseRepo := database.NewInMemoryTopUpRepository(nil)
	failRepo := &failingTopUpRepository{inner: baseRepo, failOnSuccessSave: true}
	walletRepo := database.NewInMemoryWalletRepository([]*wallet.Wallet{{ID: "w-1", MerchantID: "m-1", Balance: 1000}})
	svc := appTopUp.NewService(failRepo, walletRepo)

	model, _ := svc.Request(context.Background(), "m-1", 300, "atomic-req-2")
	_, err := svc.AdminUpdate(context.Background(), model.ID, true)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errForcedTopUpSave))

	w, wErr := walletRepo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, wErr)
	assert.Equal(t, int64(1000), w.Balance)
	persisted, pErr := baseRepo.FindByID(context.Background(), model.ID)
	require.NoError(t, pErr)
	assert.Equal(t, topup.StatusPending, persisted.Status)
}

func newTopUpService() (*appTopUp.Service, *database.InMemoryWalletRepository) {
	walletRepo := database.NewInMemoryWalletRepository([]*wallet.Wallet{
		{ID: "w-1", MerchantID: "m-1", Balance: 1000},
	})
	topupRepo := database.NewInMemoryTopUpRepository(nil)
	return appTopUp.NewService(topupRepo, walletRepo), walletRepo
}

var errForcedTopUpSave = errors.New("forced topup save failure")

type failingTopUpRepository struct {
	inner             *database.InMemoryTopUpRepository
	failOnSuccessSave bool
}

func (r *failingTopUpRepository) Create(ctx context.Context, model *topup.TopUp) error {
	return r.inner.Create(ctx, model)
}
func (r *failingTopUpRepository) FindByID(ctx context.Context, id string) (*topup.TopUp, error) {
	return r.inner.FindByID(ctx, id)
}
func (r *failingTopUpRepository) FindByRequestKey(ctx context.Context, merchantID, requestKey string) (*topup.TopUp, error) {
	return r.inner.FindByRequestKey(ctx, merchantID, requestKey)
}
func (r *failingTopUpRepository) ListByMerchantID(ctx context.Context, merchantID string) ([]*topup.TopUp, error) {
	return r.inner.ListByMerchantID(ctx, merchantID)
}
func (r *failingTopUpRepository) Save(ctx context.Context, model *topup.TopUp) error {
	if r.failOnSuccessSave && model.Status == topup.StatusSuccess {
		return errForcedTopUpSave
	}
	return r.inner.Save(ctx, model)
}
