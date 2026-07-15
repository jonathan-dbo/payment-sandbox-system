package wallet_test

import (
	"context"
	"errors"
	"testing"

	appWallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compile-time assertion that the in-memory repository satisfies the application contract.
var _ appWallet.Repository = (*database.InMemoryWalletRepository)(nil)

func TestWalletRepository_ErrNotFound(t *testing.T) {
	repo := database.NewInMemoryWalletRepository(nil)
	_, err := repo.FindByMerchantID(context.Background(), "missing-merchant")
	require.Error(t, err)
	assert.True(t, errors.Is(err, appWallet.ErrNotFound))
}

func TestWalletRepository_CreateAndFind(t *testing.T) {
	repo := database.NewInMemoryWalletRepository(nil)
	w := &domainWallet.Wallet{ID: "w-1", MerchantID: "m-1", Balance: 100}
	require.NoError(t, repo.Create(context.Background(), w))

	found, err := repo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(100), found.Balance)
}

func TestWalletRepository_Save(t *testing.T) {
	repo := database.NewInMemoryWalletRepository([]*domainWallet.Wallet{
		{ID: "w-1", MerchantID: "m-1", Balance: 100},
	})
	w, err := repo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	w.Balance += 50
	require.NoError(t, repo.Save(context.Background(), w))

	updated, err := repo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(150), updated.Balance)
}

func TestWalletRepository_SaveReturnsIndependentCopy(t *testing.T) {
	repo := database.NewInMemoryWalletRepository(nil)
	w := &domainWallet.Wallet{ID: "w-1", MerchantID: "m-1", Balance: 10}
	require.NoError(t, repo.Create(context.Background(), w))

	// Mutating the caller's pointer after Create should not affect stored state.
	w.Balance = 9999
	found, err := repo.FindByMerchantID(context.Background(), "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(10), found.Balance)
}
