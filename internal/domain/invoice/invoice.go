// Package invoice defines invoice domain entities.
package invoice

import "time"
import domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"

const (
	StatusPending = "PENDING"
	StatusPaid    = "PAID"
	StatusExpired = "EXPIRED"
)

type Invoice struct {
	ID            string
	InvoiceNumber string
	MerchantID    string
	Amount        int64
	Currency      string
	Status        string
	PaymentToken  string
	DueDate       time.Time
	CreatedAt     time.Time
}

func (i *Invoice) MarkPaid() error {
	switch i.Status {
	case StatusPending:
		i.Status = StatusPaid
		return nil
	case StatusPaid:
		return nil
	default:
		return invalidTransition(i.Status, StatusPaid)
	}
}

func (i *Invoice) MarkExpired() error {
	switch i.Status {
	case StatusPending:
		i.Status = StatusExpired
		return nil
	case StatusExpired:
		return nil
	default:
		return invalidTransition(i.Status, StatusExpired)
	}
}

func invalidTransition(from, to string) error {
	return domainErrors.InvalidTransitionError{
		Entity: "invoice",
		From:   from,
		To:     to,
	}
}
