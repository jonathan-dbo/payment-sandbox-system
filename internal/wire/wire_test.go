package wire_test

import (
	"os"
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/wire"
	"github.com/stretchr/testify/require"
)

func TestWireAppInitializationSmoke(t *testing.T) {
	if os.Getenv("INTEGRATION_DB_DSN") == "" {
		t.Skip("set INTEGRATION_DB_DSN to run wire integration smoke test")
	}

	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DATABASE_URL", os.Getenv("INTEGRATION_DB_DSN"))

	app, cleanup, err := wire.InitializeApp()
	require.NoError(t, err)
	require.NotNil(t, app)
	require.NotNil(t, app.UserService)
	require.NotNil(t, app.InvoiceService)
	require.NotNil(t, app.PaymentService)
	require.NotNil(t, app.RefundService)
	require.NotNil(t, app.TopUpService)
	require.NotNil(t, app.Dashboard)
	if cleanup != nil {
		cleanup()
	}
}
