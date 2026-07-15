package payment

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPayment_ValidCreation tests successful payment creation.
func TestNewPayment_ValidCreation(t *testing.T) {
	payment, err := NewPayment("merchant_123", 5000, "USD")

	require.NoError(t, err)
	assert.NotEmpty(t, payment.ID())
	assert.Equal(t, "merchant_123", payment.MerchantID())
	assert.Equal(t, int64(5000), payment.Amount())
	assert.Equal(t, "USD", payment.Currency())
	assert.Equal(t, StatusPending, payment.Status())
	assert.Equal(t, int64(0), payment.RefundedAmount())
	assert.NotNil(t, payment.CreatedAt())
	assert.NotNil(t, payment.UpdatedAt())
}

// TestNewPayment_InvalidMerchantID tests that empty merchant_id fails invariant check.
func TestNewPayment_InvalidMerchantID(t *testing.T) {
	payment, err := NewPayment("", 5000, "USD")

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "merchant_id must not be empty")
}

// TestNewPayment_InvalidAmount_Zero tests that zero amount fails invariant check.
func TestNewPayment_InvalidAmount_Zero(t *testing.T) {
	payment, err := NewPayment("merchant_123", 0, "USD")

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

// TestNewPayment_InvalidAmount_Negative tests that negative amount fails invariant check.
func TestNewPayment_InvalidAmount_Negative(t *testing.T) {
	payment, err := NewPayment("merchant_123", -100, "USD")

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

// TestNewPayment_InvalidCurrency tests that empty currency fails invariant check.
func TestNewPayment_InvalidCurrency(t *testing.T) {
	payment, err := NewPayment("merchant_123", 5000, "")

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "currency must not be empty")
}

// TestPaymentStatus_Transitions_ValidPath tests valid state transition sequence.
func TestPaymentStatus_Transitions_ValidPath(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")

	// PENDING -> AUTHORIZED
	err := payment.Authorize(ctx)
	require.NoError(t, err)
	assert.Equal(t, StatusAuthorized, payment.Status())

	// AUTHORIZED -> CAPTURED
	err = payment.Capture(ctx)
	require.NoError(t, err)
	assert.Equal(t, StatusCaptured, payment.Status())

	// CAPTURED -> REFUNDED (full refund)
	err = payment.Refund(ctx, 5000)
	require.NoError(t, err)
	assert.Equal(t, StatusRefunded, payment.Status())
}

// TestPaymentStatus_InvalidTransition_CaptureBeforeAuthorize tests that capturing without authorizing fails.
func TestPaymentStatus_InvalidTransition_CaptureBeforeAuthorize(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")

	// Try to capture without authorizing
	err := payment.Capture(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot capture payment in")
	assert.Contains(t, err.Error(), "PENDING")
	assert.Equal(t, StatusPending, payment.Status()) // status unchanged
}

// TestPaymentStatus_InvalidTransition_AuthorizeTwice tests that authorizing twice fails.
func TestPaymentStatus_InvalidTransition_AuthorizeTwice(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")

	// First authorization should succeed
	err := payment.Authorize(ctx)
	require.NoError(t, err)

	// Second authorization should fail
	err = payment.Authorize(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot authorize payment in")
}

// TestPaymentStatus_InvalidTransition_RefundFromPending tests that refunding without capturing fails.
func TestPaymentStatus_InvalidTransition_RefundFromPending(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")

	// Try to refund from PENDING state
	err := payment.Refund(ctx, 1000)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot refund payment in")
	assert.Contains(t, err.Error(), "PENDING")
}

// TestPaymentStatus_PartialRefund tests partial refund scenarios.
func TestPaymentStatus_PartialRefund(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	payment.Authorize(ctx)
	payment.Capture(ctx)

	// First partial refund: 2000
	err := payment.Refund(ctx, 2000)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), payment.RefundedAmount())
	assert.Equal(t, StatusCaptured, payment.Status()) // Still CAPTURED (not fully refunded)

	// Second partial refund: 2000
	err = payment.Refund(ctx, 2000)
	require.NoError(t, err)
	assert.Equal(t, int64(4000), payment.RefundedAmount())
	assert.Equal(t, StatusCaptured, payment.Status()) // Still CAPTURED

	// Final refund: 1000 (complete)
	err = payment.Refund(ctx, 1000)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), payment.RefundedAmount())
	assert.Equal(t, StatusRefunded, payment.Status()) // Now REFUNDED
}

