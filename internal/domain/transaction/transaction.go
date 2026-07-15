package transaction

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransactionType represents the type of payment-related event recorded.
type TransactionType string

const (
	TypeAuthorization TransactionType = "AUTHORIZATION"
	TypeCapture       TransactionType = "CAPTURE"
	TypeRefund        TransactionType = "REFUND"
	TypeReversal      TransactionType = "REVERSAL"
)

// IsValid returns true when the transaction type is recognized.
func (tt TransactionType) IsValid() bool {
	switch tt {
	case TypeAuthorization, TypeCapture, TypeRefund, TypeReversal:
		return true
	default:
		return false
	}
}

// TransactionStatus represents the lifecycle state of a transaction record.
type TransactionStatus string

const (
	StatusPending TransactionStatus = "PENDING"
	StatusSuccess TransactionStatus = "SUCCESS"
	StatusFailed  TransactionStatus = "FAILED"
)

// IsValid returns true when the transaction status is recognized.
func (ts TransactionStatus) IsValid() bool {
	switch ts {
	case StatusPending, StatusSuccess, StatusFailed:
		return true
	default:
		return false
	}
}

// Transaction is an immutable audit record for payment state changes.
type Transaction struct {
	id        string
	paymentID string
	type_     TransactionType
	status    TransactionStatus
	amount    int64
	timestamp time.Time
}

// NewTransaction creates a new immutable transaction record.
func NewTransaction(paymentID string, transactionType TransactionType, status TransactionStatus, amount int64, timestamp time.Time) (*Transaction, error) {
	if paymentID == "" {
		return nil, fmt.Errorf("invalid transaction: payment_id must not be empty")
	}
	if !transactionType.IsValid() {
		return nil, fmt.Errorf("invalid transaction: unknown type %q", transactionType)
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid transaction: unknown status %q", status)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("invalid transaction: amount must be greater than 0")
	}
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	return &Transaction{
		id:        uuid.New().String(),
		paymentID: paymentID,
		type_:     transactionType,
		status:    status,
		amount:    amount,
		timestamp: timestamp.UTC(),
	}, nil
}

// RestoreTransaction restores a transaction from persistence.
func RestoreTransaction(id, paymentID string, transactionType TransactionType, status TransactionStatus, amount int64, timestamp time.Time) (*Transaction, error) {
	if id == "" {
		return nil, fmt.Errorf("invalid transaction: id must not be empty")
	}
	if paymentID == "" {
		return nil, fmt.Errorf("invalid transaction: payment_id must not be empty")
	}
	if !transactionType.IsValid() {
		return nil, fmt.Errorf("invalid transaction: unknown type %q", transactionType)
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("invalid transaction: unknown status %q", status)
	}
	if amount <= 0 {
		return nil, fmt.Errorf("invalid transaction: amount must be greater than 0")
	}
	if timestamp.IsZero() {
		return nil, fmt.Errorf("invalid transaction: timestamp must not be zero")
	}

	return &Transaction{
		id:        id,
		paymentID: paymentID,
		type_:     transactionType,
		status:    status,
		amount:    amount,
		timestamp: timestamp.UTC(),
	}, nil
}

// ID returns the transaction identifier.
func (t *Transaction) ID() string { return t.id }

// PaymentID returns the related payment identifier.
func (t *Transaction) PaymentID() string { return t.paymentID }

// Type returns the transaction type.
func (t *Transaction) Type() TransactionType { return t.type_ }

// Status returns the transaction status.
func (t *Transaction) Status() TransactionStatus { return t.status }

// Amount returns the transaction amount.
func (t *Transaction) Amount() int64 { return t.amount }

// Timestamp returns the event timestamp.
func (t *Transaction) Timestamp() time.Time { return t.timestamp }

// AuditEvent represents a domain audit entry.
type AuditEvent struct {
	Who   string
	What  string
	When  time.Time
	Where string
}

// NewAuditEvent creates a new audit event with a timestamp.
func NewAuditEvent(who, what, where string, when time.Time) (*AuditEvent, error) {
	if who == "" {
		return nil, fmt.Errorf("invalid audit event: who must not be empty")
	}
	if what == "" {
		return nil, fmt.Errorf("invalid audit event: what must not be empty")
	}
	if where == "" {
		return nil, fmt.Errorf("invalid audit event: where must not be empty")
	}
	if when.IsZero() {
		when = time.Now().UTC()
	}

	return &AuditEvent{Who: who, What: what, When: when.UTC(), Where: where}, nil
}
