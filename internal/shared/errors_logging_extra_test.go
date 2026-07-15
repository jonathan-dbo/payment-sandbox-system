package shared_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"github.com/stretchr/testify/assert"
)

func TestAppError_ErrorReturnsMessage(t *testing.T) {
	err := &shared.AppError{Code: "some_code", Message: "something went wrong", StatusCode: 400}
	assert.Equal(t, "something went wrong", err.Error())
}

func TestNewAuthError_Fields(t *testing.T) {
	err := shared.NewAuthError("invalid credentials")
	assert.Equal(t, "auth_error", err.Code)
	assert.Equal(t, "invalid credentials", err.Message)
	assert.Equal(t, 401, err.StatusCode)
}

func TestNewConflictError_Fields(t *testing.T) {
	err := shared.NewConflictError("already exists")
	assert.Equal(t, "conflict_error", err.Code)
	assert.Equal(t, "already exists", err.Message)
	assert.Equal(t, 409, err.StatusCode)
}

func TestLogEvent_DoesNotPanicAndRedactsSensitiveFields(t *testing.T) {
	// LogEvent writes to the standard logger; we only assert it completes
	// without panicking and accepts a mix of sensitive/non-sensitive fields.
	assert.NotPanics(t, func() {
		shared.LogEvent("test_event", map[string]any{
			"password":    "secret-value",
			"amount":      1000,
			"description": "ok",
		})
	})
}

func TestLogEvent_HandlesEmptyFields(t *testing.T) {
	assert.NotPanics(t, func() {
		shared.LogEvent("empty_event", map[string]any{})
	})
}

func TestLogEvent_HandlesNilFields(t *testing.T) {
	assert.NotPanics(t, func() {
		shared.LogEvent("nil_event", nil)
	})
}