// TestPaymentStatus_InvalidRefund_ExceedsAmount tests that refunding more than the payment amount fails.
func TestPaymentStatus_InvalidRefund_ExceedsAmount(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	payment.Authorize(ctx)
	payment.Capture(ctx)

	// Try to refund more than payment amount
	err := payment.Refund(ctx, 6000)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot refund")
	assert.Equal(t, int64(0), payment.RefundedAmount()) // No refund applied
}

// TestPaymentStatus_InvalidRefund_ExceedsRemaining tests that second refund exceeding remaining fails.
func TestPaymentStatus_InvalidRefund_ExceedsRemaining(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	payment.Authorize(ctx)
	payment.Capture(ctx)

	// First refund: 3000 (valid)
	err := payment.Refund(ctx, 3000)
	require.NoError(t, err)

	// Try to refund more than remaining: 3000 (only 2000 remaining)
	err = payment.Refund(ctx, 3000)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot refund")
	assert.Equal(t, int64(3000), payment.RefundedAmount()) // First refund only
}

// TestPaymentStatus_InvalidRefund_NegativeAmount tests that negative refund amount fails.
func TestPaymentStatus_InvalidRefund_NegativeAmount(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	payment.Authorize(ctx)
	payment.Capture(ctx)

	// Try to refund negative amount
	err := payment.Refund(ctx, -1000)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
	assert.Equal(t, int64(0), payment.RefundedAmount())
}

