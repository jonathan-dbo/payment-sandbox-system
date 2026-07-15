package transaction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransaction_ValidCreation(t *testing.T) {
	transaction, err := NewTransaction("payment_1", TypeAuthorization, StatusSuccess, 5000, time.Time{})

	require.NoError(t, err)
	assert.NotEmpty(t, transaction.ID())
	assert.Equal(t, "payment_1", transaction.PaymentID())
	assert.Equal(t, TypeAuthorization, transaction.Type())
	assert.Equal(t, StatusSuccess, transaction.Status())
	assert.Equal(t, int64(5000), transaction.Amount())
	assert.False(t, transaction.Timestamp().IsZero())
}

func TestNewTransaction_InvalidInputs(t *testing.T) {
	tests := []struct {
		name      string
		paymentID string
		type_     TransactionType
		status    TransactionStatus
		amount    int64
	}{
		{name: "missing payment id", paymentID: "", type_: TypeAuthorization, status: StatusSuccess, amount: 100},
		{name: "invalid type", paymentID: "payment_1", type_: TransactionType("INVALID"), status: StatusSuccess, amount: 100},
		{name: "invalid status", paymentID: "payment_1", type_: TypeAuthorization, status: TransactionStatus("INVALID"), amount: 100},
		{name: "invalid amount", paymentID: "payment_1", type_: TypeAuthorization, status: StatusSuccess, amount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transaction, err := NewTransaction(tt.paymentID, tt.type_, tt.status, tt.amount, time.Now().UTC())
			assert.Error(t, err)
			assert.Nil(t, transaction)
		})
	}
}

func TestTransactionRestore(t *testing.T) {
	now := time.Now().UTC()
	transaction, err := RestoreTransaction("txn_1", "payment_1", TypeCapture, StatusSuccess, 5000, now)

	require.NoError(t, err)
	assert.Equal(t, "txn_1", transaction.ID())
	assert.Equal(t, TypeCapture, transaction.Type())
	assert.Equal(t, StatusSuccess, transaction.Status())
}

func TestTransactionValueObjects(t *testing.T) {
	assert.True(t, TypeAuthorization.IsValid())
	assert.True(t, TypeCapture.IsValid())
	assert.True(t, TypeRefund.IsValid())
	assert.True(t, TypeReversal.IsValid())
	assert.False(t, TransactionType("INVALID").IsValid())

	assert.True(t, StatusPending.IsValid())
	assert.True(t, StatusSuccess.IsValid())
	assert.True(t, StatusFailed.IsValid())
	assert.False(t, TransactionStatus("INVALID").IsValid())
}

func TestAuditEvent(t *testing.T) {
	auditEvent, err := NewAuditEvent("merchant_1", "payment_authorized", "api", time.Time{})

	require.NoError(t, err)
	assert.Equal(t, "merchant_1", auditEvent.Who)
	assert.Equal(t, "payment_authorized", auditEvent.What)
	assert.Equal(t, "api", auditEvent.Where)
	assert.False(t, auditEvent.When.IsZero())
}

func TestTransactionImmutability(t *testing.T) {
	transaction, err := NewTransaction("payment_1", TypeRefund, StatusSuccess, 5000, time.Now().UTC())
	require.NoError(t, err)

	id1 := transaction.ID()
	type1 := transaction.Type()
	status1 := transaction.Status()

	id2 := transaction.ID()
	type2 := transaction.Type()
	status2 := transaction.Status()

	assert.Equal(t, id1, id2)
	assert.Equal(t, type1, type2)
	assert.Equal(t, status1, status2)
}
