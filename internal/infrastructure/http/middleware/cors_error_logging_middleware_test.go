package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCORSTestRouter(allowedOrigins []string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.CORSMiddleware(allowedOrigins))
	engine.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return engine
}

func TestCORSMiddleware_AllowAnyWhenNoOriginsConfigured(t *testing.T) {
	engine := newCORSTestRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_AllowsConfiguredOrigin(t *testing.T) {
	engine := newCORSTestRouter([]string{"https://allowed.example.com", " https://also-allowed.example.com "})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://allowed.example.com")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "https://allowed.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Origin", rec.Header().Get("Vary"))
}

func TestCORSMiddleware_TrimsWhitespaceInConfiguredOrigins(t *testing.T) {
	engine := newCORSTestRouter([]string{" https://also-allowed.example.com "})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://also-allowed.example.com")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, "https://also-allowed.example.com", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_RejectsUnlistedOrigin(t *testing.T) {
	engine := newCORSTestRouter([]string{"https://allowed.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://not-allowed.example.com")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_NoOriginHeaderSkipsAllowOriginHeader(t *testing.T) {
	engine := newCORSTestRouter([]string{"https://allowed.example.com"})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddleware_SetsStandardHeaders(t *testing.T) {
	engine := newCORSTestRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, "GET,POST,PUT,DELETE,OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Authorization,Content-Type,Accept,Origin", rec.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "86400", rec.Header().Get("Access-Control-Max-Age"))
}

func TestCORSMiddleware_PreflightOptionsShortCircuits(t *testing.T) {
	engine := newCORSTestRouter(nil)
	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// -------------------- Error middleware --------------------

func newErrorTestRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.ErrorHandlerMiddleware())
	engine.GET("/boom", handler)
	return engine
}

func TestErrorHandlerMiddleware_RecoversFromStringPanic(t *testing.T) {
	engine := newErrorTestRouter(func(c *gin.Context) {
		panic("custom panic message")
	})
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "custom panic message")
	assert.Contains(t, rec.Body.String(), "internal_server_error")
}

func TestErrorHandlerMiddleware_RecoversFromNonStringPanic(t *testing.T) {
	engine := newErrorTestRouter(func(c *gin.Context) {
		panic(errPanicSentinel)
	})
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "An unexpected error occurred")
}

func TestErrorHandlerMiddleware_NoPanicPassesThrough(t *testing.T) {
	engine := newErrorTestRouter(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// -------------------- Structured logging middleware --------------------

func TestStructuredLoggingMiddleware_WrapsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.StructuredLoggingMiddleware())
	engine.GET("/logged", func(c *gin.Context) {
		c.JSON(http.StatusTeapot, gin.H{"status": "teapot"})
	})

	req := httptest.NewRequest(http.MethodGet, "/logged", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTeapot, rec.Code)
}

var errPanicSentinel = assertionError("boom")

type assertionError string

func (e assertionError) Error() string { return string(e) }
