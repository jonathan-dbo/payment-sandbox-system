package config_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigEnvPrecedence(t *testing.T) {
	t.Setenv("JWT_SECRET", "super-secret")
	t.Setenv("PORT", "9090")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "super-secret", cfg.JWTSecret)
	assert.Equal(t, "9090", cfg.Port)
}

func TestConfigMissingRequiredVariable(t *testing.T) {
	getenv := func(key string) string {
		if key == "JWT_SECRET" {
			return ""
		}
		return ""
	}
	_, err := config.LoadFromEnv(getenv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}
