package handlers_test

import (
	"context"
	"testing"

	appInvoice "github.com/gonszalito/go-ddd-architecture/internal/application/invoice"
	appUser "github.com/gonszalito/go-ddd-architecture/internal/application/user"
	"github.com/gonszalito/go-ddd-architecture/internal/infrastructure/database"
	apigen "github.com/gonszalito/go-ddd-architecture/internal/interfaces"
	"github.com/gonszalito/go-ddd-architecture/internal/interfaces/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandlerRegisterSuccess(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	h := handlers.NewUserHandler(svc, appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)))

	resp, err := h.Register(context.Background(), apigen.RegisterRequestObject{
		Body: &apigen.RegisterJSONRequestBody{
			Name:     "Merchant 1",
			Email:    "merchant@example.com",
			Password: "password123",
			Role:     rolePtr("MERCHANT"),
		},
	})
	require.NoError(t, err)

	success, ok := resp.(apigen.Register201JSONResponse)
	require.True(t, ok)
	assert.NotEmpty(t, success.Token)
	assert.Equal(t, "merchant@example.com", success.Email)
}

func TestAuthHandlerRegisterDuplicateEmail(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	h := handlers.NewUserHandler(svc, appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)))

	_, err := h.Register(context.Background(), apigen.RegisterRequestObject{
		Body: &apigen.RegisterJSONRequestBody{
			Name:     "Merchant 1",
			Email:    "merchant@example.com",
			Password: "password123",
			Role:     rolePtr("MERCHANT"),
		},
	})
	require.NoError(t, err)

	resp, err := h.Register(context.Background(), apigen.RegisterRequestObject{
		Body: &apigen.RegisterJSONRequestBody{
			Name:     "Merchant 2",
			Email:    "merchant@example.com",
			Password: "password123",
			Role:     rolePtr("MERCHANT"),
		},
	})
	require.NoError(t, err)

	conflict, ok := resp.(apigen.Register409JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "conflict_error", conflict.Error)
}

func TestAuthHandlerLoginInvalidPassword(t *testing.T) {
	svc := appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30)
	h := handlers.NewUserHandler(svc, appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)))

	_, err := h.Register(context.Background(), apigen.RegisterRequestObject{
		Body: &apigen.RegisterJSONRequestBody{
			Name:     "Merchant 1",
			Email:    "merchant@example.com",
			Password: "password123",
			Role:     rolePtr("MERCHANT"),
		},
	})
	require.NoError(t, err)

	resp, err := h.Login(context.Background(), apigen.LoginRequestObject{
		Body: &apigen.LoginJSONRequestBody{
			Email:    "merchant@example.com",
			Password: "wrong-password",
		},
	})
	require.NoError(t, err)

	authErr, ok := resp.(apigen.Login401JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "auth_error", authErr.Error)
}

func rolePtr(v string) *apigen.RegisterRequestRole {
	role := apigen.RegisterRequestRole(v)
	return &role
}
