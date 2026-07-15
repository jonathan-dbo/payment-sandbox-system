// Package http wires HTTP routes and middleware.
package http

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	appDashboard "github.com/gonszalito/go-ddd-architecture/internal/application/dashboard"
	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appPayment "github.com/gonszalito/go-ddd-architecture/internal/application/payment"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	infraMiddleware "github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
	"github.com/gonszalito/go-ddd-architecture/internal/interfaces/handlers"
	httpSwagger "github.com/swaggo/http-swagger"
)

type RouterDependencies struct {
	UserService        *appUser.UserService
	InvoiceService     *appInvoice.Service
	PaymentService     *appPayment.Service
	RefundService      *appRefund.Service
	TopUpService       *appTopUp.Service
	Dashboard          *appDashboard.Service
	JWTSecret          string
	CORSAllowedOrigins []string
	EnableDocs         bool
	SwaggerURL         string
}

func NewRouter(deps RouterDependencies) *gin.Engine {
	mustDeps(deps)

	userHandler := handlers.NewUserHandler(deps.UserService, deps.InvoiceService)
	paymentHandler := handlers.NewPaymentHandler(deps.PaymentService)
	refundHandler := handlers.NewRefundHandler(deps.RefundService)
	topupHandler := handlers.NewTopUpHandler(deps.TopUpService)
	dashboardHandler := handlers.NewDashboardHandler(deps.Dashboard)
	roleHandler := handlers.NewRoleHandler()
	r := gin.Default()
	r.Use(infraMiddleware.CORSMiddleware(deps.CORSAllowedOrigins))
	r.Use(infraMiddleware.StructuredLoggingMiddleware())
	r.POST("/auth/login", userHandler.LoginGin)
	r.POST("/auth/register", userHandler.RegisterGin)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339)})
	})
	r.GET("/pay/:token", paymentHandler.ResolvePaymentLink)
	r.POST("/pay/:token/intents", paymentHandler.CreatePaymentIntent)

	auth := infraMiddleware.NewAuthMiddleware(deps.JWTSecret)
	r.POST("/auth/refresh", auth.RequireAuth(), userHandler.RefreshGin)

	invoiceRoutes := r.Group("/invoices")
	invoiceRoutes.Use(auth.RequireAuth(), infraMiddleware.RequireRole(user.RoleMerchant, user.RoleAdmin))
	invoiceRoutes.GET("", userHandler.ListInvoicesGin)
	invoiceRoutes.POST("", userHandler.CreateInvoiceGin)
	invoiceRoutes.GET("/:id", userHandler.GetInvoiceGin)
	invoiceRoutes.PUT("/:id", userHandler.UpdateInvoiceGin)
	invoiceRoutes.DELETE("/:id", userHandler.DeleteInvoiceGin)

	merchantRoutes := r.Group("/merchant")
	merchantRoutes.Use(auth.RequireAuth(), infraMiddleware.RequireRole(user.RoleMerchant))
	merchantRoutes.GET("/profile", roleHandler.MerchantOnly)
	merchantRoutes.POST("/refunds", refundHandler.RequestRefund)
	merchantRoutes.GET("/refunds", refundHandler.ListHistory)
	merchantRoutes.POST("/topups", topupHandler.RequestTopUp)
	merchantRoutes.GET("/topups", topupHandler.History)

	adminRoutes := r.Group("/admin")
	adminRoutes.Use(auth.RequireAuth(), infraMiddleware.RequireRole(user.RoleAdmin))
	adminRoutes.GET("/dashboard", roleHandler.AdminOnly)
	adminRoutes.GET("/dashboard/stats", dashboardHandler.Stats)
	adminRoutes.POST("/payment-intents/:intentId/simulate", paymentHandler.SimulatePaymentIntent)
	adminRoutes.POST("/refunds/:refundId/approve", refundHandler.ApproveRefund)
	adminRoutes.POST("/refunds/:refundId/reject", refundHandler.RejectRefund)
	adminRoutes.POST("/refunds/:refundId/process", refundHandler.ProcessRefund)
	adminRoutes.POST("/topups/:topupId/status", topupHandler.AdminUpdate)

	if deps.EnableDocs {
		swaggerURL := deps.SwaggerURL
		if swaggerURL == "" {
			swaggerURL = "http://localhost:8080/api.yaml"
		}

		// Serve OpenAPI YAML from common run locations.
		r.GET("/api.yaml", func(c *gin.Context) {
			candidates := []string{
				"internal/api/api.yaml",
				"../internal/api/api.yaml",
				"api/api.yaml",
			}
			for _, path := range candidates {
				if _, err := os.Stat(path); err == nil {
					c.File(path)
					return
				}
			}
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "OpenAPI definition file not found",
			})
		})

		// Serve Swagger docs.
		r.GET("/docs/*any", gin.WrapH(httpSwagger.Handler(
			httpSwagger.URL(swaggerURL),
		)))
	}

	return r
}

func mustDeps(deps RouterDependencies) {
	if deps.UserService == nil {
		panic("router dependency missing: UserService")
	}
	if deps.InvoiceService == nil {
		panic("router dependency missing: InvoiceService")
	}
	if deps.PaymentService == nil {
		panic("router dependency missing: PaymentService")
	}
	if deps.RefundService == nil {
		panic("router dependency missing: RefundService")
	}
	if deps.TopUpService == nil {
		panic("router dependency missing: TopUpService")
	}
	if deps.Dashboard == nil {
		panic("router dependency missing: Dashboard")
	}
}
