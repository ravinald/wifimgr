package config

import (
	"os"
	"testing"
)

func TestReadTokenFromEnvFile(t *testing.T) {
	// Create a temporary test.env file for testing with multi-vendor format
	tempFile := "./test.env"
	content := "WIFIMGR_API_MIST_CREDENTIALS_KEY=testapitoken12345"
	err := os.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer func() { _ = os.Remove(tempFile) }() // Clean up temp file

	// Create a test config adapter
	cfg := &Config{
		API: API{},
	}
	ca := &ConfigAdapter{
		Config:     cfg,
		ConfigPath: "test-config.json",
	}

	// Test loading env file - in multi-vendor mode, this just loads env vars
	// but doesn't return a token directly (tokens are handled by InitializeMultiVendor)
	token, err := ca.ReadTokenFromEnvFileWithName(tempFile)
	if err != nil {
		t.Errorf("Error loading env file: %v", err)
	}

	// In multi-vendor mode, token is returned as empty - tokens are applied to API configs
	if token != "" {
		t.Errorf("Expected empty token (multi-vendor mode), got '%s'", token)
	}

	// Verify the env var was loaded into the environment
	envToken := os.Getenv("WIFIMGR_API_MIST_CREDENTIALS_KEY")
	if envToken != "testapitoken12345" {
		t.Errorf("Expected env var 'testapitoken12345', got '%s'", envToken)
	}

	// Clean up env var
	_ = os.Unsetenv("WIFIMGR_API_MIST_CREDENTIALS_KEY")
}
