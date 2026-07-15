package topup_test

import (
	"context"
	"errors"
	"testing"

	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopUpAdminUpdate_NotFound(t *testing.T) {
	svc, _ := newTopUpService()
	_, err := svc.AdminUpdate(context.Background(), "missing-topup", true)
	require.Error(t, err)
}

func TestTopUpAdminUpdate_AutoCreatesWalletWhenMissing(t *testing.T) {
	walletRepo := database.NewInMemoryWalletRepository(nil) // no wallet seeded
	topupRepo := database.NewInMemoryTopUpRepository(nil)
	svc := appTopUp.NewService(topupRepo, walletRepo)

	model, err := svc.Request(context.Background(), "m-nowallet", 300, "req-nowallet")
	require.NoError(t, err)

	updated, err := svc.AdminUpdate(context.Background(), model.ID, true)
	require.NoError(t, err)
	assert.Equal(t, topup.StatusSuccess, updated.Status)

	w, err := walletRepo.FindByMerchantID(context.Background(), "m-nowallet")
	require.NoError(t, err)
	assert.Equal(t, int64(300), w.Balance)
}

func TestTopUpAdminUpdate_WalletSaveFailureRollsBack(t *testing.T) {
	baseWalletRepo := database.NewInMemoryWalletRepository(nil)
	failWalletRepo := &failingWalletRepository{inner: baseWalletRepo, failOnSave: true}
	topupRepo := database.NewInMemoryTopUpRepository(nil)
	svc := appTopUp.NewService(topupRepo, failWalletRepo)

	model, err := svc.Request(context.Background(), "m-failwallet", 300, "req-failwallet")
	require.NoError(t, err)

	_, err = svc.AdminUpdate(context.Background(), model.ID, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "atomic topup flow failed at wallet save")
}

func TestTopUpAdminUpdate_InvalidTransitionRejected(t *testing.T) {
	svc, _ := newTopUpService()
	model, err := svc.Request(context.Background(), "m-1", 200, "req-transition")
	require.NoError(t, err)

	_, err = svc.AdminUpdate(context.Background(), model.ID, true)
	require.NoError(t, err)

	// Already SUCCESS (terminal); marking failed again should be an invalid transition.
	_, err = svc.AdminUpdate(context.Background(), model.ID, false)
	require.Error(t, err)
}

// failingWalletRepository wraps an in-memory wallet repo but forces Save to
// fail, to exercise AdminUpdate's atomic-flow rollback-on-wallet-save-failure path.
type failingWalletRepository struct {
	inner      *database.InMemoryWalletRepository
	failOnSave bool
}

func (r *failingWalletRepository) Create(ctx context.Context, model *domainWallet.Wallet) error {
	return r.inner.Create(ctx, model)
}
func (r *failingWalletRepository) FindByMerchantID(ctx context.Context, merchantID string) (*domainWallet.Wallet, error) {
	return r.inner.FindByMerchantID(ctx, merchantID)
}
func (r *failingWalletRepository) Save(ctx context.Context, model *domainWallet.Wallet) error {
	if r.failOnSave {
		return errForcedWalletSave
	}
	return r.inner.Save(ctx, model)
}

var errForcedWalletSave = errors.New("forced wallet save failure")
