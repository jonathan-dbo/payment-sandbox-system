package invoice_test

import (
	"testing"

	domainErrors "github.com/gonszalito/go-ddd-architecture/internal/domain/errors"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachineInvoiceValidTransitions(t *testing.T) {
	inv := &invoice.Invoice{Status: invoice.StatusPending}
	require.NoError(t, inv.MarkPaid())
	assert.Equal(t, invoice.StatusPaid, inv.Status)

	inv2 := &invoice.Invoice{Status: invoice.StatusPending}
	require.NoError(t, inv2.MarkExpired())
	assert.Equal(t, invoice.StatusExpired, inv2.Status)
}

func TestStateMachineInvoiceInvalidTransitions(t *testing.T) {
	inv := &invoice.Invoice{Status: invoice.StatusPaid}
	err := inv.MarkExpired()
	require.Error(t, err)
	var transitionErr domainErrors.InvalidTransitionError
	require.ErrorAs(t, err, &transitionErr)
	assert.Equal(t, "invoice", transitionErr.Entity)
}

func TestStateMachineInvoiceRepeatedTransitionRegression(t *testing.T) {
	inv := &invoice.Invoice{Status: invoice.StatusPaid}
	require.NoError(t, inv.MarkPaid())
	require.NoError(t, inv.MarkPaid())
	assert.Equal(t, invoice.StatusPaid, inv.Status)
}
