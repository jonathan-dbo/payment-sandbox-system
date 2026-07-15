//go:build wireinject

package wire

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	appdashboard "github.com/gonszalito/go-ddd-architecture/internal/application/dashboard"
	appinvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	apppayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	apprefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	apptopup "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	appwallet "github.com/gonszalito/go-ddd-architecture/internal/application/wallet"
	appuser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	"github.com/gonszalito/go-ddd-architecture/internal/config"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"

	// Ensure postgres driver is linked in wire builds
	_ "github.com/lib/pq"

	"github.com/google/wire"
)

// ProvideConfig loads application configuration from environment.
func ProvideConfig() (*config.Config, error) {
	return config.Load()
}

type App struct {
	UserService    *appuser.UserService
	InvoiceService *appinvoice.Service
	PaymentService *apppayment.Service
	RefundService  *apprefund.Service
	TopUpService   *apptopup.Service
	Dashboard      *appdashboard.Service
}

func ProvideDatabase(cfg *config.Config) (*sql.DB, func(), error) {
	dsn := cfg.DatabaseURL
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, func() {}, err
	}

	// Validate connection quickly; do not block startup for long.
	if err := pingDB(db); err != nil {
		_ = db.Close()
		return nil, func() {}, err
	}

	cleanup := func() {
		if err := db.Close(); err != nil {
			log.Println("error closing DB:", err)
		}
	}
	return db, cleanup, nil
}

func ProvideUserRepository(db *sql.DB) (appuser.UserRepository, error) {
	return database.NewPostgresUserRepository(db), nil
}

func pingDB(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// ProvideUserService wires the application service.
func ProvideUserService(cfg *config.Config, repo appuser.UserRepository) *appuser.UserService {
	return appuser.NewUserService(repo, cfg.JWTSecret, cfg.JWTExpiryMinutes)
}

func ProvideInvoiceRepository(db *sql.DB) (appinvoice.InvoiceRepository, error) {
	return database.NewPostgresInvoiceRepository(db), nil
}

func ProvideInvoiceService(repo appinvoice.InvoiceRepository) *appinvoice.Service {
	return appinvoice.NewService(repo)
}

func ProvidePaymentRepository(db *sql.DB) (apppayment.Repository, error) {
	return database.NewPostgresPaymentRepository(db), nil
}

func ProvideRefundRepository(db *sql.DB) (apprefund.Repository, error) {
	return database.NewPostgresRefundRepository(db), nil
}

func ProvideTopUpRepository(db *sql.DB) (apptopup.Repository, error) {
	return database.NewPostgresTopUpRepository(db), nil
}

func ProvideWalletRepository(db *sql.DB) (appwallet.Repository, error) {
	return database.NewPostgresWalletRepository(db), nil
}

func ProvidePaymentService(repo apppayment.Repository, invoiceService *appinvoice.Service) *apppayment.Service {
	return apppayment.NewService(repo, invoiceService)
}

func ProvideRefundService(repo apprefund.Repository, walletRepo appwallet.Repository, invoiceService *appinvoice.Service) *apprefund.Service {
	return apprefund.NewService(repo, walletRepo, invoiceService)
}

func ProvideTopUpService(repo apptopup.Repository, walletRepo appwallet.Repository) *apptopup.Service {
	return apptopup.NewService(repo, walletRepo)
}

func ProvideDashboardService(invoiceRepo appinvoice.InvoiceRepository, paymentRepo apppayment.Repository, refundRepo apprefund.Repository) *appdashboard.Service {
	return appdashboard.NewService(invoiceRepo, paymentRepo, refundRepo)
}

// InitializeUserService is the Wire injector that constructs the UserService and cleanup.
func InitializeUserService() (*appuser.UserService, func(), error) {
	wire.Build(
		ProvideConfig,
		ProvideDatabase,
		ProvideUserRepository,
		ProvideUserService,
	)
	return nil, nil, nil
}

func ProvideApp(
	userService *appuser.UserService,
	invoiceService *appinvoice.Service,
	paymentService *apppayment.Service,
	refundService *apprefund.Service,
	topupService *apptopup.Service,
	dashboardService *appdashboard.Service,
) *App {
	return &App{
		UserService:    userService,
		InvoiceService: invoiceService,
		PaymentService: paymentService,
		RefundService:  refundService,
		TopUpService:   topupService,
		Dashboard:      dashboardService,
	}
}

func InitializeApp() (*App, func(), error) {
	wire.Build(
		ProvideConfig,
		ProvideDatabase,
		ProvideUserRepository,
		ProvideUserService,
		ProvideInvoiceRepository,
		ProvideInvoiceService,
		ProvidePaymentRepository,
		ProvideRefundRepository,
		ProvideTopUpRepository,
		ProvideWalletRepository,
		ProvidePaymentService,
		ProvideRefundService,
		ProvideTopUpService,
		ProvideDashboardService,
		ProvideApp,
	)
	return nil, nil, nil
}
