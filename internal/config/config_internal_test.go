package config

import "testing"

// TestGetEnv exercises the unexported getEnv helper (a thin wrapper around
// getEnvWith using os.Getenv) which otherwise has no callers in this
// codebase but is kept as a small public-ish convenience helper.
func TestGetEnv(t *testing.T) {
	t.Setenv("CONFIG_GET_ENV_TEST_KEY", "actual-value")
	if got := getEnv("CONFIG_GET_ENV_TEST_KEY", "fallback"); got != "actual-value" {
		t.Fatalf("expected actual-value, got %s", got)
	}

	t.Setenv("CONFIG_GET_ENV_TEST_KEY_UNSET", "")
	if got := getEnv("CONFIG_GET_ENV_TEST_KEY_UNSET", "fallback-value"); got != "fallback-value" {
		t.Fatalf("expected fallback-value, got %s", got)
	}
}
