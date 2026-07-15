package user_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	domainUser "github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthRefresh_EmptyUserIDRejected(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	_, err := svc.Refresh(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user context")
}

func TestAuthRefresh_UnknownUserIDRejected(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	_, err := svc.Refresh(context.Background(), "does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user context")
}

func TestAuthRegister_DefaultsToMerchantRoleWhenEmpty(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	result, err := svc.Register(context.Background(), "Default Role", "default-role@example.com", "secret123", "")
	require.NoError(t, err)
	assert.Equal(t, domainUser.RoleMerchant, result.Role)
}

func TestAuthRegister_PropagatesNonNotFoundLookupError(t *testing.T) {
	repo := &failingLookupUserRepository{failWith: errors.New("db connection lost")}
	svc := appUser.NewUserService(repo, "test-secret", 30)
	_, err := svc.Register(context.Background(), "Whoever", "whoever@example.com", "secret123", "MERCHANT")
	require.Error(t, err)
	assert.Equal(t, "db connection lost", err.Error())
}

func TestAuthRegister_TreatsSQLNoRowsAsNotFound(t *testing.T) {
	repo := &failingLookupUserRepository{failWith: sql.ErrNoRows}
	svc := appUser.NewUserService(repo, "test-secret", 30)
	result, err := svc.Register(context.Background(), "Fresh User", "fresh@example.com", "secret123", "MERCHANT")
	require.NoError(t, err)
	assert.Equal(t, "fresh@example.com", result.Email)
}

// failingLookupUserRepository always fails FindByEmail with a configured
// error, to exercise the non-not-found-error propagation branch of Register,
// and the sql.ErrNoRows-treated-as-not-found branch of isUserNotFoundError.
type failingLookupUserRepository struct {
	failWith error
	created  *domainUser.User
}

func (r *failingLookupUserRepository) FindByID(id string) (*domainUser.User, error) {
	return nil, errors.New("not implemented")
}
func (r *failingLookupUserRepository) FindByEmail(email string) (*domainUser.User, error) {
	return nil, r.failWith
}
func (r *failingLookupUserRepository) Create(u *domainUser.User) error {
	r.created = u
	return nil
}
func (r *failingLookupUserRepository) CreateWithMerchant(u *domainUser.User) error {
	r.created = u
	return nil
}
func (r *failingLookupUserRepository) Save(u *domainUser.User) error {
	return nil
}
