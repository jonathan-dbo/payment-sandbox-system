// Package invoice contains invoice application contracts and use cases.
package invoice

import (
	"context"
	"time"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
)

type ListFilter struct {
	MerchantID string
	Status     string
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	PageSize   int
}

type InvoiceRepository interface {
	Create(ctx context.Context, inv *invoice.Invoice) error
	FindByID(ctx context.Context, id string) (*invoice.Invoice, error)
	FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*invoice.Invoice, error)
	FindByPaymentToken(ctx context.Context, paymentToken string) (*invoice.Invoice, error)
	Save(ctx context.Context, inv *invoice.Invoice) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter ListFilter) ([]*invoice.Invoice, error)
}
