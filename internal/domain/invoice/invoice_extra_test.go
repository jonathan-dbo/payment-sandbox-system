package invoice_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvoiceMarkPaid_IdempotentWhenAlreadyPaid(t *testing.T) {
	inv := &invoice.Invoice{Status: invoice.StatusPaid}
	err := inv.MarkPaid()
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPaid, inv.Status)
}

func TestInvoiceMarkPaid_InvalidFromExpired(t *testing.T) {
	inv := &invoice.Invoice{Status: invoice.StatusExpired}
	err := inv.MarkPaid()
	require.Error(t, err)
}

func TestInvoiceMarkExpired_IdempotentWhenAlreadyExpired(t *testing.T) {
	inv := &invoice.Invoice{Status: invoice.StatusExpired}
	err := inv.MarkExpired()
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusExpired, inv.Status)
}

func TestInvoiceMarkExpired_InvalidFromPaid(t *testing.T) {
	inv := &invoice.Invoice{Status: invoice.StatusPaid}
	err := inv.MarkExpired()
	require.Error(t, err)
}
