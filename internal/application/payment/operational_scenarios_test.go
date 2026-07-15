package payment_test

import (
	"context"
	"testing"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/wallet"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationalInvoiceToPaymentSimulationHappyPath(t *testing.T) {
	ctx := context.Background()

	invoiceRepo := database.NewInMemoryInvoiceRepository(nil)
	invoiceService := appInvoice.NewService(invoiceRepo)
	paymentService := appPayment.NewService(database.NewInMemoryPaymentRepository(nil), invoiceService)

	inv, err := invoiceService.Create(ctx, appInvoice.CreateInvoiceInput{
		MerchantID: "m-1",
		Amount:     1500,
		Currency:   "USD",
	})
	require.NoError(t, err)

	intent, err := paymentService.CreateIntent(ctx, inv.PaymentToken, appPayment.MethodWallet)
	require.NoError(t, err)

	updatedIntent, err := paymentService.SimulateAdminOutcome(ctx, intent.ID, appPayment.OutcomeSuccess)
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", updatedIntent.Status)

	updatedInvoice, err := invoiceService.GetByID(ctx, inv.ID)
	require.NoError(t, err)
	assert.Equal(t, invoice.StatusPaid, updatedInvoice.Status)
}

func TestOperationalInvoiceToRefundApproveSuccessScenario(t *testing.T) {
	ctx := context.Background()

	invoiceService := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	walletRepo := database.NewInMemoryWalletRepository([]*wallet.Wallet{
		{ID: "w-1", MerchantID: "m-1", Balance: 2000},
	})
	refundService := appRefund.NewService(database.NewInMemoryRefundRepository(nil), walletRepo, invoiceService)

	inv, err := invoiceService.Create(ctx, appInvoice.CreateInvoiceInput{
		MerchantID: "m-1",
		Amount:     700,
		Currency:   "USD",
	})
	require.NoError(t, err)
	require.NoError(t, inv.MarkPaid())
	require.NoError(t, invoiceService.Save(ctx, inv))

	requested, err := refundService.RequestRefund(ctx, inv.ID, "m-1", 700)
	require.NoError(t, err)
	assert.Equal(t, "REQUESTED", requested.Status)

	approved, err := refundService.Approve(ctx, requested.ID)
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", approved.Status)

	processed, err := refundService.Process(ctx, requested.ID, true)
	require.NoError(t, err)
	assert.Equal(t, "SUCCESS", processed.Status)

	w, err := walletRepo.FindByMerchantID(ctx, "m-1")
	require.NoError(t, err)
	assert.Equal(t, int64(2700), w.Balance)
}
