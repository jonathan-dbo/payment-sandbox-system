// Package topup defines wallet top-up lifecycle.
package topup

import domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"

const (
	StatusPending = "PENDING"
	StatusSuccess = "SUCCESS"
	StatusFailed  = "FAILED"
)

type TopUp struct {
	ID         string
	MerchantID string
	Amount     int64
	Status     string
	RequestKey string
}

func (t *TopUp) MarkSuccess() error {
	switch t.Status {
	case StatusPending:
		t.Status = StatusSuccess
		return nil
	case StatusSuccess:
		return nil
	default:
		return invalidTransition(t.Status, StatusSuccess)
	}
}

func (t *TopUp) MarkFailed() error {
	switch t.Status {
	case StatusPending:
		t.Status = StatusFailed
		return nil
	case StatusFailed:
		return nil
	default:
		return invalidTransition(t.Status, StatusFailed)
	}
}

func invalidTransition(from, to string) error {
	return domainErrors.InvalidTransitionError{
		Entity: "top_up",
		From:   from,
		To:     to,
	}
}
