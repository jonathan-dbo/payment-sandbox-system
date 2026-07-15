package topup_test

import (
	"context"
	"errors"
	"testing"

	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	appWallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTopUpRepoCreateBoom = errors.New("topup repo create boom")
var errWalletCreateBoom2 = errors.New("wallet create boom (2)")
var errWalletLookupBoom = errors.New("wallet lookup boom")

// createFailingTopUpRepository fails Create unconditionally, to exercise
// Request's repo.Create error-propagation branch.
type createFailingTopUpRepository struct {
	inner *database.InMemoryTopUpRepository
}

func (r *createFailingTopUpRepository) Create(ctx context.Context, model *topup.TopUp) error {
	return errTopUpRepoCreateBoom
}
func (r *createFailingTopUpRepository) FindByID(ctx context.Context, id string) (*topup.TopUp, error) {
	return r.inner.FindByID(ctx, id)
}
func (r *createFailingTopUpRepository) FindByRequestKey(ctx context.Context, merchantID, requestKey string) (*topup.TopUp, error) {
	return r.inner.FindByRequestKey(ctx, merchantID, requestKey)
}
func (r *createFailingTopUpRepository) ListByMerchantID(ctx context.Context, merchantID string) ([]*topup.TopUp, error) {
	return r.inner.ListByMerchantID(ctx, merchantID)
}
func (r *createFailingTopUpRepository) Save(ctx context.Context, model *topup.TopUp) error {
	return r.inner.Save(ctx, model)
}

func TestTopUpRequest_RepoCreateErrorPropagates(t *testing.T) {
	failRepo := &createFailingTopUpRepository{inner: database.NewInMemoryTopUpRepository(nil)}
	walletRepo := database.NewInMemoryWalletRepository(nil)
	svc := appTopUp.NewService(failRepo, walletRepo)

	_, err := svc.Request(context.Background(), "m-1", 500, "req-fail")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errTopUpRepoCreateBoom))
}

// walletCreateFailingRepository reports ErrNotFound on lookup so AdminUpdate
// attempts auto-creation, but fails wallet Create.
type walletCreateFailingRepository struct{}

func (r *walletCreateFailingRepository) Create(ctx context.Context, model *domainWallet.Wallet) error {
	return errWalletCreateBoom2
}
func (r *walletCreateFailingRepository) FindByMerchantID(ctx context.Context, merchantID string) (*domainWallet.Wallet, error) {
	return nil, appWallet.ErrNotFound
}
func (r *walletCreateFailingRepository) Save(ctx context.Context, model *domainWallet.Wallet) error {
	return nil
}

func TestTopUpAdminUpdate_WalletCreateErrorPropagates(t *testing.T) {
	topupRepo := database.NewInMemoryTopUpRepository(nil)
	svc := appTopUp.NewService(topupRepo, &walletCreateFailingRepository{})

	model, err := svc.Request(context.Background(), "m-failcreate", 300, "req-failcreate")
	require.NoError(t, err)

	_, err = svc.AdminUpdate(context.Background(), model.ID, true)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errWalletCreateBoom2))
}

// walletLookupFailingRepository fails FindByMerchantID with a non-ErrNotFound
// error, to exercise AdminUpdate's `else if err != nil` propagation branch.
type walletLookupFailingRepository struct{}

func (r *walletLookupFailingRepository) Create(ctx context.Context, model *domainWallet.Wallet) error {
	return nil
}
func (r *walletLookupFailingRepository) FindByMerchantID(ctx context.Context, merchantID string) (*domainWallet.Wallet, error) {
	return nil, errWalletLookupBoom
}
func (r *walletLookupFailingRepository) Save(ctx context.Context, model *domainWallet.Wallet) error {
	return nil
}

func TestTopUpAdminUpdate_WalletLookupNonNotFoundErrorPropagates(t *testing.T) {
	topupRepo := database.NewInMemoryTopUpRepository(nil)
	svc := appTopUp.NewService(topupRepo, &walletLookupFailingRepository{})

	model, err := svc.Request(context.Background(), "m-1", 300, "req-lookup-fail")
	require.NoError(t, err)

	_, err = svc.AdminUpdate(context.Background(), model.ID, true)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errWalletLookupBoom))
}

