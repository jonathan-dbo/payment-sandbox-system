package merchant_test

import (
	"testing"
	"time"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/merchant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerchant_CreatedAtAccessor(t *testing.T) {
	now := time.Now().UTC()
	m, err := merchant.RestoreMerchant("m-1", "Acme", "acme@example.com", merchant.StatusActive, "hash", now, now)
	require.NoError(t, err)
	assert.Equal(t, now, m.CreatedAt())
}

func TestMerchant_UpdatedAtAccessor(t *testing.T) {
	created := time.Now().UTC().Add(-time.Hour)
	updated := time.Now().UTC()
	m, err := merchant.RestoreMerchant("m-1", "Acme", "acme@example.com", merchant.StatusActive, "hash", created, updated)
	require.NoError(t, err)
	assert.Equal(t, updated, m.UpdatedAt())
}

func TestMerchant_RestoreMerchant_InvalidStatusRejected(t *testing.T) {
	now := time.Now().UTC()
	_, err := merchant.RestoreMerchant("m-1", "Acme", "acme@example.com", merchant.MerchantStatus("BOGUS"), "hash", now, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown status")
}

func TestMerchant_RestoreMerchant_ValidStatuses(t *testing.T) {
	now := time.Now().UTC()
	for _, status := range []merchant.MerchantStatus{merchant.StatusPending, merchant.StatusActive, merchant.StatusSuspended} {
		m, err := merchant.RestoreMerchant("m-1", "Acme", "acme@example.com", status, "hash", now, now)
		require.NoError(t, err)
		assert.Equal(t, status, m.Status())
	}
}

func TestMerchant_Suspend_ValidFromPending(t *testing.T) {
	m, err := merchant.NewMerchant("Acme", "acme@example.com", "hash")
	require.NoError(t, err)
	require.NoError(t, m.Suspend())
	assert.Equal(t, merchant.StatusSuspended, m.Status())
}

func TestMerchant_Suspend_ValidFromActive(t *testing.T) {
	m, err := merchant.NewMerchant("Acme", "acme@example.com", "hash")
	require.NoError(t, err)
	require.NoError(t, m.Activate())
	require.NoError(t, m.Suspend())
	assert.Equal(t, merchant.StatusSuspended, m.Status())
}

func TestMerchant_Suspend_InvalidFromInactive(t *testing.T) {
	now := time.Now().UTC()
	m, err := merchant.RestoreMerchant("m-1", "Acme", "acme@example.com", merchant.StatusInactive, "hash", now, now)
	require.NoError(t, err)
	err = m.Suspend()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot suspend merchant")
}

func TestMerchantStatus_CanTransitionTo_SuspendedToActive(t *testing.T) {
	assert.True(t, merchant.StatusSuspended.CanTransitionTo(merchant.StatusActive))
}

func TestMerchantStatus_CanTransitionTo_SameStateNotAutomaticallyAllowed(t *testing.T) {
	// The domain does not special-case "no-op" transitions: moving to the
	// same status must still appear in that status's explicit allow-list.
	assert.False(t, merchant.StatusPending.CanTransitionTo(merchant.StatusPending))
	assert.False(t, merchant.StatusActive.CanTransitionTo(merchant.StatusActive))
	assert.False(t, merchant.StatusSuspended.CanTransitionTo(merchant.StatusSuspended))
}

func TestMerchantStatus_CanTransitionTo_InvalidTransitionsRejected(t *testing.T) {
	assert.False(t, merchant.StatusSuspended.CanTransitionTo(merchant.StatusPending))
	assert.False(t, merchant.StatusInactive.CanTransitionTo(merchant.StatusActive))
}

func TestMerchantStatus_IsValid_InactiveIsRecognized(t *testing.T) {
	assert.True(t, merchant.StatusInactive.IsValid())
}