// TestPaymentStatus_InvalidRefund_ZeroAmount tests that zero refund amount fails.
func TestPaymentStatus_InvalidRefund_ZeroAmount(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	payment.Authorize(ctx)
	payment.Capture(ctx)

	// Try to refund zero amount
	err := payment.Refund(ctx, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

// TestPaymentStatus_Fail tests failing payment from PENDING and AUTHORIZED states.
func TestPaymentStatus_Fail_FromPending(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")

	err := payment.Fail(ctx)

	require.NoError(t, err)
	assert.Equal(t, StatusFailed, payment.Status())
}

// TestPaymentStatus_Fail_FromAuthorized tests failing payment from AUTHORIZED state.
func TestPaymentStatus_Fail_FromAuthorized(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	payment.Authorize(ctx)

	err := payment.Fail(ctx)

	require.NoError(t, err)
	assert.Equal(t, StatusFailed, payment.Status())
}

// TestPaymentStatus_InvalidFail_FromCaptured tests that failing from CAPTURED state fails.
func TestPaymentStatus_InvalidFail_FromCaptured(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	payment.Authorize(ctx)
	payment.Capture(ctx)

	err := payment.Fail(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot fail payment in")
	assert.Equal(t, StatusCaptured, payment.Status()) // status unchanged
}

// TestPaymentStatus_TerminalStates tests that terminal states prevent further transitions.
func TestPaymentStatus_TerminalStates(t *testing.T) {
	ctx := context.Background()

	// Test REFUNDED is terminal
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	payment.Authorize(ctx)
	payment.Capture(ctx)
	payment.Refund(ctx, 5000)

	err := payment.Authorize(ctx) // Try to transition from REFUNDED
	assert.Error(t, err)

	// Test FAILED is terminal
	payment2, _ := NewPayment("merchant_123", 5000, "USD")
	payment2.Fail(ctx)

	err = payment2.Authorize(ctx) // Try to transition from FAILED
	assert.Error(t, err)
}

// TestPaymentAccessors tests accessor methods return correct values (immutability).
func TestPaymentAccessors(t *testing.T) {
	payment, _ := NewPayment("merchant_123", 5000, "USD")

	assert.NotEmpty(t, payment.ID())
	assert.Equal(t, "merchant_123", payment.MerchantID())
	assert.Equal(t, int64(5000), payment.Amount())
	assert.Equal(t, "USD", payment.Currency())
	assert.NotNil(t, payment.CreatedAt())
	assert.NotNil(t, payment.UpdatedAt())
}

// TestPaymentQueries tests helper query methods.
func TestPaymentQueries(t *testing.T) {
	ctx := context.Background()
	payment, _ := NewPayment("merchant_123", 5000, "USD")

	// Initially PENDING
	assert.False(t, payment.IsAuthorized())
	assert.False(t, payment.IsCaptured())
	assert.False(t, payment.IsRefunded())
	assert.False(t, payment.IsFailed())
	assert.True(t, payment.CanCaptureMore())

	// After authorization
	payment.Authorize(ctx)
	assert.True(t, payment.IsAuthorized())
	assert.False(t, payment.IsCaptured())

	// After capture
	payment.Capture(ctx)
	assert.False(t, payment.IsAuthorized())
	assert.True(t, payment.IsCaptured())

	// After partial refund
	payment.Refund(ctx, 2000)
	assert.False(t, payment.IsRefunded())
	assert.Equal(t, int64(2000), payment.RefundedAmount())
	assert.True(t, payment.CanCaptureMore()) // Still has refundable amount

	// After full refund
	payment.Refund(ctx, 3000)
	assert.True(t, payment.IsRefunded())
	assert.False(t, payment.CanCaptureMore())
}

// TestRestorePayment tests payment restoration from persistence.
func TestRestorePayment(t *testing.T) {
	now, _ := NewPayment("merchant_123", 5000, "USD")
	createdAt := now.CreatedAt()
	updatedAt := now.UpdatedAt()

	restored, err := RestorePayment("payment_456", "merchant_123", 5000, "USD", StatusCaptured, 1000, createdAt, updatedAt)

	require.NoError(t, err)
	assert.Equal(t, "payment_456", restored.ID())
	assert.Equal(t, "merchant_123", restored.MerchantID())
	assert.Equal(t, int64(5000), restored.Amount())
	assert.Equal(t, "USD", restored.Currency())
	assert.Equal(t, StatusCaptured, restored.Status())
	assert.Equal(t, int64(1000), restored.RefundedAmount())
}

// TestRestorePayment_InvalidStatus tests that restoring with invalid status fails.
func TestRestorePayment_InvalidStatus(t *testing.T) {
	payment, _ := NewPayment("m", 1, "U")
	restored, err := RestorePayment("payment_456", "merchant_123", 5000, "USD", PaymentStatus("INVALID"), 0, payment.CreatedAt(), time.Now())

	assert.Error(t, err)
	assert.Nil(t, restored)
	assert.Contains(t, err.Error(), "unknown status")
}

// TestPaymentStatusValueObject tests PaymentStatus value object.
func TestPaymentStatusValueObject(t *testing.T) {
	tests := []struct {
		status PaymentStatus
		valid  bool
	}{
		{StatusPending, true},
		{StatusAuthorized, true},
		{StatusCaptured, true},
		{StatusRefunded, true},
		{StatusFailed, true},
		{PaymentStatus("INVALID"), false},
		{PaymentStatus(""), false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.valid, tt.status.IsValid(), "status %q should be valid=%v", tt.status, tt.valid)
	}
}

// TestPaymentStatusTransitions tests status transition matrix.
func TestPaymentStatusTransitions(t *testing.T) {
	tests := []struct {
		from          PaymentStatus
		to            PaymentStatus
		canTransition bool
	}{
		{StatusPending, StatusAuthorized, true},
		{StatusPending, StatusFailed, true},
		{StatusPending, StatusCaptured, false},
		{StatusAuthorized, StatusCaptured, true},
		{StatusAuthorized, StatusRefunded, true},
		{StatusAuthorized, StatusFailed, true},
		{StatusCaptured, StatusRefunded, true},
		{StatusCaptured, StatusAuthorized, false},
		{StatusRefunded, StatusCaptured, false},
		{StatusFailed, StatusAuthorized, false},
	}

	for _, tt := range tests {
		result := tt.from.CanTransitionTo(tt.to)
		assert.Equal(t, tt.canTransition, result,
			"transition from %q to %q should be %v", tt.from, tt.to, tt.canTransition)
	}
}

// TestPaymentStatusTerminal tests terminal state detection.
func TestPaymentStatusTerminal(t *testing.T) {
	tests := []struct {
		status   PaymentStatus
		terminal bool
	}{
		{StatusPending, false},
		{StatusAuthorized, false},
		{StatusCaptured, false},
		{StatusRefunded, true},
		{StatusFailed, true},
	}

	for _, tt := range tests {
		result := tt.status.IsTerminalState()
		assert.Equal(t, tt.terminal, result,
			"status %q terminal=%v", tt.status, tt.terminal)
	}
}

// TestPaymentImmutability tests that Payment aggregate fields are not mutated via accessors.
func TestPaymentImmutability(t *testing.T) {
	payment, _ := NewPayment("merchant_123", 5000, "USD")
	id1 := payment.ID()
	amount1 := payment.Amount()
	currency1 := payment.Currency()

	// Call accessors multiple times
	id2 := payment.ID()
	amount2 := payment.Amount()
	currency2 := payment.Currency()

	// Values should remain identical
	assert.Equal(t, id1, id2)
	assert.Equal(t, amount1, amount2)
	assert.Equal(t, currency1, currency2)
}
