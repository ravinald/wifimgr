package filehash

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateFileHash(t *testing.T) {
	// Create a temporary test file
	tempFile, err := os.CreateTemp("", "testhash-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tempFile.Name()) }()

	// Write some data to the file
	testData := "This is test data for hashing"
	if _, err := tempFile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	_ = tempFile.Close()

	// Calculate the hash of the file
	hash, err := CalculateFileHash(tempFile.Name())
	if err != nil {
		t.Fatalf("CalculateFileHash failed: %v", err)
	}

	// Verify hash is not empty
	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify hash is consistent
	hash2, err := CalculateFileHash(tempFile.Name())
	if err != nil {
		t.Fatalf("Second CalculateFileHash failed: %v", err)
	}
	if hash != hash2 {
		t.Errorf("Hash is not consistent: %s != %s", hash, hash2)
	}
}

func TestMetadataFileCreation(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "filehash-test-*")
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

	// Create metadata file
	description := "Test file for metadata"
	if err := CreateMetadataFile(testFilePath, description); err != nil {
		t.Fatalf("CreateMetadataFile failed: %v", err)
	}

	// Verify metadata file was created
	metaPath := testFilePath + ".meta"
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Errorf("Metadata file was not created at %s", metaPath)
	}

	// Verify file integrity
	valid, err := VerifyFileIntegrity(testFilePath)
	if err != nil {
		t.Fatalf("VerifyFileIntegrity failed: %v", err)
	}
	if !valid {
		t.Error("File integrity verification should be valid for newly created metadata")
	}

	// Modify the file
	modifiedData := `{"test": "modified data", "version": 1.1}`
	if err := os.WriteFile(testFilePath, []byte(modifiedData), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Verify integrity fails after modification
	valid, err = VerifyFileIntegrity(testFilePath)
	if err != nil {
		t.Fatalf("VerifyFileIntegrity failed after modification: %v", err)
	}
	if valid {
		t.Error("File integrity verification should fail after file modification")
	}

	// Update metadata
	if err := UpdateMetadataIfNeeded(testFilePath, description); err != nil {
		t.Fatalf("UpdateMetadataIfNeeded failed: %v", err)
	}

	// Verify integrity passes after metadata update
	valid, err = VerifyFileIntegrity(testFilePath)
	if err != nil {
		t.Fatalf("VerifyFileIntegrity failed after metadata update: %v", err)
	}
	if !valid {
		t.Error("File integrity verification should pass after metadata update")
	}
}

func TestGetFileMetadata(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "filehash-metadata-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a test file
	testFilePath := filepath.Join(tempDir, "metadata-test.json")
	testData := `{"test": "metadata", "timestamp": "2023-01-01"}`
	if err := os.WriteFile(testFilePath, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create metadata file
	description := "Test file for GetFileMetadata"
	if err := CreateMetadataFile(testFilePath, description); err != nil {
		t.Fatalf("CreateMetadataFile failed: %v", err)
	}

	// Get the metadata
	metadata, err := GetFileMetadata(testFilePath)
	if err != nil {
		t.Fatalf("GetFileMetadata failed: %v", err)
	}

	// Verify metadata content
	if metadata.File.Description != description {
		t.Errorf("Expected description %q, got %q", description, metadata.File.Description)
	}
	if metadata.File.Path != testFilePath {
		t.Errorf("Expected path %q, got %q", testFilePath, metadata.File.Path)
	}

	// Try to get metadata for non-existent file
	_, err = GetFileMetadata(filepath.Join(tempDir, "nonexistent.json"))
	if err == nil {
		t.Error("GetFileMetadata should fail for non-existent file")
	}
}
