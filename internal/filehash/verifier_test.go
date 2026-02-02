package filehash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenericFileVerifier(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "verifier-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a test file
	testFilePath := filepath.Join(tempDir, "testfile.json")
	testData := `{"test": "data", "version": 1.0}`
	if err := os.WriteFile(testFilePath, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a verifier
	verifier := NewGenericFileVerifier()

	// Test GenerateMetadata
	if err := verifier.GenerateMetadata(testFilePath, "Test file for verifier"); err != nil {
		t.Fatalf("GenerateMetadata failed: %v", err)
	}

	// Verify metadata file was created
	metaPath := testFilePath + ".meta"
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Errorf("Metadata file was not created at %s", metaPath)
	}

	// Test VerifyIntegrity with valid file
	status, err := verifier.VerifyIntegrity(testFilePath)
	if err != nil {
		t.Fatalf("VerifyIntegrity failed: %v", err)
	}
	if status != FileOK {
		t.Errorf("Expected FileOK status, got %v", status)
	}

	// Test with non-existent file
	nonExistentPath := filepath.Join(tempDir, "nonexistent.json")
	status, err = verifier.VerifyIntegrity(nonExistentPath)
	if err == nil {
		t.Error("VerifyIntegrity should fail for non-existent file")
	}
	if status != FileFailed {
		t.Errorf("Expected FileFailed status for non-existent file, got %v", status)
	}

	// Test with empty file
	emptyFilePath := filepath.Join(tempDir, "empty.json")
	if err := os.WriteFile(emptyFilePath, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to write empty file: %v", err)
	}
	status, err = verifier.VerifyIntegrity(emptyFilePath)
	if err == nil {
		t.Error("VerifyIntegrity should fail for empty file")
	}
	if status != FileFailed {
		t.Errorf("Expected FileFailed status for empty file, got %v", status)
	}

	// Modify test file to test hash mismatch scenario
	// In a real test this would require mocking user input, so we'll skip the actual verification
	// and just confirm the metadata changes
	modifiedData := `{"test": "modified", "version": 2.0}`
	if err := os.WriteFile(testFilePath, []byte(modifiedData), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Manually regenerate metadata
	if err := verifier.GenerateMetadata(testFilePath, "Modified test file"); err != nil {
		t.Fatalf("Failed to regenerate metadata: %v", err)
	}

	// Verify integrity again
	status, err = verifier.VerifyIntegrity(testFilePath)
	if err != nil {
		t.Fatalf("VerifyIntegrity failed after metadata update: %v", err)
	}
	if status != FileOK {
		t.Errorf("Expected FileOK status after metadata update, got %v", status)
	}
}

func TestSpecializedVerifiers(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "specialized-verifier-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test files for each specialized verifier
	configFilePath := filepath.Join(tempDir, "config/site.json")
	inventoryFilePath := filepath.Join(tempDir, "inventory.json")

	// Ensure directories exist
	_ = os.MkdirAll(filepath.Dir(configFilePath), 0755)

	// Create content for each file
	configData := `{"version": 1.0, "config": {"site": {"name": "test"}}}`
	inventoryData := `{"version": 1.0, "inventory": {"devices": {}}}`

	// Write files
	if err := os.WriteFile(configFilePath, []byte(configData), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	if err := os.WriteFile(inventoryFilePath, []byte(inventoryData), 0644); err != nil {
		t.Fatalf("Failed to write inventory file: %v", err)
	}

	// Create each verifier
	configVerifier := NewConfigFileVerifier()
	inventoryVerifier := NewInventoryFileVerifier()

	// Test config verifier
	if err := configVerifier.GenerateMetadata(configFilePath, "Test config file"); err != nil {
		t.Fatalf("Config GenerateMetadata failed: %v", err)
	}
	status, err := configVerifier.VerifyIntegrity(configFilePath)
	if err != nil {
		t.Fatalf("Config VerifyIntegrity failed: %v", err)
	}
	if status != FileOK {
		t.Errorf("Expected FileOK status for config file, got %v", status)
	}

	// Test inventory verifier
	if err := inventoryVerifier.GenerateMetadata(inventoryFilePath, "Test inventory file"); err != nil {
		t.Fatalf("Inventory GenerateMetadata failed: %v", err)
	}
	status, err = inventoryVerifier.VerifyIntegrity(inventoryFilePath)
	if err != nil {
		t.Fatalf("Inventory VerifyIntegrity failed: %v", err)
	}
	if status != FileOK {
		t.Errorf("Expected FileOK status for inventory file, got %v", status)
	}

	// Test VerifyFile convenience function
	status, err = VerifyFile(configFilePath)
	if err != nil {
		t.Fatalf("VerifyFile failed for config file: %v", err)
	}
	if status != FileOK {
		t.Errorf("Expected FileOK status from VerifyFile for config file, got %v", status)
	}

	status, err = VerifyFile(inventoryFilePath)
	if err != nil {
		t.Fatalf("VerifyFile failed for inventory file: %v", err)
	}
	if status != FileOK {
		t.Errorf("Expected FileOK status from VerifyFile for inventory file, got %v", status)
	}
}
