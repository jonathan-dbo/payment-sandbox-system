package middleware_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/http/middleware"
)

// mustBenchToken builds a signed JWT for benchmark setup (mirrors mustToken
// in auth_middleware_test.go, but accepts *testing.B instead of *testing.T).
func mustBenchToken(b *testing.B, secret, sub, role string) string {
	b.Helper()
	claims := jwt.MapClaims{
		"sub":  sub,
		"role": role,
		"exp":  time.Now().Add(30 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		b.Fatalf("failed to sign bench token: %v", err)
	}
	return signed
}

// silenceGinAndLog discards gin's own request-logger output and the
// standard log package output for the duration of a benchmark, and puts gin
// in release mode, so per-request logging I/O doesn't distort results.
func silenceGinAndLog(b *testing.B) {
	b.Helper()
	prevGinMode := gin.Mode()
	gin.SetMode(gin.ReleaseMode)
	prevGinWriter := gin.DefaultWriter
	gin.DefaultWriter = io.Discard
	prevLogOut := log.Writer()
	log.SetOutput(io.Discard)
	b.Cleanup(func() {
		gin.SetMode(prevGinMode)
		gin.DefaultWriter = prevGinWriter
		log.SetOutput(prevLogOut)
	})
}

// BenchmarkAuthMiddleware_RequireAuth measures JWT parsing + claims
// extraction cost for an authenticated request.
func BenchmarkAuthMiddleware_RequireAuth(b *testing.B) {
	silenceGinAndLog(b)
	auth := middleware.NewAuthMiddleware("bench-secret")
	engine := gin.New()
	engine.Use(auth.RequireAuth())
	engine.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	token := mustBenchToken(b, "bench-secret", "u-1", user.RoleMerchant)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

// BenchmarkRequireRole measures the role-check middleware in isolation
// (post-auth context lookup + allow-list check).
func BenchmarkRequireRole(b *testing.B) {
	silenceGinAndLog(b)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set(middleware.ContextRoleKey, user.RoleMerchant)
		c.Next()
	})
	engine.Use(middleware.RequireRole(user.RoleMerchant))
	engine.GET("/merchant-only", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/merchant-only", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

// BenchmarkCORSMiddleware measures the CORS header-setting hot path for a
// simple GET request with an Origin header.
func BenchmarkCORSMiddleware(b *testing.B) {
	silenceGinAndLog(b)
	engine := gin.New()
	engine.Use(middleware.CORSMiddleware([]string{"https://allowed.example.com"}))
	engine.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://allowed.example.com")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}

// BenchmarkStructuredLoggingMiddleware measures the request-wrapping and
// LogEvent-emission overhead added by StructuredLoggingMiddleware.
func BenchmarkStructuredLoggingMiddleware(b *testing.B) {
	silenceGinAndLog(b)
	engine := gin.New()
	engine.Use(middleware.StructuredLoggingMiddleware())
	engine.GET("/logged", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/logged", nil)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rec.Code)
		}
	}
}
