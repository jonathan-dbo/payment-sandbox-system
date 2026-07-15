package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	domainUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthRegisterSuccess(t *testing.T) {
	repo := database.NewInMemoryUserRepository(nil)
	svc := appUser.NewUserService(repo, "test-secret", 30)

	result, err := svc.Register(context.Background(), "Demo", "merchant@example.com", "password123", "")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, "merchant@example.com", result.Email)
	assert.Equal(t, "MERCHANT", result.Role)
	assert.True(t, result.ExpiresAt.After(time.Now().Add(20*time.Minute)))

	stored, err := repo.FindByEmail("merchant@example.com")
	require.NoError(t, err)
	assert.NotEqual(t, "password123", stored.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("password123")))
	assert.True(t, repo.MerchantExistsForUser(stored.ID))
}

func TestAuthRegisterDuplicateEmail(t *testing.T) {
	repo := database.NewInMemoryUserRepository(nil)
	svc := appUser.NewUserService(repo, "test-secret", 30)
	_, err := svc.Register(context.Background(), "Demo", "merchant@example.com", "password123", "")
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), "Demo 2", "merchant@example.com", "password456", "")
	require.Error(t, err)

	var appErr *shared.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "conflict_error", appErr.Code)
}

func TestAuthLoginSuccess(t *testing.T) {
	hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	repo := database.NewInMemoryUserRepository([]*domainUser.User{
		{ID: "u1", Email: "merchant@example.com", PasswordHash: string(hashed), Role: "MERCHANT"},
	})
	svc := appUser.NewUserService(repo, "test-secret", 45)

	result, err := svc.Login(context.Background(), "merchant@example.com", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Token)
	assert.Equal(t, "u1", result.UserID)
	assert.Equal(t, "MERCHANT", result.Role)
}

func TestAuthLoginInvalidPassword(t *testing.T) {
	hashed, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	require.NoError(t, err)
	repo := database.NewInMemoryUserRepository([]*domainUser.User{
		{ID: "u1", Email: "merchant@example.com", PasswordHash: string(hashed), Role: "MERCHANT"},
	})
	svc := appUser.NewUserService(repo, "test-secret", 45)

	_, err = svc.Login(context.Background(), "merchant@example.com", "wrong-password")
	require.Error(t, err)

	var appErr *shared.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "auth_error", appErr.Code)
}

func TestAuthJWTExpiryClaimValidation(t *testing.T) {
	repo := database.NewInMemoryUserRepository(nil)
	svc := appUser.NewUserService(repo, "test-secret", 15)

	result, err := svc.Register(context.Background(), "Demo", "merchant@example.com", "password123", "")
	require.NoError(t, err)

	parsed, err := jwt.Parse(result.Token, func(token *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)

	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	exp, err := claims.GetExpirationTime()
	require.NoError(t, err)
	assert.WithinDuration(t, result.ExpiresAt, exp.Time, time.Second)
}

func TestAuthRegisterSupportsAllRoles(t *testing.T) {
	repo := database.NewInMemoryUserRepository(nil)
	svc := appUser.NewUserService(repo, "test-secret", 30)

	admin, err := svc.Register(context.Background(), "Admin", "admin@example.com", "password123", "ADMIN")
	require.NoError(t, err)
	assert.Equal(t, "ADMIN", admin.Role)
	assert.False(t, repo.MerchantExistsForUser(admin.UserID))

	endUser, err := svc.Register(context.Background(), "User", "user@example.com", "password123", "USER")
	require.NoError(t, err)
	assert.Equal(t, "USER", endUser.Role)
	assert.False(t, repo.MerchantExistsForUser(endUser.UserID))

	merchant, err := svc.Register(context.Background(), "Merchant", "merchant2@example.com", "password123", "MERCHANT")
	require.NoError(t, err)
	assert.Equal(t, "MERCHANT", merchant.Role)
	assert.True(t, repo.MerchantExistsForUser(merchant.UserID))
}

func TestAuthRegisterRejectsInvalidRole(t *testing.T) {
	repo := database.NewInMemoryUserRepository(nil)
	svc := appUser.NewUserService(repo, "test-secret", 30)

	_, err := svc.Register(context.Background(), "Demo", "role-invalid@example.com", "password123", "SUPERADMIN")
	require.Error(t, err)

	var appErr *shared.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "auth_error", appErr.Code)
}

func TestAuthRefreshSuccess(t *testing.T) {
	repo := database.NewInMemoryUserRepository(nil)
	svc := appUser.NewUserService(repo, "test-secret", 30)

	registered, err := svc.Register(context.Background(), "Demo", "refresh@example.com", "password123", "MERCHANT")
	require.NoError(t, err)

	refreshed, err := svc.Refresh(context.Background(), registered.UserID)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.Token)
	assert.Equal(t, registered.UserID, refreshed.UserID)
	assert.Equal(t, "MERCHANT", refreshed.Role)
}
