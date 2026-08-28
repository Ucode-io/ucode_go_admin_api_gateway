package config

import (
	"os"
	"path/filepath"
	"testing"

	"ucode/ucode_go_api_gateway/pkg/secureenv"
)

func TestLoadEncryptedMetaAdsEnvironment(t *testing.T) {
	t.Setenv("SECRET_KEY", "runtime-secret")
	restoreEnv(t, "META_ACCESS_TOKEN")
	restoreEnv(t, "META_AD_ACCOUNT_ID")
	os.Unsetenv("META_ACCESS_TOKEN")
	os.Unsetenv("META_AD_ACCOUNT_ID")

	encoded, err := secureenv.Encrypt([]byte("META_ACCESS_TOKEN=meta-secret\nMETA_AD_ACCOUNT_ID=123\n"), "runtime-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "meta-ads.env.enc")
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := loadEncryptedMetaAdsEnvironment(path); err != nil {
		t.Fatalf("loadEncryptedMetaAdsEnvironment() error = %v", err)
	}
	if got := os.Getenv("META_ACCESS_TOKEN"); got != "meta-secret" {
		t.Fatalf("META_ACCESS_TOKEN = %q, want %q", got, "meta-secret")
	}
	if got := os.Getenv("META_AD_ACCOUNT_ID"); got != "123" {
		t.Fatalf("META_AD_ACCOUNT_ID = %q, want %q", got, "123")
	}
}

func TestLoadEncryptedMetaAdsEnvironmentDoesNotOverrideExistingValue(t *testing.T) {
	t.Setenv("SECRET_KEY", "runtime-secret")
	t.Setenv("META_ACCESS_TOKEN", "existing-secret")

	encoded, err := secureenv.Encrypt([]byte("META_ACCESS_TOKEN=encrypted-secret\n"), "runtime-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "meta-ads.env.enc")
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := loadEncryptedMetaAdsEnvironment(path); err != nil {
		t.Fatalf("loadEncryptedMetaAdsEnvironment() error = %v", err)
	}
	if got := os.Getenv("META_ACCESS_TOKEN"); got != "existing-secret" {
		t.Fatalf("META_ACCESS_TOKEN = %q, want existing value", got)
	}
}

func restoreEnv(t *testing.T, key string) {
	t.Helper()
	value, exists := os.LookupEnv(key)
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}
