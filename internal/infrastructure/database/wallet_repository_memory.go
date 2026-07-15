// Package database contains in-memory and SQL repositories.
package database

import (
	"context"
	"sync"

	appWallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	domainWallet "github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
)

type InMemoryWalletRepository struct {
	mu      sync.RWMutex
	wallets map[string]*domainWallet.Wallet
}

func NewInMemoryWalletRepository(seed []*domainWallet.Wallet) *InMemoryWalletRepository {
	repo := &InMemoryWalletRepository{wallets: map[string]*domainWallet.Wallet{}}
	for _, model := range seed {
		copy := *model
		repo.wallets[copy.MerchantID] = &copy
	}
	return repo
}

func (r *InMemoryWalletRepository) Create(_ context.Context, model *domainWallet.Wallet) error {
	return r.Save(context.Background(), model)
}

func (r *InMemoryWalletRepository) FindByMerchantID(_ context.Context, merchantID string) (*domainWallet.Wallet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	model, ok := r.wallets[merchantID]
	if !ok {
		return nil, appWallet.ErrNotFound
	}
	copy := *model
	return &copy, nil
}

func (r *InMemoryWalletRepository) Save(_ context.Context, model *domainWallet.Wallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *model
	r.wallets[model.MerchantID] = &copy
	return nil
}
