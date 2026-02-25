package config

import (
	"encoding/json"
	"os"
	"testing"
)

func TestExpandSecretsBundle(t *testing.T) {
	// Helper to clean env vars after each sub-test
	cleanup := func(keys ...string) {
		for _, k := range keys {
			os.Unsetenv(k)
		}
	}

	t.Run("no-op when APP_SECRETS_JSON is not set", func(t *testing.T) {
		os.Unsetenv("APP_SECRETS_JSON")
		if err := expandSecretsBundle(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("sets env vars from valid JSON bundle", func(t *testing.T) {
		defer cleanup("DATABASE_URL", "REDIS_URL", "APP_SECRETS_JSON")

		bundle := map[string]string{
			"DATABASE_URL": "postgres://user:pass@host/db",
			"REDIS_URL":    "redis://host:6379",
		}
		raw, _ := json.Marshal(bundle)
		os.Setenv("APP_SECRETS_JSON", string(raw))

		if err := expandSecretsBundle(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := os.Getenv("DATABASE_URL"); got != "postgres://user:pass@host/db" {
			t.Errorf("DATABASE_URL = %q, want %q", got, "postgres://user:pass@host/db")
		}
		if got := os.Getenv("REDIS_URL"); got != "redis://host:6379" {
			t.Errorf("REDIS_URL = %q, want %q", got, "redis://host:6379")
		}
	})

	t.Run("does not overwrite existing env vars (local .env takes precedence)", func(t *testing.T) {
		defer cleanup("DATABASE_URL", "APP_SECRETS_JSON")

		os.Setenv("DATABASE_URL", "original-value") // already set (e.g. from .env)

		bundle := map[string]string{
			"DATABASE_URL": "secret-manager-value",
		}
		raw, _ := json.Marshal(bundle)
		os.Setenv("APP_SECRETS_JSON", string(raw))

		if err := expandSecretsBundle(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := os.Getenv("DATABASE_URL"); got != "original-value" {
			t.Errorf("DATABASE_URL = %q, want %q (should not be overwritten)", got, "original-value")
		}
	})

	t.Run("skips CHANGE_ME placeholder values", func(t *testing.T) {
		defer cleanup("API_SECRET_KEY", "APP_SECRETS_JSON")

		bundle := map[string]string{
			"API_SECRET_KEY": "CHANGE_ME",
		}
		raw, _ := json.Marshal(bundle)
		os.Setenv("APP_SECRETS_JSON", string(raw))

		if err := expandSecretsBundle(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := os.Getenv("API_SECRET_KEY"); got != "" {
			t.Errorf("API_SECRET_KEY should remain empty, got %q", got)
		}
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		defer cleanup("APP_SECRETS_JSON")

		os.Setenv("APP_SECRETS_JSON", "not-valid-json{{{")

		if err := expandSecretsBundle(); err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}
