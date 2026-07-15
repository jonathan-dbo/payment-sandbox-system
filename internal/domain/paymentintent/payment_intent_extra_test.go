package paymentintent_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentIntentMarkFailed_IdempotentWhenAlreadyFailed(t *testing.T) {
	p := &paymentintent.PaymentIntent{Status: paymentintent.StatusFailed}
	err := p.MarkFailed()
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusFailed, p.Status)
}

func TestPaymentIntentMarkFailed_InvalidFromSuccess(t *testing.T) {
	p := &paymentintent.PaymentIntent{Status: paymentintent.StatusSuccess}
	err := p.MarkFailed()
	require.Error(t, err)
}
