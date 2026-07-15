package http_test

import (
	"testing"

	appDashboard "github.com/gonszalito/go-ddd-architecture/internal/application/dashboard"
	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	infraDB "github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	infraHTTP "github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http"
	"github.com/stretchr/testify/require"
)

func TestRouterDependenciesMissingShouldPanic(t *testing.T) {
	require.Panics(t, func() {
		_ = infraHTTP.NewRouter(infraHTTP.RouterDependencies{})
	})
	require.Panics(t, func() {
		_ = infraHTTP.NewRouterCodegen(infraHTTP.RouterDependencies{})
	})
}

func TestRouterDependenciesCompleteShouldNotPanic(t *testing.T) {
	deps := validRouterDeps()

	require.NotPanics(t, func() {
		_ = infraHTTP.NewRouter(deps)
	})
	require.NotPanics(t, func() {
		_ = infraHTTP.NewRouterCodegen(deps)
	})
}

func validRouterDeps() infraHTTP.RouterDependencies {
	userService := appUser.NewUserService(infraDB.NewInMemoryUserRepository(nil), "test-secret", 60)
	invoiceService := appInvoice.NewService(infraDB.NewInMemoryInvoiceRepository(nil))
	paymentService := appPayment.NewService(infraDB.NewInMemoryPaymentRepository(nil), invoiceService)
	refundService := appRefund.NewService(infraDB.NewInMemoryRefundRepository(nil), infraDB.NewInMemoryWalletRepository(nil), invoiceService)
	topupService := appTopUp.NewService(infraDB.NewInMemoryTopUpRepository(nil), infraDB.NewInMemoryWalletRepository(nil))
	dashboardService := appDashboard.NewService(
		infraDB.NewInMemoryInvoiceRepository(nil),
		infraDB.NewInMemoryPaymentRepository(nil),
		infraDB.NewInMemoryRefundRepository(nil),
	)

	return infraHTTP.RouterDependencies{
		UserService:    userService,
		InvoiceService: invoiceService,
		PaymentService: paymentService,
		RefundService:  refundService,
		TopUpService:   topupService,
		Dashboard:      dashboardService,
		JWTSecret:      "test-secret",
	}
}
