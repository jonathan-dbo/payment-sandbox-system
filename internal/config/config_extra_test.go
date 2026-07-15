package config_test

import (
	"testing"

	"github.com/gonszalito/go-ddd-architecture/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadFromEnv_InvalidDBPort(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "JWT_SECRET":
			return "super-secret"
		case "DB_PORT":
			return "not-a-number"
		default:
			return ""
		}
	}
	_, err := config.LoadFromEnv(getenv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid DB_PORT")
}

func TestConfigLoadFromEnv_InvalidJWTExpiryMinutes(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "JWT_SECRET":
			return "super-secret"
		case "JWT_EXPIRY_MINUTES":
			return "not-a-number"
		default:
			return ""
		}
	}
	_, err := config.LoadFromEnv(getenv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JWT_EXPIRY_MINUTES")
}

func TestConfigLoadFromEnv_MissingDBHostRejected(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "JWT_SECRET":
			return "super-secret"
		case "DB_HOST":
			return "   "
		default:
			return ""
		}
	}
	_, err := config.LoadFromEnv(getenv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required postgres config")
}

func TestConfigLoadFromEnv_DefaultJWTSecretRejected(t *testing.T) {
	getenv := func(key string) string { return "" }
	_, err := config.LoadFromEnv(getenv)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestConfigLoadFromEnv_CustomCORSOrigins(t *testing.T) {
	getenv := func(key string) string {
		switch key {
		case "JWT_SECRET":
			return "super-secret"
		case "CORS_ALLOWED_ORIGINS":
			return "https://a.example.com, , https://b.example.com,"
		default:
			return ""
		}
	}
	cfg, err := config.LoadFromEnv(getenv)
	require.NoError(t, err)
	assert.Equal(t, []string{"https://a.example.com", "https://b.example.com"}, cfg.CORSAllowedOrigins)
}

func TestConfigLoadFromEnv_AllDefaultsApplied(t *testing.T) {
	getenv := func(key string) string {
		if key == "JWT_SECRET" {
			return "super-secret"
		}
		return ""
	}
	cfg, err := config.LoadFromEnv(getenv)
	require.NoError(t, err)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "localhost", cfg.DBHost)
	assert.Equal(t, 5432, cfg.DBPort)
	assert.Equal(t, "user", cfg.DBUser)
	assert.Equal(t, "password", cfg.DBPassword)
	assert.Equal(t, "postgres", cfg.DBName)
	assert.Equal(t, 60, cfg.JWTExpiryMinutes)
}

func TestMustLoad_PanicsWhenJWTSecretMissing(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	assert.Panics(t, func() {
		config.MustLoad()
	})
}

func TestMustLoad_SucceedsWithValidConfig(t *testing.T) {
	t.Setenv("JWT_SECRET", "super-secret-for-must-load")
	assert.NotPanics(t, func() {
		cfg := config.MustLoad()
		assert.Equal(t, "super-secret-for-must-load", cfg.JWTSecret)
	})
}
