package transaction_test

import (
	"testing"
	"time"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreTransaction_EmptyIDRejected(t *testing.T) {
	_, err := transaction.RestoreTransaction("", "pay-1", transaction.TypeCapture, transaction.StatusSuccess, 100, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id must not be empty")
}

func TestRestoreTransaction_EmptyPaymentIDRejected(t *testing.T) {
	_, err := transaction.RestoreTransaction("t-1", "", transaction.TypeCapture, transaction.StatusSuccess, 100, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "payment_id must not be empty")
}

func TestRestoreTransaction_InvalidTypeRejected(t *testing.T) {
	_, err := transaction.RestoreTransaction("t-1", "pay-1", transaction.TransactionType("BOGUS"), transaction.StatusSuccess, 100, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
}

func TestRestoreTransaction_InvalidStatusRejected(t *testing.T) {
	_, err := transaction.RestoreTransaction("t-1", "pay-1", transaction.TypeCapture, transaction.TransactionStatus("BOGUS"), 100, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown status")
}

func TestRestoreTransaction_NonPositiveAmountRejected(t *testing.T) {
	_, err := transaction.RestoreTransaction("t-1", "pay-1", transaction.TypeCapture, transaction.StatusSuccess, 0, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "amount must be greater than 0")
}

func TestRestoreTransaction_ZeroTimestampRejected(t *testing.T) {
	_, err := transaction.RestoreTransaction("t-1", "pay-1", transaction.TypeCapture, transaction.StatusSuccess, 100, time.Time{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp must not be zero")
}

func TestRestoreTransaction_ValidRestoration(t *testing.T) {
	now := time.Now().UTC()
	tx, err := transaction.RestoreTransaction("t-1", "pay-1", transaction.TypeCapture, transaction.StatusSuccess, 100, now)
	require.NoError(t, err)
	assert.Equal(t, "t-1", tx.ID())
	assert.Equal(t, "pay-1", tx.PaymentID())
	assert.Equal(t, transaction.TypeCapture, tx.Type())
	assert.Equal(t, transaction.StatusSuccess, tx.Status())
	assert.Equal(t, int64(100), tx.Amount())
	assert.WithinDuration(t, now, tx.Timestamp(), time.Second)
}

func TestNewAuditEvent_EmptyWhoRejected(t *testing.T) {
	_, err := transaction.NewAuditEvent("", "did something", "here", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "who must not be empty")
}

func TestNewAuditEvent_EmptyWhatRejected(t *testing.T) {
	_, err := transaction.NewAuditEvent("someone", "", "here", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "what must not be empty")
}

func TestNewAuditEvent_EmptyWhereRejected(t *testing.T) {
	_, err := transaction.NewAuditEvent("someone", "did something", "", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "where must not be empty")
}

func TestNewAuditEvent_DefaultsWhenZeroTimestamp(t *testing.T) {
	event, err := transaction.NewAuditEvent("someone", "did something", "here", time.Time{})
	require.NoError(t, err)
	assert.False(t, event.When.IsZero())
}

func TestNewAuditEvent_ValidCreation(t *testing.T) {
	now := time.Now().UTC()
	event, err := transaction.NewAuditEvent("someone", "did something", "here", now)
	require.NoError(t, err)
	assert.Equal(t, "someone", event.Who)
	assert.Equal(t, "did something", event.What)
	assert.Equal(t, "here", event.Where)
	assert.WithinDuration(t, now, event.When, time.Second)
}
