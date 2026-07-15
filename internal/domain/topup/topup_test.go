package topup_test

import (
	"testing"

	domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTopUpMarkSuccess_FromPending(t *testing.T) {
	tu := &topup.TopUp{ID: "t-1", Status: topup.StatusPending}
	err := tu.MarkSuccess()
	require.NoError(t, err)
	assert.Equal(t, topup.StatusSuccess, tu.Status)
}

func TestTopUpMarkSuccess_IdempotentFromSuccess(t *testing.T) {
	tu := &topup.TopUp{ID: "t-1", Status: topup.StatusSuccess}
	err := tu.MarkSuccess()
	require.NoError(t, err)
	assert.Equal(t, topup.StatusSuccess, tu.Status)
}

func TestTopUpMarkSuccess_InvalidFromFailed(t *testing.T) {
	tu := &topup.TopUp{ID: "t-1", Status: topup.StatusFailed}
	err := tu.MarkSuccess()
	require.Error(t, err)
	var transitionErr domainErrors.InvalidTransitionError
	require.ErrorAs(t, err, &transitionErr)
	assert.Equal(t, "top_up", transitionErr.Entity)
	assert.Equal(t, topup.StatusFailed, transitionErr.From)
	assert.Equal(t, topup.StatusSuccess, transitionErr.To)
	assert.Equal(t, topup.StatusFailed, tu.Status)
}

func TestTopUpMarkFailed_FromPending(t *testing.T) {
	tu := &topup.TopUp{ID: "t-1", Status: topup.StatusPending}
	err := tu.MarkFailed()
	require.NoError(t, err)
	assert.Equal(t, topup.StatusFailed, tu.Status)
}

func TestTopUpMarkFailed_IdempotentFromFailed(t *testing.T) {
	tu := &topup.TopUp{ID: "t-1", Status: topup.StatusFailed}
	err := tu.MarkFailed()
	require.NoError(t, err)
	assert.Equal(t, topup.StatusFailed, tu.Status)
}

func TestTopUpMarkFailed_InvalidFromSuccess(t *testing.T) {
	tu := &topup.TopUp{ID: "t-1", Status: topup.StatusSuccess}
	err := tu.MarkFailed()
	require.Error(t, err)
	var transitionErr domainErrors.InvalidTransitionError
	require.ErrorAs(t, err, &transitionErr)
	assert.Equal(t, topup.StatusSuccess, tu.Status)
}

func TestTopUpFieldAssignment(t *testing.T) {
	tu := &topup.TopUp{
		ID:         "t-1",
		MerchantID: "m-1",
		Amount:     500,
		Status:     topup.StatusPending,
		RequestKey: "req-1",
	}
	assert.Equal(t, "t-1", tu.ID)
	assert.Equal(t, "m-1", tu.MerchantID)
	assert.Equal(t, int64(500), tu.Amount)
	assert.Equal(t, "req-1", tu.RequestKey)
}
