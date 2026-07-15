package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appRefund "github.com/gonszalito/go-ddd-architecture/internal/application/refund"
	appTopUp "github.com/gonszalito/go-ddd-architecture/internal/application/topup"
	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	domainInvoice "github.com/gonszalito/go-ddd-architecture/internal/domain/invoice"
	domainRefund "github.com/gonszalito/go-ddd-architecture/internal/domain/refund"
	domainTopUp "github.com/gonszalito/go-ddd-architecture/internal/domain/topup"
	domainUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	apigen "github.com/gonszalito/go-ddd-architecture/internal/interfaces"
	"github.com/gonszalito/go-ddd-architecture/internal/interfaces/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== fakes for forcing raw (non-AppError) failures ====================

// failingInvoiceRepo forces every InvoiceRepository call to fail with a plain
// (non-AppError) error so we can exercise the 500-branches of CreateInvoice,
// ListInvoices, CreateInvoiceGin, and ListInvoicesGin that require a
// repository-level failure (the in-memory repo never fails these calls).
type failingInvoiceRepo struct{}

func (f *failingInvoiceRepo) Create(ctx context.Context, inv *domainInvoice.Invoice) error {
	return errors.New("boom: create failed")
}
func (f *failingInvoiceRepo) FindByID(ctx context.Context, id string) (*domainInvoice.Invoice, error) {
	return nil, errors.New("boom: find failed")
}
func (f *failingInvoiceRepo) FindByInvoiceNumber(ctx context.Context, invoiceNumber string) (*domainInvoice.Invoice, error) {
	return nil, appInvoice.ErrNotFound
}
func (f *failingInvoiceRepo) FindByPaymentToken(ctx context.Context, paymentToken string) (*domainInvoice.Invoice, error) {
	return nil, appInvoice.ErrNotFound
}
func (f *failingInvoiceRepo) Save(ctx context.Context, inv *domainInvoice.Invoice) error {
	return errors.New("boom: save failed")
}
func (f *failingInvoiceRepo) Delete(ctx context.Context, id string) error {
	return errors.New("boom: delete failed")
}
func (f *failingInvoiceRepo) List(ctx context.Context, filter appInvoice.ListFilter) ([]*domainInvoice.Invoice, error) {
	return nil, errors.New("boom: list failed")
}

// failingUserRepo forces UserService.Register to bubble up a raw (non-AppError)
// error from FindByEmail (since the error string isn't a recognized "not found"
// sentinel), exercising the fallback branch in registerErrorResponse /
// writeRegisterError that isn't reachable through the in-memory repository.
type failingUserRepo struct{}

func (f *failingUserRepo) FindByID(id string) (*domainUser.User, error) {
	return nil, errors.New("boom: lookup failed")
}
func (f *failingUserRepo) FindByEmail(email string) (*domainUser.User, error) {
	return nil, errors.New("boom: lookup failed")
}
func (f *failingUserRepo) Create(u *domainUser.User) error             { return errors.New("boom: create failed") }
func (f *failingUserRepo) CreateWithMerchant(u *domainUser.User) error { return errors.New("boom: create failed") }
func (f *failingUserRepo) Save(u *domainUser.User) error               { return errors.New("boom: save failed") }

// failingRefundRepo forces RefundService.History to return a raw error so we
// can exercise ListHistory's error branch.
type failingRefundRepo struct{}

func (f *failingRefundRepo) Create(ctx context.Context, model *domainRefund.Refund) error {
	return errors.New("boom: create failed")
}
func (f *failingRefundRepo) FindByID(ctx context.Context, id string) (*domainRefund.Refund, error) {
	return nil, appRefund.ErrNotFound
}
func (f *failingRefundRepo) List(ctx context.Context, filter appRefund.ListFilter) ([]*domainRefund.Refund, error) {
	return nil, errors.New("boom: list failed")
}
func (f *failingRefundRepo) ListByInvoiceID(ctx context.Context, invoiceID string) ([]*domainRefund.Refund, error) {
	return nil, errors.New("boom: list failed")
}
func (f *failingRefundRepo) Save(ctx context.Context, model *domainRefund.Refund) error {
	return errors.New("boom: save failed")
}

// failingTopUpHistoryRepo forces TopUpService.History to return a raw error so
// we can exercise History's error branch.
type failingTopUpHistoryRepo struct{}

