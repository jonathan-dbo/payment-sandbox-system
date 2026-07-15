package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
	"github.com/stretchr/testify/assert"
)

// TestRequireRole_MissingAuthContext exercises RequireRole's "missing auth
// context" branch (auth_middleware.go:71-77), which is only reachable when
// RequireRole is mounted without a preceding RequireAuth() (or equivalent)
// call populating ContextRoleKey.
func TestRequireRole_MissingAuthContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/no-context", middleware.RequireRole("ADMIN"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/no-context", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeBody(t, rec)
	assert.Equal(t, "auth_error", resp["error"])
	assert.Equal(t, "missing auth context", resp["message"])
}

// TestRequireRole_InvalidAuthContextType exercises RequireRole's "invalid
// auth context" branch (auth_middleware.go:80-86), reached when
// ContextRoleKey holds a non-string value.
func TestRequireRole_InvalidAuthContextType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/bad-role-type", func(c *gin.Context) {
		c.Set(middleware.ContextRoleKey, 12345) // wrong type on purpose
		c.Next()
	}, middleware.RequireRole("ADMIN"), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/bad-role-type", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeBody(t, rec)
	assert.Equal(t, "auth_error", resp["error"])
	assert.Equal(t, "invalid auth context", resp["message"])
}

// TestRequireAuth_ExpiredToken exercises the jwt.Parse error branch via an
// already-expired token (distinct from TestRoleInvalidToken's malformed
// token string, which fails parsing for a different reason).
func TestRequireAuth_ExpiredToken(t *testing.T) {
	engine := setupRoleTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/merchant/profile", nil)
	req.Header.Set("Authorization", "Bearer "+mustExpiredToken(t, "test-secret", "u1", "MERCHANT"))
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeBody(t, rec)
	assert.Equal(t, "auth_error", resp["error"])
	assert.Equal(t, "invalid token", resp["message"])
}

// TestRequireAuth_MissingBearerPrefix exercises the "missing or invalid
// authorization header" branch when the header value doesn't include the
// "Bearer " prefix, so TrimPrefix leaves it unchanged.
func TestRequireAuth_MissingBearerPrefix(t *testing.T) {
	engine := setupRoleTestRouter("test-secret")
	req := httptest.NewRequest(http.MethodGet, "/merchant/profile", nil)
	req.Header.Set("Authorization", "not-a-bearer-token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	resp := decodeBody(t, rec)
	assert.Equal(t, "auth_error", resp["error"])
	assert.Equal(t, "missing or invalid authorization header", resp["message"])
}
