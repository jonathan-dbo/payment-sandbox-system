package refund

import (
	"context"
	"errors"
	"time"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
)

var ErrNotFound = errors.New("refund not found")

type ListFilter struct {
	MerchantID string
	StartDate  *time.Time
	EndDate    *time.Time
}

type Repository interface {
	Create(ctx context.Context, model *refund.Refund) error
	FindByID(ctx context.Context, id string) (*refund.Refund, error)
	List(ctx context.Context, filter ListFilter) ([]*refund.Refund, error)
	ListByInvoiceID(ctx context.Context, invoiceID string) ([]*refund.Refund, error)
	Save(ctx context.Context, model *refund.Refund) error
}
