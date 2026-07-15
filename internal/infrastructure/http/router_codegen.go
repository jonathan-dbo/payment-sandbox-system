// Package http wires HTTP routes and middleware.
package http

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	infraMiddleware "github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
	apigen "github.com/gonszalito/go-ddd-architecture/internal/interfaces"
	"github.com/gonszalito/go-ddd-architecture/internal/interfaces/handlers"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouterCodegen demonstrates route mounting via oapi-codegen wrappers.
// It is an alternative to NewRouter and is intentionally separate.
func NewRouterCodegen(deps RouterDependencies) *gin.Engine {
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

	auth := infraMiddleware.NewAuthMiddleware(deps.JWTSecret)
	requireAuth := auth.RequireAuth()
	requireMerchant := infraMiddleware.RequireRole(user.RoleMerchant)
	requireAdmin := infraMiddleware.RequireRole(user.RoleAdmin)
	requireInvoiceRole := infraMiddleware.RequireRole(user.RoleMerchant, user.RoleAdmin)

	// Generated mount gets one middleware chain, so apply conditional role checks by path.
	codegenMiddleware := func(c *gin.Context) {
		path := c.Request.URL.Path
		switch {
		case path == "/auth/refresh":
			requireAuth(c)
			if c.IsAborted() {
				return
			}
		case strings.HasPrefix(path, "/invoices"):
			requireAuth(c)
			if c.IsAborted() {
				return
			}
			requireInvoiceRole(c)
		case strings.HasPrefix(path, "/merchant/"):
			requireAuth(c)
			if c.IsAborted() {
				return
			}
			requireMerchant(c)
		case strings.HasPrefix(path, "/admin/"):
			requireAuth(c)
			if c.IsAborted() {
				return
			}
			requireAdmin(c)
		}
	}

	apigen.RegisterHandlersWithOptions(r, &codegenServerAdapter{
		user:      userHandler,
		payment:   paymentHandler,
		refund:    refundHandler,
		topup:     topupHandler,
		dashboard: dashboardHandler,
		role:      roleHandler,
	}, apigen.GinServerOptions{
		Middlewares: []apigen.MiddlewareFunc{codegenMiddleware},
	})

	if deps.EnableDocs {
		swaggerURL := deps.SwaggerURL
		if swaggerURL == "" {
			swaggerURL = "http://localhost:8080/api.yaml"
		}

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

		r.GET("/docs/*any", gin.WrapH(httpSwagger.Handler(
			httpSwagger.URL(swaggerURL),
		)))
	}

	return r
}

type codegenServerAdapter struct {
	user      *handlers.UserHandler
	payment   *handlers.PaymentHandler
	refund    *handlers.RefundHandler
	topup     *handlers.TopUpHandler
	dashboard *handlers.DashboardHandler
	role      *handlers.RoleHandler
}

func (a *codegenServerAdapter) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().UTC().Format(time.RFC3339)})
}

func (a *codegenServerAdapter) Login(c *gin.Context) { a.user.LoginGin(c) }

func (a *codegenServerAdapter) Refresh(c *gin.Context) { a.user.RefreshGin(c) }

func (a *codegenServerAdapter) Register(c *gin.Context) { a.user.RegisterGin(c) }

func (a *codegenServerAdapter) AdminDashboard(c *gin.Context) { a.role.AdminOnly(c) }

func (a *codegenServerAdapter) ListInvoices(c *gin.Context, _ apigen.ListInvoicesParams) {
	a.user.ListInvoicesGin(c)
}

func (a *codegenServerAdapter) CreateInvoice(c *gin.Context) { a.user.CreateInvoiceGin(c) }

func (a *codegenServerAdapter) DeleteInvoice(c *gin.Context, id string) {
	c.Params = append(c.Params, gin.Param{Key: "id", Value: id})
	a.user.DeleteInvoiceGin(c)
}

func (a *codegenServerAdapter) GetInvoice(c *gin.Context, id string) {
	c.Params = append(c.Params, gin.Param{Key: "id", Value: id})
	a.user.GetInvoiceGin(c)
}

func (a *codegenServerAdapter) UpdateInvoice(c *gin.Context, id string) {
	c.Params = append(c.Params, gin.Param{Key: "id", Value: id})
	a.user.UpdateInvoiceGin(c)
}

func (a *codegenServerAdapter) MerchantProfile(c *gin.Context) { a.role.MerchantOnly(c) }

func (a *codegenServerAdapter) ListRefundHistory(c *gin.Context, _ apigen.ListRefundHistoryParams) {
	a.refund.ListHistory(c)
}

func (a *codegenServerAdapter) RequestRefund(c *gin.Context) { a.refund.RequestRefund(c) }

func (a *codegenServerAdapter) ListTopUpHistory(c *gin.Context, _ apigen.ListTopUpHistoryParams) {
	a.topup.History(c)
}

func (a *codegenServerAdapter) RequestTopUp(c *gin.Context) { a.topup.RequestTopUp(c) }

func (a *codegenServerAdapter) SimulatePaymentIntent(c *gin.Context, intentId string) {
	c.Params = append(c.Params, gin.Param{Key: "intentId", Value: intentId})
	a.payment.SimulatePaymentIntent(c)
}

func (a *codegenServerAdapter) ApproveRefund(c *gin.Context, refundId string) {
	c.Params = append(c.Params, gin.Param{Key: "refundId", Value: refundId})
	a.refund.ApproveRefund(c)
}

func (a *codegenServerAdapter) RejectRefund(c *gin.Context, refundId string) {
	c.Params = append(c.Params, gin.Param{Key: "refundId", Value: refundId})
	a.refund.RejectRefund(c)
}

func (a *codegenServerAdapter) ProcessRefund(c *gin.Context, refundId string) {
	c.Params = append(c.Params, gin.Param{Key: "refundId", Value: refundId})
	a.refund.ProcessRefund(c)
}

func (a *codegenServerAdapter) UpdateTopUpStatus(c *gin.Context, topupId string) {
	c.Params = append(c.Params, gin.Param{Key: "topupId", Value: topupId})
	a.topup.AdminUpdate(c)
}

func (a *codegenServerAdapter) ResolvePaymentLink(c *gin.Context, token string) {
	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})
	a.payment.ResolvePaymentLink(c)
}

func (a *codegenServerAdapter) CreatePaymentIntent(c *gin.Context, token string) {
	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})
	a.payment.CreatePaymentIntent(c)
}

func (a *codegenServerAdapter) GetDashboardStats(c *gin.Context, _ apigen.GetDashboardStatsParams) {
	a.dashboard.Stats(c)
}
