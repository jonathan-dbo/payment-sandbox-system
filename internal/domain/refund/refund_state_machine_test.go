package refund_test

import (
	"testing"

	domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachineRefundValidTransitions(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusRequested}
	require.NoError(t, r.Approve())
	require.NoError(t, r.MarkSuccess())
	assert.Equal(t, refund.StatusSuccess, r.Status)

	r2 := &refund.Refund{Status: refund.StatusRequested}
	require.NoError(t, r2.Reject())
	assert.Equal(t, refund.StatusRejected, r2.Status)
}

func TestStateMachineRefundInvalidTransitions(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusRequested}
	err := r.MarkSuccess()
	require.Error(t, err)
	var transitionErr domainErrors.InvalidTransitionError
	require.ErrorAs(t, err, &transitionErr)
	assert.Equal(t, "refund", transitionErr.Entity)
}

func TestStateMachineRefundRepeatedTransitionRegression(t *testing.T) {
	r := &refund.Refund{Status: refund.StatusApproved}
	require.NoError(t, r.MarkFailed())
	require.NoError(t, r.MarkFailed())
	assert.Equal(t, refund.StatusFailed, r.Status)
}
