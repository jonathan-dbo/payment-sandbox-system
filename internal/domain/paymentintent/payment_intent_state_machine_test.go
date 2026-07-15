package paymentintent_test

import (
	"testing"

	domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachinePaymentIntentValidTransitions(t *testing.T) {
	p := &paymentintent.PaymentIntent{Status: paymentintent.StatusPending}
	require.NoError(t, p.MarkSuccess())
	assert.Equal(t, paymentintent.StatusSuccess, p.Status)

	p2 := &paymentintent.PaymentIntent{Status: paymentintent.StatusPending}
	require.NoError(t, p2.MarkFailed())
	assert.Equal(t, paymentintent.StatusFailed, p2.Status)
}

func TestStateMachinePaymentIntentInvalidTransitions(t *testing.T) {
	p := &paymentintent.PaymentIntent{Status: paymentintent.StatusFailed}
	err := p.MarkSuccess()
	require.Error(t, err)
	var transitionErr domainErrors.InvalidTransitionError
	require.ErrorAs(t, err, &transitionErr)
	assert.Equal(t, "payment_intent", transitionErr.Entity)
}

func TestStateMachinePaymentIntentRepeatedTransitionRegression(t *testing.T) {
	p := &paymentintent.PaymentIntent{Status: paymentintent.StatusSuccess}
	require.NoError(t, p.MarkSuccess())
	require.NoError(t, p.MarkSuccess())
	assert.Equal(t, paymentintent.StatusSuccess, p.Status)
}
