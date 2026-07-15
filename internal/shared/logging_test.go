package shared_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"github.com/stretchr/testify/assert"
)

func TestLoggingSensitiveDataExclusion(t *testing.T) {
	fields := map[string]any{
		"password":      "p@ss",
		"access_token":  "abc",
		"jwt_secret":    "secret",
		"normal_field":  "visible",
		"merchant_id":   "m-1",
		"request_token": "rt",
	}
	redacted := shared.RedactFields(fields)
	assert.Equal(t, "[REDACTED]", redacted["password"])
	assert.Equal(t, "[REDACTED]", redacted["access_token"])
	assert.Equal(t, "[REDACTED]", redacted["jwt_secret"])
	assert.Equal(t, "[REDACTED]", redacted["request_token"])
	assert.Equal(t, "visible", redacted["normal_field"])
	assert.Equal(t, "m-1", redacted["merchant_id"])
}