func (f *failingTopUpHistoryRepo) Create(ctx context.Context, model *domainTopUp.TopUp) error {
	return errors.New("boom: create failed")
}
func (f *failingTopUpHistoryRepo) FindByID(ctx context.Context, id string) (*domainTopUp.TopUp, error) {
	return nil, appTopUp.ErrNotFound
}
func (f *failingTopUpHistoryRepo) FindByRequestKey(ctx context.Context, merchantID, requestKey string) (*domainTopUp.TopUp, error) {
	return nil, appTopUp.ErrNotFound
}
func (f *failingTopUpHistoryRepo) ListByMerchantID(ctx context.Context, merchantID string) ([]*domainTopUp.TopUp, error) {
	return nil, errors.New("boom: list failed")
}
func (f *failingTopUpHistoryRepo) Save(ctx context.Context, model *domainTopUp.TopUp) error {
	return errors.New("boom: save failed")
}

// ==================== UserHandler.Register / Login (apigen request-object methods) ====================

func TestUserHandler_Register_NilBody(t *testing.T) {
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)),
	)

	resp, err := h.Register(context.Background(), apigen.RegisterRequestObject{Body: nil})
	require.NoError(t, err)
	badReq, ok := resp.(apigen.Register400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "invalid_request", badReq.Error)
}

func TestUserHandler_Register_RawErrorFallback(t *testing.T) {
	// A repo that fails FindByEmail with a non-sentinel error makes
	// UserService.Register return the raw error directly (not wrapped in
	// shared.AppError), exercising registerErrorResponse's final fallback.
	h := handlers.NewUserHandler(
		appUser.NewUserService(&failingUserRepo{}, "test-secret", 30),
		appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)),
	)

	resp, err := h.Register(context.Background(), apigen.RegisterRequestObject{
		Body: &apigen.RegisterJSONRequestBody{
			Name:     "Someone",
			Email:    "someone@example.com",
			Password: "password123",
			Role:     rolePtr("MERCHANT"),
		},
	})
	require.NoError(t, err)
	fail, ok := resp.(apigen.Register500JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "internal_server_error", fail.Error)
}

func TestUserHandler_Login_NilBody(t *testing.T) {
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)),
	)

	resp, err := h.Login(context.Background(), apigen.LoginRequestObject{Body: nil})
	require.NoError(t, err)
	badReq, ok := resp.(apigen.Login400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "invalid_request", badReq.Error)
}

func TestUserHandler_Login_Success(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	h := handlers.NewUserHandler(svc, appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)))

	_, err := h.Register(context.Background(), apigen.RegisterRequestObject{
		Body: &apigen.RegisterJSONRequestBody{
			Name:     "Login Ctx",
			Email:    "login-ctx@example.com",
			Password: "password123",
			Role:     rolePtr("MERCHANT"),
		},
	})
	require.NoError(t, err)

	resp, err := h.Login(context.Background(), apigen.LoginRequestObject{
		Body: &apigen.LoginJSONRequestBody{
			Email:    "login-ctx@example.com",
			Password: "password123",
		},
	})
	require.NoError(t, err)
	success, ok := resp.(apigen.Login200JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "login-ctx@example.com", success.Email)
	assert.NotEmpty(t, success.Token)
}

// ==================== UserHandler.CreateInvoice / ListInvoices (apigen request-object methods) ====================

func TestUserHandler_CreateInvoice_NilBody(t *testing.T) {
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)),
	)

	resp, err := h.CreateInvoice(context.Background(), apigen.CreateInvoiceRequestObject{Body: nil})
	require.NoError(t, err)
	badReq, ok := resp.(apigen.CreateInvoice400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "invalid_request", badReq.Error)
}

func TestUserHandler_CreateInvoice_ValidationError(t *testing.T) {
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)),
	)

	resp, err := h.CreateInvoice(context.Background(), apigen.CreateInvoiceRequestObject{
		Body: &apigen.CreateInvoiceJSONRequestBody{MerchantId: "", Amount: 0},
	})
	require.NoError(t, err)
	badReq, ok := resp.(apigen.CreateInvoice400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "invalid_request", badReq.Error)
}

func TestUserHandler_CreateInvoice_ServiceError(t *testing.T) {
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(&failingInvoiceRepo{}),
	)

	resp, err := h.CreateInvoice(context.Background(), apigen.CreateInvoiceRequestObject{
		Body: &apigen.CreateInvoiceJSONRequestBody{MerchantId: "m1", Amount: 1000},
	})
	require.NoError(t, err)
	fail, ok := resp.(apigen.CreateInvoice500JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "internal_server_error", fail.Error)
}

func TestUserHandler_ListInvoices_ServiceError(t *testing.T) {
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(&failingInvoiceRepo{}),
	)

	resp, err := h.ListInvoices(context.Background(), apigen.ListInvoicesRequestObject{
		Params: apigen.ListInvoicesParams{},
	})
	require.NoError(t, err)
	fail, ok := resp.(apigen.ListInvoices500JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "internal_server_error", fail.Error)
}

