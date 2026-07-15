package wire_test

import (
	"testing"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	"github.com/gonszalito/go-ddd-architecture/internal/config"
	domainUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/gonszalito/go-ddd-architecture/internal/wire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the pure wiring functions in wire_gen.go that only
// assemble already-constructed dependencies (no *sql.DB / live database
// connection required). The DB-dependent providers (ProvideDatabase,
// Provide*Repository, InitializeApp, InitializeUserService) are covered by
// wire_test.go's TestWireAppInitializationSmoke, which is gated behind the
// INTEGRATION_DB_DSN environment variable since they require a real
// PostgreSQL connection and cannot be meaningfully unit tested with fakes
// without defeating the purpose of a wiring smoke test.

func TestProvideUserService_WiresRepositoryAndConfig(t *testing.T) {
	repo := database.NewInMemoryUserRepository(nil)
	cfg := &config.Config{JWTSecret: "test-secret", JWTExpiryMinutes: 30}

	svc := wire.ProvideUserService(cfg, repo)
	require.NotNil(t, svc)

	// Sanity check the wired service is actually functional end-to-end.
	result, err := svc.Register(t.Context(), "Wired User", "wired@example.com", "secret123", domainUser.RoleMerchant)
	require.NoError(t, err)
	assert.Equal(t, "wired@example.com", result.Email)
}

func TestProvideInvoiceService_Wires(t *testing.T) {
	repo := database.NewInMemoryInvoiceRepository(nil)
	svc := wire.ProvideInvoiceService(repo)
	require.NotNil(t, svc)

	inv, err := svc.Create(t.Context(), appInvoice.CreateInvoiceInput{MerchantID: "m-1", Amount: 1000, Currency: "USD"})
	require.NoError(t, err)
	assert.Equal(t, "m-1", inv.MerchantID)
}

func TestProvidePaymentService_Wires(t *testing.T) {
	invoiceRepo := database.NewInMemoryInvoiceRepository(nil)
	invoiceSvc := wire.ProvideInvoiceService(invoiceRepo)
	paymentRepo := database.NewInMemoryPaymentRepository(nil)

	svc := wire.ProvidePaymentService(paymentRepo, invoiceSvc)
	require.NotNil(t, svc)
}

func TestProvideRefundService_Wires(t *testing.T) {
	invoiceRepo := database.NewInMemoryInvoiceRepository(nil)
	invoiceSvc := wire.ProvideInvoiceService(invoiceRepo)
	refundRepo := database.NewInMemoryRefundRepository(nil)
	walletRepo := database.NewInMemoryWalletRepository(nil)

	svc := wire.ProvideRefundService(refundRepo, walletRepo, invoiceSvc)
	require.NotNil(t, svc)
}

func TestProvideTopUpService_Wires(t *testing.T) {
	topupRepo := database.NewInMemoryTopUpRepository(nil)
	walletRepo := database.NewInMemoryWalletRepository(nil)

	svc := wire.ProvideTopUpService(topupRepo, walletRepo)
	require.NotNil(t, svc)
}

func TestProvideDashboardService_Wires(t *testing.T) {
	invoiceRepo := database.NewInMemoryInvoiceRepository(nil)
	paymentRepo := database.NewInMemoryPaymentRepository(nil)
	refundRepo := database.NewInMemoryRefundRepository(nil)

	svc := wire.ProvideDashboardService(invoiceRepo, paymentRepo, refundRepo)
	require.NotNil(t, svc)
}

func TestProvideApp_AssemblesAllServices(t *testing.T) {
	invoiceRepo := database.NewInMemoryInvoiceRepository(nil)
	invoiceSvc := wire.ProvideInvoiceService(invoiceRepo)
	userRepo := database.NewInMemoryUserRepository(nil)
	userSvc := wire.ProvideUserService(&config.Config{JWTSecret: "s", JWTExpiryMinutes: 30}, userRepo)
	paymentRepo := database.NewInMemoryPaymentRepository(nil)
	paymentSvc := wire.ProvidePaymentService(paymentRepo, invoiceSvc)
	refundRepo := database.NewInMemoryRefundRepository(nil)
	walletRepo := database.NewInMemoryWalletRepository(nil)
	refundSvc := wire.ProvideRefundService(refundRepo, walletRepo, invoiceSvc)
	topupRepo := database.NewInMemoryTopUpRepository(nil)
	topupSvc := wire.ProvideTopUpService(topupRepo, walletRepo)
	dashboardSvc := wire.ProvideDashboardService(invoiceRepo, paymentRepo, refundRepo)

	app := wire.ProvideApp(userSvc, invoiceSvc, paymentSvc, refundSvc, topupSvc, dashboardSvc)
	require.NotNil(t, app)
	assert.Same(t, userSvc, app.UserService)
	assert.Same(t, invoiceSvc, app.InvoiceService)
	assert.Same(t, paymentSvc, app.PaymentService)
	assert.Same(t, refundSvc, app.RefundService)
	assert.Same(t, topupSvc, app.TopUpService)
	assert.Same(t, dashboardSvc, app.Dashboard)
}

func TestProvideConfig_LoadsFromEnvironment(t *testing.T) {
	t.Setenv("JWT_SECRET", "wire-test-secret")
	cfg, err := wire.ProvideConfig()
	require.NoError(t, err)
	assert.Equal(t, "wire-test-secret", cfg.JWTSecret)
}
