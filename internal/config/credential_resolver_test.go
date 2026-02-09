package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/encryption"
)

func TestResolveCredential_FromEnv(t *testing.T) {
	// Set up env var
	os.Setenv("WIFIMGR_API_MIST_CREDENTIALS_API_TOKEN", "test-token-from-env")
	defer os.Unsetenv("WIFIMGR_API_MIST_CREDENTIALS_API_TOKEN")

	// Resolve credential
	value, err := ResolveCredential("api.mist.credentials.api_token")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if value != "test-token-from-env" {
		t.Errorf("expected 'test-token-from-env', got %q", value)
	}
}

func TestResolveCredential_FromConfig(t *testing.T) {
	// Set up Viper
	viper.Reset()
	viper.Set("api.mist.credentials.api_token", "test-token-from-config")
	defer viper.Reset()

	// Resolve credential
	value, err := ResolveCredential("api.mist.credentials.api_token")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if value != "test-token-from-config" {
		t.Errorf("expected 'test-token-from-config', got %q", value)
	}
}

func TestResolveCredential_EnvTakesPrecedence(t *testing.T) {
	// Set up both env var and config
	os.Setenv("WIFIMGR_API_MIST_CREDENTIALS_API_TOKEN", "from-env")
	defer os.Unsetenv("WIFIMGR_API_MIST_CREDENTIALS_API_TOKEN")

	viper.Reset()
	viper.Set("api.mist.credentials.api_token", "from-config")
	defer viper.Reset()

	// Env should take precedence
	value, err := ResolveCredential("api.mist.credentials.api_token")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if value != "from-env" {
		t.Errorf("expected 'from-env' (env takes precedence), got %q", value)
	}
}

func TestResolveCredential_DecryptsEncryptedValue(t *testing.T) {
	// Encrypt a test value
	password := "testpassword123"
	plaintext := "my-secret-token"
	encrypted, err := encryption.Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Set up Viper with encrypted value
	viper.Reset()
	viper.Set("api.mist.credentials.api_token", encrypted)
	defer viper.Reset()

	// Set password env var
	os.Setenv("WIFIMGR_PASSWORD", password)
	defer os.Unsetenv("WIFIMGR_PASSWORD")

	// Resolve credential - should decrypt
	value, err := ResolveCredential("api.mist.credentials.api_token")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if value != plaintext {
		t.Errorf("expected %q, got %q", plaintext, value)
	}
}

func TestResolveCredential_FailsWithoutPassword(t *testing.T) {
	// Encrypt a test value
	password := "testpassword123"
	plaintext := "my-secret-token"
	encrypted, err := encryption.Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Set up Viper with encrypted value but NO password
	viper.Reset()
	viper.Set("api.mist.credentials.api_token", encrypted)
	defer viper.Reset()

	// Make sure password is not set
	os.Unsetenv("WIFIMGR_PASSWORD")

	// Resolve credential - should fail
	_, err = ResolveCredential("api.mist.credentials.api_token")
	if err == nil {
		t.Fatal("expected error when password not set for encrypted value")
	}
}

func TestResolveCredential_NotFound(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	// Clear any env vars
	os.Unsetenv("WIFIMGR_NONEXISTENT_CREDENTIAL")

	_, err := ResolveCredential("nonexistent.credential")
	if err == nil {
		t.Fatal("expected error for missing credential")
	}
}

func TestHasEncryptedCredentials(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	// No encrypted credentials
	viper.Set("api.mist.credentials.api_token", "plaintext-token")
	if HasEncryptedCredentials() {
		t.Error("expected false for plaintext credentials")
	}

	// Add encrypted credential
	viper.Set("api.mist.credentials.api_token", "enc:somethingencrypted")
	if !HasEncryptedCredentials() {
		t.Error("expected true for encrypted credentials")
	}
}

func TestIsCredentialAvailable(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	viper.Set("api.mist.credentials.api_token", "available-token")

	if !IsCredentialAvailable("api.mist.credentials.api_token") {
		t.Error("expected credential to be available")
	}

	if IsCredentialAvailable("nonexistent.credential") {
		t.Error("expected credential to not be available")
	}
}
