package merchant

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMerchant_ValidCreation(t *testing.T) {
	merchant, err := NewMerchant("Acme Corp", "acme@example.com", "hash_123")

	require.NoError(t, err)
	assert.NotEmpty(t, merchant.ID())
	assert.Equal(t, "Acme Corp", merchant.Name())
	assert.Equal(t, "acme@example.com", merchant.Email())
	assert.Equal(t, StatusPending, merchant.Status())
	assert.Equal(t, "hash_123", merchant.GetAPIKeyHash())
}

func TestNewMerchant_InvalidInputs(t *testing.T) {
	tests := []struct {
		name         string
		merchantName string
		email        string
		apiKeyHash   string
	}{
		{name: "missing name", merchantName: "", email: "a@example.com", apiKeyHash: "hash"},
		{name: "missing email", merchantName: "Acme", email: "", apiKeyHash: "hash"},
		{name: "missing api key hash", merchantName: "Acme", email: "a@example.com", apiKeyHash: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merchant, err := NewMerchant(tt.merchantName, tt.email, tt.apiKeyHash)
			assert.Error(t, err)
			assert.Nil(t, merchant)
		})
	}
}

func TestMerchant_ActivateAndSuspend(t *testing.T) {
	merchant, err := NewMerchant("Acme Corp", "acme@example.com", "hash_123")
	require.NoError(t, err)

	err = merchant.Activate()
	require.NoError(t, err)
	assert.Equal(t, StatusActive, merchant.Status())

	err = merchant.Suspend()
	require.NoError(t, err)
	assert.Equal(t, StatusSuspended, merchant.Status())
}

func TestMerchant_InvalidTransition_FromInactive(t *testing.T) {
	now := merchantTimeHelper(t)
	merchant, err := RestoreMerchant("merchant_1", "Acme Corp", "acme@example.com", StatusInactive, "hash_123", now, now)
	require.NoError(t, err)

	err = merchant.Activate()
	assert.Error(t, err)
	assert.Equal(t, StatusInactive, merchant.Status())
}

func TestMerchantStatusValueObject(t *testing.T) {
	assert.True(t, StatusPending.IsValid())
	assert.True(t, StatusActive.IsValid())
	assert.True(t, StatusSuspended.IsValid())
	assert.True(t, StatusInactive.IsValid())
	assert.False(t, MerchantStatus("UNKNOWN").IsValid())
}

func TestMerchantStatusTransitions(t *testing.T) {
	tests := []struct {
		from MerchantStatus
		to   MerchantStatus
		ok   bool
	}{
		{StatusPending, StatusActive, true},
		{StatusPending, StatusSuspended, true},
		{StatusPending, StatusInactive, true},
		{StatusActive, StatusSuspended, true},
		{StatusActive, StatusInactive, true},
		{StatusSuspended, StatusActive, true},
		{StatusSuspended, StatusInactive, true},
		{StatusInactive, StatusActive, false},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.ok, tt.from.CanTransitionTo(tt.to))
	}
}

func TestMerchantRestore(t *testing.T) {
	now := merchantTimeHelper(t)
	merchant, err := RestoreMerchant("merchant_1", "Acme Corp", "acme@example.com", StatusActive, "hash_123", now, now)
	require.NoError(t, err)
	assert.Equal(t, "merchant_1", merchant.ID())
	assert.Equal(t, StatusActive, merchant.Status())
}

func merchantTimeHelper(t *testing.T) (now time.Time) {
	t.Helper()
	return time.Now().UTC()
}
