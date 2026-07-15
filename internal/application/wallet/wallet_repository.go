package wallet

import (
	"context"
	"errors"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
)

var ErrNotFound = errors.New("wallet not found")

type Repository interface {
	Create(ctx context.Context, model *wallet.Wallet) error
	FindByMerchantID(ctx context.Context, merchantID string) (*wallet.Wallet, error)
	Save(ctx context.Context, model *wallet.Wallet) error
}
