package user_test

import (
	"testing"
	"time"

	"github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/stretchr/testify/assert"
)

func TestUserChangeEmail(t *testing.T) {
	u := &user.User{
		ID:    "u-1",
		Name:  "Jane Doe",
		Email: "old@example.com",
		Role:  user.RoleMerchant,
	}
	u.ChangeEmail("new@example.com")
	assert.Equal(t, "new@example.com", u.Email)
}

func TestUserChangeEmail_EmptyString(t *testing.T) {
	u := &user.User{Email: "old@example.com"}
	u.ChangeEmail("")
	assert.Equal(t, "", u.Email)
}

func TestUserRoleConstants(t *testing.T) {
	assert.Equal(t, "MERCHANT", user.RoleMerchant)
	assert.Equal(t, "ADMIN", user.RoleAdmin)
	assert.Equal(t, "USER", user.RoleUser)
}

func TestUserFieldAssignment(t *testing.T) {
	now := time.Now().UTC()
	u := &user.User{
		ID:           "u-1",
		Name:         "Jane Doe",
		Email:        "jane@example.com",
		PasswordHash: "hashed",
		Role:         user.RoleAdmin,
		CreatedAt:    now,
	}
	assert.Equal(t, "u-1", u.ID)
	assert.Equal(t, "Jane Doe", u.Name)
	assert.Equal(t, "jane@example.com", u.Email)
	assert.Equal(t, "hashed", u.PasswordHash)
	assert.Equal(t, user.RoleAdmin, u.Role)
	assert.Equal(t, now, u.CreatedAt)
}
