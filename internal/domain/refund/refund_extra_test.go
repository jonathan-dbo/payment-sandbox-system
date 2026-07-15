package refund_test

import (
	"testing"

	domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefundApprove_IdempotentWhenAlreadyApproved(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusApproved}
	err := r.Approve()
	require.NoError(t, err)
	assert.Equal(t, refund.StatusApproved, r.Status)
}

func TestRefundApprove_InvalidFromTerminalState(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusSuccess}
	err := r.Approve()
	require.Error(t, err)
	var transitionErr domainErrors.InvalidTransitionError
	require.ErrorAs(t, err, &transitionErr)
	assert.Equal(t, "refund", transitionErr.Entity)
}

func TestRefundReject_IdempotentWhenAlreadyRejected(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusRejected}
	err := r.Reject()
	require.NoError(t, err)
	assert.Equal(t, refund.StatusRejected, r.Status)
}

func TestRefundReject_InvalidFromTerminalState(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusFailed}
	err := r.Reject()
	require.Error(t, err)
	var transitionErr domainErrors.InvalidTransitionError
	require.ErrorAs(t, err, &transitionErr)
}

func TestRefundMarkSuccess_IdempotentWhenAlreadySuccess(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusSuccess}
	err := r.MarkSuccess()
	require.NoError(t, err)
}

func TestRefundMarkFailed_IdempotentWhenAlreadyFailed(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusFailed}
	err := r.MarkFailed()
	require.NoError(t, err)
}
