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

func TestInvoiceHandlerCreateAndList(t *testing.T) {
	h := handlers.NewUserHandler(
		appUser.NewUserService(database.NewInMemoryUserRepository(nil), "test-secret", 30),
		appInvoice.NewService(database.NewInMemoryInvoiceRepository(nil)),
	)

	createResp, err := h.CreateInvoice(context.Background(), apigen.CreateInvoiceRequestObject{
		Body: &apigen.CreateInvoiceJSONRequestBody{
			MerchantId: "m1",
			Amount:     1000,
		},
	})
	require.NoError(t, err)
	_, ok := createResp.(apigen.CreateInvoice201JSONResponse)
	require.True(t, ok)

	listResp, err := h.ListInvoices(context.Background(), apigen.ListInvoicesRequestObject{
		Params: apigen.ListInvoicesParams{MerchantId: ptr("m1"), Page: ptr("1"), PageSize: ptr("10")},
	})
	require.NoError(t, err)
	list, ok := listResp.(apigen.ListInvoices200JSONResponse)
	require.True(t, ok)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "m1", list.Items[0].MerchantId)
}

func ptr(v string) *string { return &v }
