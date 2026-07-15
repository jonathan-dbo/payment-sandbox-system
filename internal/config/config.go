// Package config handles environment-based application configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DatabaseURL        string
	DBHost             string
	DBPort             int
	DBUser             string
	DBPassword         string
	DBName             string
	JWTSecret          string
	JWTExpiryMinutes   int
	CORSAllowedOrigins []string
}

func Load() (*Config, error) {
	_ = godotenv.Load(".env.example")
	return LoadFromEnv(os.Getenv)
}

func LoadFromEnv(getenv func(string) string) (*Config, error) {
	dbPort, err := strconv.Atoi(getEnvWith("DB_PORT", "5432", getenv))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}

	jwtExpiryMinutes, err := strconv.Atoi(getEnvWith("JWT_EXPIRY_MINUTES", "60", getenv))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_MINUTES: %w", err)
	}

	cfg := &Config{
		Port:             getEnvWith("PORT", "8080", getenv),
		DatabaseURL:      getEnvWith("DATABASE_URL", "", getenv),
		DBHost:           getEnvWith("DB_HOST", "localhost", getenv),
		DBPort:           dbPort,
		DBUser:           getEnvWith("DB_USER", "user", getenv),
		DBPassword:       getEnvWith("DB_PASSWORD", "password", getenv),
		DBName:           getEnvWith("DB_NAME", "postgres", getenv),
		JWTSecret:        getEnvWith("JWT_SECRET", "change-me", getenv),
		JWTExpiryMinutes: jwtExpiryMinutes,
		CORSAllowedOrigins: parseCSV(
			getEnvWith("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173", getenv),
		),
	}

	if strings.TrimSpace(cfg.JWTSecret) == "" || cfg.JWTSecret == "change-me" {
		return nil, fmt.Errorf("missing required config: JWT_SECRET")
	}

	if strings.TrimSpace(cfg.DBHost) == "" || strings.TrimSpace(cfg.DBUser) == "" || strings.TrimSpace(cfg.DBName) == "" {
		return nil, fmt.Errorf("missing required postgres config: DB_HOST, DB_USER, DB_NAME")
	}

	return cfg, nil
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func getEnv(key, fallback string) string {
	return getEnvWith(key, fallback, os.Getenv)
}

func getEnvWith(key, fallback string, getenv func(string) string) string {
	if value := getenv(key); value != "" {
		return value
	}
	return fallback
}