// ==================== roleValue ====================

func TestUserHandler_Register_NilRoleDefaultsViaRoleValue(t *testing.T) {
	// req.Body.Role == nil exercises roleValue's nil branch; the user service
	// defaults an empty role string to MERCHANT.
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)),
	)

	resp, err := h.Register(context.Background(), apigen.RegisterRequestObject{
		Body: &apigen.RegisterJSONRequestBody{
			Name:     "No Role",
			Email:    "no-role@example.com",
			Password: "password123",
			Role:     nil,
		},
	})
	require.NoError(t, err)
	success, ok := resp.(apigen.Register201JSONResponse)
	require.True(t, ok)
	assert.Equal(t, domainUser.RoleMerchant, success.Role)
}

// ==================== UserHandler Gin methods: raw-error / edge branches ====================

func TestUserHandler_RegisterGin_RawErrorFallback(t *testing.T) {
	router := setupGinRouter()
	h := handlers.NewUserHandler(
		appUser.NewUserService(&failingUserRepo{}, "test-secret", 30),
		appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)),
	)
	router.POST("/auth/register", h.RegisterGin)

	rec := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]any{"name": "A", "email": "a@example.com", "password": "secret123", "role": "MERCHANT"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "internal_server_error", payload["error"])
}

func TestUserHandler_CreateInvoiceGin_ServiceError(t *testing.T) {
	router := setupGinRouter()
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(&failingInvoiceRepo{}),
	)
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"amount":500}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUserHandler_CreateInvoiceGin_MerchantExplicitOwnID(t *testing.T) {
	// Covers the branch where a MERCHANT explicitly supplies merchantId equal
	// to their own userID (merchantID != "" and merchantID == userID), which
	// skips both the "default to own ID" and "forbidden" sub-branches.
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("u-1", "MERCHANT"), h.CreateInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"merchantId":"u-1","amount":500}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestUserHandler_CreateInvoiceGin_AdminCanUseArbitraryMerchant(t *testing.T) {
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("admin-1", "ADMIN"), h.CreateInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"merchantId":"m-other","amount":500}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestUserHandler_ListInvoicesGin_ServiceError(t *testing.T) {
	router := setupGinRouter()
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(&failingInvoiceRepo{}),
	)
	router.GET("/invoices", withIdentity("u-1", "MERCHANT"), h.ListInvoicesGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestUserHandler_ListInvoicesGin_AdminSeesAllMerchants(t *testing.T) {
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.POST("/invoices", withIdentity("m-1", "MERCHANT"), h.CreateInvoiceGin)
	router.GET("/invoices", withIdentity("admin-1", "ADMIN"), h.ListInvoicesGin)

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/invoices", bytes.NewBufferString(`{"merchantId":"m-1","amount":500}`))
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices?merchantId=m-1", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// ==================== authIdentity edge branches ====================

func TestAuthIdentity_MissingRoleInContext(t *testing.T) {
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.GET("/invoices/:id", func(c *gin.Context) {
		c.Set("user_id", "u-1")
		// role intentionally not set
		c.Next()
	}, h.GetInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices/x", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthIdentity_InvalidUserIDType(t *testing.T) {
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.GET("/invoices/:id", func(c *gin.Context) {
		c.Set("user_id", 12345) // wrong type
		c.Set("user_role", "MERCHANT")
		c.Next()
	}, h.GetInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices/x", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthIdentity_InvalidRoleType(t *testing.T) {
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.GET("/invoices/:id", func(c *gin.Context) {
		c.Set("user_id", "u-1")
		c.Set("user_role", 999) // wrong type
		c.Next()
	}, h.GetInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices/x", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthIdentity_BlankUserID(t *testing.T) {
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.GET("/invoices/:id", func(c *gin.Context) {
		c.Set("user_id", "   ")
		c.Set("user_role", "MERCHANT")
		c.Next()
	}, h.GetInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices/x", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthIdentity_BlankRole(t *testing.T) {
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.GET("/invoices/:id", func(c *gin.Context) {
		c.Set("user_id", "u-1")
		c.Set("user_role", "   ")
		c.Next()
	}, h.GetInvoiceGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices/x", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== RefreshGin invalid auth-context type branch ====================

func TestUserHandler_RefreshGin_InvalidUserIDType(t *testing.T) {
	router := setupGinRouter()
	h := newUserHandlerFixture()
	router.POST("/auth/refresh", func(c *gin.Context) {
		c.Set("user_id", 42)
		c.Next()
	}, h.RefreshGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// ==================== RefundHandler ====================

func TestRefundHandler_RequestRefund_InvalidBody(t *testing.T) {
	router := setupGinRouter()
	h := newRefundHandlerFixture()
	router.POST("/merchant/refunds", h.RequestRefund)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/merchant/refunds", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRefundHandler_ListHistory_ServiceErrorBranch(t *testing.T) {
	router := setupGinRouter()
	invoiceSvc := appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil))
	refundSvc := appRefund.NewService(&failingRefundRepo{}, database.NewInMemoryWalletRepository(nil), invoiceSvc)
	h := handlers.NewRefundHandler(refundSvc)
	router.GET("/merchant/refunds", h.ListHistory)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/merchant/refunds?merchantId=m1", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "refund_error", payload["error"])
}

func TestRefundHandler_ListHistory_PageSizeClampedAbove100(t *testing.T) {
	router := setupGinRouter()
	h := newRefundHandlerFixture()
	router.GET("/merchant/refunds", h.ListHistory)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/merchant/refunds?pageSize=500", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, float64(100), payload["size"])
}

// ==================== TopUpHandler ====================

func TestTopUpHandler_RequestTopUp_InvalidBody(t *testing.T) {
	router := setupGinRouter()
	h := newTopUpHandlerFixture()
	router.POST("/merchant/topups", h.RequestTopUp)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/merchant/topups", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTopUpHandler_History_ServiceErrorBranch(t *testing.T) {
	router := setupGinRouter()
	topupSvc := appTopUp.NewService(&failingTopUpHistoryRepo{}, database.NewInMemoryWalletRepository(nil))
	h := handlers.NewTopUpHandler(topupSvc)
	router.GET("/merchant/topups", h.History)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/merchant/topups?merchantId=m1", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "topup_error", payload["error"])
}

func TestTopUpHandler_History_PageSizeClampedAbove100(t *testing.T) {
	router := setupGinRouter()
	h := newTopUpHandlerFixture()
	router.GET("/merchant/topups", h.History)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/merchant/topups?pageSize=999", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, float64(100), payload["size"])
}

// ==================== PaymentHandler ====================

func TestPaymentHandler_ResolvePaymentLink_BlankTokenParam(t *testing.T) {
	// Directly injects an empty "token" param (bypassing gin's normal :token
	// matching, which never produces an empty segment) to exercise the
	// strings.TrimSpace(token) == "" guard clause.
	router := setupGinRouter()
	h, _ := newPaymentHandlerFixture()
	router.GET("/pay-blank", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "token", Value: "   "}}
		h.ResolvePaymentLink(c)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/pay-blank", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, "invalid_request", payload["error"])
}

func TestPaymentHandler_CreatePaymentIntent_BlankTokenParam(t *testing.T) {
	router := setupGinRouter()
	h, _ := newPaymentHandlerFixture()
	router.POST("/pay-blank/intents", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "token", Value: ""}}
		h.CreatePaymentIntent(c)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/pay-blank/intents", bytes.NewBufferString(`{"method":"WALLET"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPaymentHandler_SimulatePaymentIntent_BlankIntentIDParam(t *testing.T) {
	router := setupGinRouter()
	h, _ := newPaymentHandlerFixture()
	router.POST("/simulate-blank", func(c *gin.Context) {
		c.Params = gin.Params{{Key: "intentId", Value: ""}}
		h.SimulatePaymentIntent(c)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/simulate-blank", bytes.NewBufferString(`{"outcome":"SUCCESS"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ==================== errors.go: respondError / parsePagination ====================

func TestRespondError_LogsInternalServerErrorBranch(t *testing.T) {
	router := setupGinRouter()
	h, _ := newPaymentHandlerFixture()
	// SimulatePaymentIntent's underlying service error path returns 400, so
	// to reach the >=500 branch in respondError we invoke it through a
	// service failure that the handler maps to 500: CreateInvoice's
	// repository failure path (already 500) is exercised elsewhere; here we
	// exercise it directly via UpdateInvoiceGin's invoice_error branch bumped
	// to 500 is not available, so use the ListInvoicesGin service error path
	// (500) which also flows through respondError.
	_ = h
	uh := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(&failingInvoiceRepo{}),
	)
	router.GET("/invoices", withIdentity("u-1", "MERCHANT"), uh.ListInvoicesGin)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/invoices", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestParsePagination_InvalidAndNegativeValuesFallBackToDefaults(t *testing.T) {
	router := setupGinRouter()
	h := newTopUpHandlerFixture()
	router.GET("/merchant/topups", h.History)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/merchant/topups?page=-5&pageSize=abc", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Equal(t, float64(1), payload["page"])
	assert.Equal(t, float64(10), payload["size"])
}

// ==================== shared test helpers ====================

func setupGinRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}
