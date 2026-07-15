// Package paymentintent defines payment intent state transitions.
package paymentintent

import (
	"time"

	domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"
)

const (
	StatusPending = "PENDING"
	StatusSuccess = "SUCCESS"
	StatusFailed  = "FAILED"
)

type PaymentIntent struct {
	ID        string
	InvoiceID string
	Method    string
	Status    string
	DueAt     time.Time
	CreatedAt time.Time
}

func (p *PaymentIntent) MarkSuccess() error {
	switch p.Status {
	case StatusPending:
		p.Status = StatusSuccess
		return nil
	case StatusSuccess:
		return nil
	default:
		return invalidTransition(p.Status, StatusSuccess)
	}
}

func (p *PaymentIntent) MarkFailed() error {
	switch p.Status {
	case StatusPending:
		p.Status = StatusFailed
		return nil
	case StatusFailed:
		return nil
	default:
		return invalidTransition(p.Status, StatusFailed)
	}
}

func invalidTransition(from, to string) error {
	return domainErrors.InvalidTransitionError{
		Entity: "payment_intent",
		From:   from,
		To:     to,
	}
}
