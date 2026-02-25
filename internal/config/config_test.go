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

	// Use unique test-only key names to avoid collisions with real .env values
	const (
		testDBKey     = "TEST_SECRETS_DATABASE_URL"
		testRedisKey  = "TEST_SECRETS_REDIS_URL"
		testSecretKey = "TEST_SECRETS_API_KEY"
		testBundleKey = "APP_SECRETS_JSON"
	)

	t.Run("no-op when APP_SECRETS_JSON is not set", func(t *testing.T) {
		os.Unsetenv(testBundleKey)
		if err := expandSecretsBundle(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("sets env vars from valid JSON bundle", func(t *testing.T) {
		defer cleanup(testDBKey, testRedisKey, testBundleKey)

		bundle := map[string]string{
			testDBKey:    "postgres://user:pass@host/db",
			testRedisKey: "redis://host:6379",
		}
		raw, _ := json.Marshal(bundle)
		os.Setenv(testBundleKey, string(raw))

		if err := expandSecretsBundle(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := os.Getenv(testDBKey); got != "postgres://user:pass@host/db" {
			t.Errorf("%s = %q, want %q", testDBKey, got, "postgres://user:pass@host/db")
		}
		if got := os.Getenv(testRedisKey); got != "redis://host:6379" {
			t.Errorf("%s = %q, want %q", testRedisKey, got, "redis://host:6379")
		}
	})

	t.Run("does not overwrite existing env vars (local .env takes precedence)", func(t *testing.T) {
		defer cleanup(testDBKey, testBundleKey)

		os.Setenv(testDBKey, "original-value") // already set (e.g. from .env)

		bundle := map[string]string{
			testDBKey: "secret-manager-value",
		}
		raw, _ := json.Marshal(bundle)
		os.Setenv(testBundleKey, string(raw))

		if err := expandSecretsBundle(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := os.Getenv(testDBKey); got != "original-value" {
			t.Errorf("%s = %q, want %q (should not be overwritten)", testDBKey, got, "original-value")
		}
	})

	t.Run("skips CHANGE_ME placeholder values", func(t *testing.T) {
		defer cleanup(testSecretKey, testBundleKey)

		bundle := map[string]string{
			testSecretKey: "CHANGE_ME",
		}
		raw, _ := json.Marshal(bundle)
		os.Setenv(testBundleKey, string(raw))

		if err := expandSecretsBundle(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got := os.Getenv(testSecretKey); got != "" {
			t.Errorf("%s should remain empty, got %q", testSecretKey, got)
		}
	})

	t.Run("returns error on invalid JSON", func(t *testing.T) {
		defer cleanup(testBundleKey)

		os.Setenv(testBundleKey, "not-valid-json{{{")

		if err := expandSecretsBundle(); err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}
