package user_test

import (
	"context"
	"errors"
	"testing"

	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	domainUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRegister_EmptyEmailRejected(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	_, err := svc.Register(context.Background(), "Demo", "  ", "password123", "MERCHANT")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email and password are required")
}

func TestAuthRegister_EmptyPasswordRejected(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	_, err := svc.Register(context.Background(), "Demo", "demo@example.com", "   ", "MERCHANT")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email and password are required")
}

func TestAuthRegister_CreateWithMerchantErrorPropagates(t *testing.T) {
	repo := &createFailingUserRepository{failWith: errors.New("insert failed")}
	svc := appUser.NewUserService(repo, "test-secret", 30)
	_, err := svc.Register(context.Background(), "Demo", "demo@example.com", "password123", "MERCHANT")
	require.Error(t, err)
	assert.Equal(t, "insert failed", err.Error())
}

func TestAuthLogin_UnknownEmailRejected(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	_, err := svc.Login(context.Background(), "does-not-exist@example.com", "whatever")
	require.Error(t, err)

	var appErr *shared.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "auth_error", appErr.Code)
}

func TestIsUserNotFoundError_MatchesPlainNotFoundString(t *testing.T) {
	repo := &lookupErrorUserRepository{failWith: errors.New("not_found")}
	svc := appUser.NewUserService(repo, "test-secret", 30)
	// Register should treat "not_found" string error as "no existing user" and proceed.
	result, err := svc.Register(context.Background(), "Demo", "demo2@example.com", "password123", "MERCHANT")
	require.NoError(t, err)
	assert.Equal(t, "demo2@example.com", result.Email)
}

func TestIsUserNotFoundError_MatchesUserNotFoundString(t *testing.T) {
	repo := &lookupErrorUserRepository{failWith: errors.New("user not found")}
	svc := appUser.NewUserService(repo, "test-secret", 30)
	result, err := svc.Register(context.Background(), "Demo", "demo3@example.com", "password123", "MERCHANT")
	require.NoError(t, err)
	assert.Equal(t, "demo3@example.com", result.Email)
}

// createFailingUserRepository allows FindByEmail to report not-found (so
// Register proceeds) but fails CreateWithMerchant.
type createFailingUserRepository struct {
	failWith error
}

func (r *createFailingUserRepository) FindByID(id string) (*domainUser.User, error) {
	return nil, errors.New("not implemented")
}
func (r *createFailingUserRepository) FindByEmail(email string) (*domainUser.User, error) {
	return nil, errors.New("user not found")
}
func (r *createFailingUserRepository) Create(u *domainUser.User) error { return nil }
func (r *createFailingUserRepository) CreateWithMerchant(u *domainUser.User) error {
	return r.failWith
}
func (r *createFailingUserRepository) Save(u *domainUser.User) error { return nil }

// lookupErrorUserRepository always fails FindByEmail with a configured error
// string, to exercise isUserNotFoundError's string-matching branches.
type lookupErrorUserRepository struct {
	failWith error
	created  *domainUser.User
}

func (r *lookupErrorUserRepository) FindByID(id string) (*domainUser.User, error) {
	return nil, errors.New("not implemented")
}
func (r *lookupErrorUserRepository) FindByEmail(email string) (*domainUser.User, error) {
	return nil, r.failWith
}
func (r *lookupErrorUserRepository) Create(u *domainUser.User) error {
	r.created = u
	return nil
}
func (r *lookupErrorUserRepository) CreateWithMerchant(u *domainUser.User) error {
	r.created = u
	return nil
}
func (r *lookupErrorUserRepository) Save(u *domainUser.User) error { return nil }
