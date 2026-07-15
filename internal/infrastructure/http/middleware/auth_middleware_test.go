package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleMerchantAllowedAdminDenied(t *testing.T) {
	engine := setupRoleTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/merchant/profile", nil)
	req.Header.Set("Authorization", "Bearer "+mustToken(t, "test-secret", "u1", user.RoleMerchant))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/merchant/profile", nil)
	req.Header.Set("Authorization", "Bearer "+mustToken(t, "test-secret", "u1", user.RoleAdmin))
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRoleAdminAllowedMerchantDenied(t *testing.T) {
	engine := setupRoleTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+mustToken(t, "test-secret", "u1", user.RoleAdmin))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+mustToken(t, "test-secret", "u1", user.RoleMerchant))
	rec = httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRoleMissingToken(t *testing.T) {
	engine := setupRoleTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/merchant/profile", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeBody(t, rec)
	assert.Equal(t, "auth_error", resp["error"])
}

func TestRoleInvalidToken(t *testing.T) {
	engine := setupRoleTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/merchant/profile", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeBody(t, rec)
	assert.Equal(t, "auth_error", resp["error"])
}

func TestRolePublicEndpointAccessible(t *testing.T) {
	engine := setupRoleTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/public/health", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func setupRoleTestRouter(secret string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	auth := middleware.NewAuthMiddleware(secret)

	merchant := engine.Group("/merchant")
	merchant.Use(auth.RequireAuth(), middleware.RequireRole(user.RoleMerchant))
	merchant.GET("/profile", func(c *gin.Context) {
		role, _ := c.Get(middleware.ContextRoleKey)
		c.JSON(http.StatusOK, gin.H{"role": role})
	})

	admin := engine.Group("/admin")
	admin.Use(auth.RequireAuth(), middleware.RequireRole(user.RoleAdmin))
	admin.GET("/dashboard", func(c *gin.Context) {
		role, _ := c.Get(middleware.ContextRoleKey)
		c.JSON(http.StatusOK, gin.H{"role": role})
	})

	engine.GET("/public/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return engine
}

func mustToken(t *testing.T, secret, sub, role string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(30 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func mustExpiredToken(t *testing.T, secret, sub, role string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(-30 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	var out map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &out)
	require.NoError(t, err)
	return out
}
