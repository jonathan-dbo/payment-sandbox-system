// Package refund defines refund request lifecycle transitions.
package refund

import "time"
import domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"

const (
	StatusRequested = "REQUESTED"
	StatusApproved  = "APPROVED"
	StatusRejected  = "REJECTED"
	StatusSuccess   = "SUCCESS"
	StatusFailed    = "FAILED"
)

type Refund struct {
	ID         string
	InvoiceID  string
	MerchantID string
	Amount     int64
	Status     string
	History    []string
	CreatedAt  time.Time
}

func (r *Refund) Approve() error {
	switch r.Status {
	case StatusRequested:
		r.Status = StatusApproved
		r.History = append(r.History, "APPROVED")
		return nil
	case StatusApproved:
		return nil
	default:
		return invalidTransition(r.Status, StatusApproved)
	}
}

func (r *Refund) Reject() error {
	switch r.Status {
	case StatusRequested:
		r.Status = StatusRejected
		r.History = append(r.History, "REJECTED")
		return nil
	case StatusRejected:
		return nil
	default:
		return invalidTransition(r.Status, StatusRejected)
	}
}

func (r *Refund) MarkSuccess() error {
	switch r.Status {
	case StatusApproved:
		r.Status = StatusSuccess
		r.History = append(r.History, "SUCCESS")
		return nil
	case StatusSuccess:
		return nil
	default:
		return invalidTransition(r.Status, StatusSuccess)
	}
}

func (r *Refund) MarkFailed() error {
	switch r.Status {
	case StatusApproved:
		r.Status = StatusFailed
		r.History = append(r.History, "FAILED")
		return nil
	case StatusFailed:
		return nil
	default:
		return invalidTransition(r.Status, StatusFailed)
	}
}

func invalidTransition(from, to string) error {
	return domainErrors.InvalidTransitionError{
		Entity: "refund",
		From:   from,
		To:     to,
	}
}