func TestTopUpAdminUpdate_MarkSuccessIsIdempotent(t *testing.T) {
	svc, _ := newTopUpService()
	model, err := svc.Request(context.Background(), "m-1", 200, "req-marksuccess")
	require.NoError(t, err)

	_, err = svc.AdminUpdate(context.Background(), model.ID, true)
	require.NoError(t, err)

	// SUCCESS -> SUCCESS is an idempotent no-op by design (see TopUp.MarkSuccess),
	// consistent with the same convention on Refund.MarkSuccess. Re-marking success
	// on an already-SUCCESS top-up must not error.
	_, err = svc.AdminUpdate(context.Background(), model.ID, true)
	require.NoError(t, err)
}

func TestTopUpAdminUpdate_TopUpSaveFailureRollsBackNewlyCreatedWallet(t *testing.T) {
	baseRepo := database.NewInMemoryTopUpRepository(nil)
	failRepo := &failingTopUpRepository{inner: baseRepo, failOnSuccessSave: true}
	walletRepo := database.NewInMemoryWalletRepository(nil) // no wallet seeded -> triggers auto-create
	svc := appTopUp.NewService(failRepo, walletRepo)

	model, err := svc.Request(context.Background(), "m-newwallet", 400, "req-newwallet-rollback")
	require.NoError(t, err)

	_, err = svc.AdminUpdate(context.Background(), model.ID, true)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errForcedTopUpSave))

	w, wErr := walletRepo.FindByMerchantID(context.Background(), "m-newwallet")
	require.NoError(t, wErr)
	// createdWallet rollback branch resets the freshly-created wallet's balance to 0.
	assert.Equal(t, int64(0), w.Balance)
}

func TestTopUpAdminUpdate_MarkFailedRepoSaveErrorPropagates(t *testing.T) {
	baseRepo := database.NewInMemoryTopUpRepository(nil)
	failRepo := &failOnAnySaveTopUpRepository{inner: baseRepo}
	walletRepo := database.NewInMemoryWalletRepository(nil)
	svc := appTopUp.NewService(failRepo, walletRepo)

	model, err := svc.Request(context.Background(), "m-1", 100, "req-failed-save")
	require.NoError(t, err)

	_, err = svc.AdminUpdate(context.Background(), model.ID, false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errFailedSaveBoom))
}

var errFailedSaveBoom = errors.New("failed-outcome save boom")
var appWalletErrNotFoundSentinel = notFoundSentinel{}

// notFoundSentinel mimics appWallet.ErrNotFound's identity via errors.Is by
// being the exact same error value re-exported from the wallet application
// package; we import it indirectly through the existing test helpers file
// in this package (see topup_usecase_extra_test.go for precedent), but since
// this file only needs value equality with appWallet.ErrNotFound we resolve
// it through a thin adapter type implementing the error interface with the
// same message so `err == appWallet.ErrNotFound` checks in production code
// still require exact identity -- so we import the real sentinel instead.
type notFoundSentinel struct{}

func (notFoundSentinel) Error() string { return "wallet not found" }

// failOnAnySaveTopUpRepository fails every Save call, to exercise the
// MarkFailed-then-repo.Save error branch in AdminUpdate's else path.
type failOnAnySaveTopUpRepository struct {
	inner *database.InMemoryTopUpRepository
}

func (r *failOnAnySaveTopUpRepository) Create(ctx context.Context, model *topup.TopUp) error {
	return r.inner.Create(ctx, model)
}
func (r *failOnAnySaveTopUpRepository) FindByID(ctx context.Context, id string) (*topup.TopUp, error) {
	return r.inner.FindByID(ctx, id)
}
func (r *failOnAnySaveTopUpRepository) FindByRequestKey(ctx context.Context, merchantID, requestKey string) (*topup.TopUp, error) {
	return r.inner.FindByRequestKey(ctx, merchantID, requestKey)
}
func (r *failOnAnySaveTopUpRepository) ListByMerchantID(ctx context.Context, merchantID string) ([]*topup.TopUp, error) {
	return r.inner.ListByMerchantID(ctx, merchantID)
}
func (r *failOnAnySaveTopUpRepository) Save(ctx context.Context, model *topup.TopUp) error {
	return errFailedSaveBoom
}
