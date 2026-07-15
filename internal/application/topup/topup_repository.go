package topup

import (
	"context"
	"errors"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
)

var (
	ErrNotFound     = errors.New("topup not found")
	ErrDuplicateKey = errors.New("duplicate topup request key")
)

type Repository interface {
	Create(ctx context.Context, model *topup.TopUp) error
	FindByID(ctx context.Context, id string) (*topup.TopUp, error)
	FindByRequestKey(ctx context.Context, merchantID, requestKey string) (*topup.TopUp, error)
	ListByMerchantID(ctx context.Context, merchantID string) ([]*topup.TopUp, error)
	Save(ctx context.Context, model *topup.TopUp) error
}
