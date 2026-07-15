package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Payment is the aggregate root for payment transactions.
// It represents a single payment order with state transitions and business invariants.
type Payment struct {
	// id: unique identifier for the payment.
	id string

	// merchantID: the merchant who initiated the payment.
	merchantID string

	// amount: payment amount in the smallest currency unit (e.g., cents for USD).
	amount int64

	// currency: ISO 4217 currency code (e.g., "USD", "EUR").
	currency string

	// status: current state of the payment.
	status PaymentStatus

	// refundedAmount: total amount refunded (for partial refund tracking).
	refundedAmount int64

	// createdAt: timestamp when payment was created.
	createdAt time.Time

	// updatedAt: timestamp of last state change.
	updatedAt time.Time
}

// NewPayment creates a new Payment aggregate with domain invariant checks.
// Returns an error if any invariant is violated.
func NewPayment(
	merchantID string,
	amount int64,
	currency string,
) (*Payment, error) {
	// Invariant: merchant_id must not be empty
	if merchantID == "" {
		return nil, fmt.Errorf("invalid payment: merchant_id must not be empty")
	}

	// Invariant: amount must be positive
	if amount <= 0 {
		return nil, fmt.Errorf("invalid payment: amount must be greater than 0, got %d", amount)
	}

	// Invariant: currency must not be empty and should be valid ISO 4217 code
	if currency == "" {
		return nil, fmt.Errorf("invalid payment: currency must not be empty")
	}

	now := time.Now().UTC()
	return &Payment{
		id:             uuid.New().String(),
		merchantID:     merchantID,
		amount:         amount,
		currency:       currency,
		status:         StatusPending,
		refundedAmount: 0,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// RestorePayment restores a Payment from persistence (used by repository layer).
// It bypasses invariant checks assuming data integrity from persistence.
func RestorePayment(
	id string,
	merchantID string,
	amount int64,
	currency string,
	status PaymentStatus,
	refundedAmount int64,
	createdAt time.Time,
	updatedAt time.Time,
) (*Payment, error) {
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid payment: unknown status %q", status)
	}

	return &Payment{
		id:             id,
		merchantID:     merchantID,
		amount:         amount,
		currency:       currency,
		status:         status,
		refundedAmount: refundedAmount,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}, nil
}

// === Accessors (immutable value retrieval) ===

// ID returns the payment's unique identifier.
func (p *Payment) ID() string {
	return p.id
}

// MerchantID returns the merchant who initiated the payment.
func (p *Payment) MerchantID() string {
	return p.merchantID
}

// Amount returns the payment amount.
func (p *Payment) Amount() int64 {
	return p.amount
}

// Currency returns the payment currency.
func (p *Payment) Currency() string {
	return p.currency
}

// Status returns the current payment status.
func (p *Payment) Status() PaymentStatus {
	return p.status
}

// RefundedAmount returns the total amount refunded.
func (p *Payment) RefundedAmount() int64 {
	return p.refundedAmount
}

// CreatedAt returns the payment creation timestamp.
func (p *Payment) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the timestamp of the last state change.
func (p *Payment) UpdatedAt() time.Time {
	return p.updatedAt
}

// === Domain Methods (state transitions with invariant validation) ===

// Authorize transitions the payment from PENDING to AUTHORIZED.
// This is the first state transition after payment creation.
// Error: if payment is not in PENDING state.
func (p *Payment) Authorize(ctx context.Context) error {
	if p.status != StatusPending {
		return fmt.Errorf("invalid state transition: cannot authorize payment in %q state", p.status)
	}

	p.status = StatusAuthorized
	p.updatedAt = time.Now().UTC()
	return nil
}

// Capture transitions the payment from AUTHORIZED to CAPTURED.
// This captures previously authorized funds.
// Error: if payment is not in AUTHORIZED state.
func (p *Payment) Capture(ctx context.Context) error {
	if p.status != StatusAuthorized {
		return fmt.Errorf("invalid state transition: cannot capture payment in %q state (must be %q)", p.status, StatusAuthorized)
	}

	p.status = StatusCaptured
	p.updatedAt = time.Now().UTC()
	return nil
}

// Refund transitions the payment toward REFUNDED and reduces the refunded amount.
// Can be called multiple times for partial refunds.
// Invariant: refundAmount must not exceed remaining capturable amount (amount - refundedAmount).
// Error: if payment is not in CAPTURED or AUTHORIZED state, or if refund amount is invalid.
func (p *Payment) Refund(ctx context.Context, refundAmount int64) error {
	// Only CAPTURED or AUTHORIZED payments can be refunded
	if p.status != StatusCaptured && p.status != StatusAuthorized {
		return fmt.Errorf("invalid state transition: cannot refund payment in %q state", p.status)
	}

	// Invariant: refund amount must be positive
	if refundAmount <= 0 {
		return fmt.Errorf("invalid refund: amount must be greater than 0, got %d", refundAmount)
	}

	// Invariant: cannot refund more than the original amount minus already refunded
	remainingRefundable := p.amount - p.refundedAmount
	if refundAmount > remainingRefundable {
		return fmt.Errorf("invalid refund: cannot refund %d, only %d remaining (original: %d, already refunded: %d)",
			refundAmount, remainingRefundable, p.amount, p.refundedAmount)
	}

	p.refundedAmount += refundAmount

	// Transition to REFUNDED only if all funds have been refunded
	if p.refundedAmount >= p.amount {
		p.status = StatusRefunded
	}

	p.updatedAt = time.Now().UTC()
	return nil
}

// Fail transitions the payment to FAILED.
// Used when authorization or capture fails.
// Can only be called from PENDING or AUTHORIZED states.
func (p *Payment) Fail(ctx context.Context) error {
	if p.status != StatusPending && p.status != StatusAuthorized {
		return fmt.Errorf("invalid state transition: cannot fail payment in %q state", p.status)
	}

	p.status = StatusFailed
	p.updatedAt = time.Now().UTC()
	return nil
}

// === Queries ===

// IsAuthorized returns true if payment is in AUTHORIZED state.
func (p *Payment) IsAuthorized() bool {
	return p.status == StatusAuthorized
}

// IsCaptured returns true if payment is in CAPTURED state.
func (p *Payment) IsCaptured() bool {
	return p.status == StatusCaptured
}

// IsRefunded returns true if payment is in REFUNDED state.
func (p *Payment) IsRefunded() bool {
	return p.status == StatusRefunded
}

// IsFailed returns true if payment is in FAILED state.
func (p *Payment) IsFailed() bool {
	return p.status == StatusFailed
}

// CanCaptureMore returns true if there are funds available to capture (not yet refunded).
func (p *Payment) CanCaptureMore() bool {
	return p.refundedAmount < p.amount
}
