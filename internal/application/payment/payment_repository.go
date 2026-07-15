package payment

import (
	"context"
	"errors"
	"time"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
)

var ErrNotFound = errors.New("payment intent not found")

type Repository interface {
	Create(ctx context.Context, intent *paymentintent.PaymentIntent) error
	FindByID(ctx context.Context, id string) (*paymentintent.PaymentIntent, error)
	Save(ctx context.Context, intent *paymentintent.PaymentIntent) error
	List(ctx context.Context, filter ListFilter) ([]*paymentintent.PaymentIntent, error)
}

type ListFilter struct {
	InvoiceID  string
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
}
