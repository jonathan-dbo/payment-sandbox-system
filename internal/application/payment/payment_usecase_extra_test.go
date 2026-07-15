package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentCreateIntent_InvalidMethodRejected(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	_, err := svc.CreateIntent(context.Background(), "pay-token-1", "BOGUS_METHOD")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidMethod)
}

func TestPaymentCreateIntent_UnknownTokenRejected(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	_, err := svc.CreateIntent(context.Background(), "unknown-token", MethodWallet)
	require.Error(t, err)
}

func TestPaymentCreateIntent_RepoCreateErrorPropagates(t *testing.T) {
	svc, paymentRepo, _ := newPaymentServiceWithInvoice(t)
	paymentRepo.failOnCreate = true

	_, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errForcedCreate))
}

func TestPaymentSimulateAdminOutcome_IntentNotFound(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	_, err := svc.SimulateAdminOutcome(context.Background(), "missing-intent", OutcomeSuccess)
	require.Error(t, err)
}

func TestPaymentSimulateAdminOutcome_InvalidOutcomeRejected(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, "NOT_A_REAL_OUTCOME")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid simulation outcome")
}

func TestPaymentSimulateAdminOutcome_ExpiryPath_InvoiceSaveFailureRollsBackIntent(t *testing.T) {
	svc, paymentRepo, invoiceRepo := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	svc.nowFn = func() time.Time { return intent.DueAt.Add(time.Second) }
	invoiceRepo.failOnSave = true

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "atomic expiry flow failed at invoice save")

	persistedIntent, findErr := paymentRepo.FindByID(context.Background(), intent.ID)
	require.NoError(t, findErr)
	assert.Equal(t, paymentintent.StatusPending, persistedIntent.Status)
}

func TestPaymentSimulateAdminOutcome_ExpiryPath_IntentSaveFailure(t *testing.T) {
	svc, paymentRepo, _ := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	svc.nowFn = func() time.Time { return intent.DueAt.Add(time.Second) }
	paymentRepo.failOnSave = true

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "atomic expiry flow failed at intent save")
}

func TestPaymentSimulateAdminOutcome_ExpiryPath_Success(t *testing.T) {
	svc, _, invoiceRepo := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	svc.nowFn = func() time.Time { return intent.DueAt.Add(time.Second) }
	updated, err := svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.NoError(t, err)
	assert.Equal(t, paymentintent.StatusFailed, updated.Status)

	inv, err := invoiceRepo.FindByID(context.Background(), "inv-1")
	require.NoError(t, err)
	assert.Equal(t, "EXPIRED", inv.Status)
}

func TestPaymentSimulateAdminOutcome_MarkFailedInvalidTransition(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.NoError(t, err)

	// Intent is now SUCCESS (terminal); attempting to mark it FAILED again
	// should surface the domain's invalid-transition error.
	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeFailed)
	require.Error(t, err)
}
