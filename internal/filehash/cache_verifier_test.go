package filehash

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCacheVerifierWithSchemaValidation tests the integration of schema validation with the cache verifier
func TestCacheVerifierWithSchemaValidation(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "cache-verifier-schema-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a schema directory
	schemaDir := filepath.Join(tempDir, "schemas")
	if err := os.MkdirAll(schemaDir, 0755); err != nil {
		t.Fatalf("Failed to create schema directory: %v", err)
	}

	// Create a test cache schema
	cacheSchema := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["version", "cache"],
		"properties": {
			"version": {
				"type": "number",
				"enum": [1.0]
			},
			"cache": {
				"type": "object",
				"required": ["sites", "inventory"],
				"properties": {
					"sites": {
						"type": "object"
					},
					"inventory": {
						"type": "object",
						"required": ["ap", "switch", "gateway"],
						"properties": {
							"ap": {
								"type": "object"
							},
							"switch": {
								"type": "object"
							},
							"gateway": {
								"type": "object"
							}
						}
					}
				}
			}
		}
	}`

	// Write schema to file
	schemaPath := filepath.Join(schemaDir, "cache-schema.json")
	if err := os.WriteFile(schemaPath, []byte(cacheSchema), 0644); err != nil {
		t.Fatalf("Failed to write schema file: %v", err)
	}

	// Create a valid cache file
	validCache := map[string]interface{}{
		"version": 1.0,
		"cache": map[string]interface{}{
			"sites": map[string]interface{}{},
			"inventory": map[string]interface{}{
				"ap":      map[string]interface{}{},
				"switch":  map[string]interface{}{},
				"gateway": map[string]interface{}{},
			},
		},
	}

	// Create an invalid cache file (missing gateway in inventory)
	invalidCache := map[string]interface{}{
		"version": 1.0,
		"cache": map[string]interface{}{
			"sites": map[string]interface{}{},
			"inventory": map[string]interface{}{
				"ap":     map[string]interface{}{},
				"switch": map[string]interface{}{},
				// Missing "gateway" field
			},
		},
	}

	// Create a valid cache file
	validCachePath := filepath.Join(tempDir, "valid-cache.json")
	validCacheBytes, _ := json.MarshalIndent(validCache, "", "  ")
	if err := os.WriteFile(validCachePath, validCacheBytes, 0644); err != nil {
		t.Fatalf("Failed to write valid cache file: %v", err)
	}

	// Create metadata for the valid cache file
	if err := CreateMetadataFile(validCachePath, "Test valid cache file"); err != nil {
		t.Fatalf("Failed to create metadata for valid cache: %v", err)
	}

	// Create an invalid cache file
	invalidCachePath := filepath.Join(tempDir, "invalid-cache.json")
	invalidCacheBytes, _ := json.MarshalIndent(invalidCache, "", "  ")
	if err := os.WriteFile(invalidCachePath, invalidCacheBytes, 0644); err != nil {
		t.Fatalf("Failed to write invalid cache file: %v", err)
	}

	// Create metadata for the invalid cache file
	if err := CreateMetadataFile(invalidCachePath, "Test invalid cache file"); err != nil {
		t.Fatalf("Failed to create metadata for invalid cache: %v", err)
	}

	// Mock function to count regenerate calls
	regenerateCalled := 0
	mockRegenerateFunc := func(ctx context.Context, orgID string) error {
		regenerateCalled++
		return nil
	}

	// Test with valid cache file and schema validation
	t.Run("ValidCache", func(t *testing.T) {
		// Create a cache verifier with schema directory
		verifier := NewCacheFileVerifier(context.Background(), "test-org", mockRegenerateFunc)
		verifier.SetSchemaDirectory(schemaDir)

		// Reset counter
		regenerateCalled = 0

		// Verify the valid cache - should pass both structure and schema validation
		status, err := verifier.VerifyIntegrity(validCachePath)
		if err != nil {
			t.Errorf("Verification failed unexpectedly: %v", err)
		}

		// Should be FileOK since the cache is valid
		if status != FileOK {
			t.Errorf("Expected FileOK status, got %v", status)
		}

		// Regenerate should not have been called
		if regenerateCalled > 0 {
			t.Errorf("Regenerate was called unexpectedly")
		}
	})

	// Test with invalid cache file that fails schema validation
	// We can't easily test user input for "n" in unit tests, so for this test
	// we'll just verify that the verifier tries to handle the corruption
	t.Run("InvalidCache", func(t *testing.T) {
		// Override os.Stdin for this test to simulate user input of "y" to regenerate
		oldStdin := os.Stdin
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.Write([]byte("y\n"))
		_ = w.Close()
		defer func() { os.Stdin = oldStdin }()

		// Create a cache verifier with schema directory
		verifier := NewCacheFileVerifier(context.Background(), "test-org", mockRegenerateFunc)
		verifier.SetSchemaDirectory(schemaDir)

		// Reset counter
		regenerateCalled = 0

		// Verify the invalid cache - should fail schema validation
		status, _ := verifier.VerifyIntegrity(invalidCachePath)

		// We should get FileRegenerated because we're simulating user saying "y" to regenerate
		if status != FileRegenerated {
			t.Errorf("Expected FileRegenerated status, got %v", status)
		}

		// Regenerate should have been called
		if regenerateCalled != 1 {
			t.Errorf("Expected regenerate to be called once, got %d", regenerateCalled)
		}
	})
}
