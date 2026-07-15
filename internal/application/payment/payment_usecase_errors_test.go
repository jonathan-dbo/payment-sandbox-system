package payment

import (
	"context"
	"errors"
	"testing"
	"time"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/paymentintent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentSimulateAdminOutcome_InvoiceLookupErrorPropagates(t *testing.T) {
	svc, _, invoiceRepo := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	delete(invoiceRepo.byID, "inv-1") // FindByID will now fail with ErrNotFound

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.Error(t, err)
	assert.True(t, errors.Is(err, appInvoice.ErrNotFound))
}

func TestPaymentSimulateAdminOutcome_ExpiryPath_RollbackIntentSaveAlsoFails(t *testing.T) {
	svc, paymentRepo, invoiceRepo := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	svc.nowFn = func() time.Time { return intent.DueAt.Add(time.Second) }
	invoiceRepo.failOnSave = true
	// Fail every Save call on the payment repo *after* the first (the initial
	// intent save inside the expiry branch must succeed so we reach the
	// invoice-save-then-rollback path).
	saveCount := 0
	originalSave := paymentRepo.saveFn
	paymentRepo.saveFn = func(ctx context.Context, in *paymentintent.PaymentIntent) error {
		saveCount++
		if saveCount == 1 {
			if originalSave != nil {
				return originalSave(ctx, in)
			}
			cp := *in
			paymentRepo.byID[in.ID] = &cp
			return nil
		}
		return errRollbackSaveFails
	}

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rollback intent failed")
}

func TestPaymentSimulateAdminOutcome_InvoiceMarkPaidErrorPropagates(t *testing.T) {
	svc, _, invoiceRepo := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	// EXPIRED -> PAID is an invalid domain transition, forcing the
	// `inv.MarkPaid()` error branch inside the success-outcome case.
	inv := invoiceRepo.byID["inv-1"]
	inv.Status = invoice.StatusExpired

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.Error(t, err)
}

func TestPaymentSimulateAdminOutcome_MarkFailedOnTerminalIntentPropagatesError(t *testing.T) {
	svc, _, _ := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.NoError(t, err)

	// intent now SUCCESS (terminal); OutcomeFailed forces intent.MarkFailed() error branch.
	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeFailed)
	require.Error(t, err)
}

func TestPaymentSimulateAdminOutcome_PaymentFlow_RollbackIntentSaveAlsoFails(t *testing.T) {
	svc, paymentRepo, invoiceRepo := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	invoiceRepo.failOnSave = true
	saveCount := 0
	paymentRepo.saveFn = func(ctx context.Context, in *paymentintent.PaymentIntent) error {
		saveCount++
		if saveCount == 1 {
			cp := *in
			paymentRepo.byID[in.ID] = &cp
			return nil
		}
		return errRollbackSaveFails
	}

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rollback intent failed")
}

func TestPaymentSimulateAdminOutcome_PaymentFlow_RollbackInvoiceSaveAlsoFails(t *testing.T) {
	svc, paymentRepo, invoiceRepo := newPaymentServiceWithInvoice(t)
	intent, err := svc.CreateIntent(context.Background(), "pay-token-1", MethodWallet)
	require.NoError(t, err)

	saveCount := 0
	invoiceRepo.saveFn = func(ctx context.Context, inv *invoice.Invoice) error {
		saveCount++
		return errRollbackInvoiceSaveFails
	}
	_ = paymentRepo

	_, err = svc.SimulateAdminOutcome(context.Background(), intent.ID, OutcomeSuccess)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rollback invoice failed")
}

var errRollbackSaveFails = errors.New("rollback save also failed")
var errRollbackInvoiceSaveFails = errors.New("rollback invoice save also failed")
