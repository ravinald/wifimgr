package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// FileMetadata represents metadata for file integrity checking
type FileMetadata struct {
	FileName     string    `json:"file_name"`
	FileType     string    `json:"file_type"`
	Size         int64     `json:"size"`
	Hash         string    `json:"hash"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"last_modified"`
	Version      int       `json:"version"`
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Sync to ensure data is written to disk
	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}

// CreateFileMetadata creates a metadata file for the given file
func CreateFileMetadata(filePath, metaPath, fileType string) error {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	// Calculate file hash
	hash, err := calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate hash for %s: %w", filePath, err)
	}

	// Create metadata
	metadata := FileMetadata{
		FileName:     fileInfo.Name(),
		FileType:     fileType,
		Size:         fileInfo.Size(),
		Hash:         hash,
		Created:      time.Now().UTC(),
		LastModified: fileInfo.ModTime().UTC(),
		Version:      1,
	}

	// Write metadata to file
	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	err = os.WriteFile(metaPath, metaData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write metadata file %s: %w", metaPath, err)
	}

	return nil
}

// VerifyFileIntegrity verifies the integrity of a file using its metadata
func VerifyFileIntegrity(filePath, metaPath string) error {
	// Check if both files exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("metadata file does not exist: %s", metaPath)
	}

	// Read metadata
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
	}

	var metadata FileMetadata
	err = json.Unmarshal(metaData, &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata file %s: %w", metaPath, err)
	}

	// Get current file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info for %s: %w", filePath, err)
	}

	// Check file size
	if fileInfo.Size() != metadata.Size {
		return fmt.Errorf("file size mismatch: expected %d, got %d", metadata.Size, fileInfo.Size())
	}

	// Calculate and verify hash
	currentHash, err := calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate current hash for %s: %w", filePath, err)
	}

	if currentHash != metadata.Hash {
		return fmt.Errorf("file hash mismatch: expected %s, got %s", metadata.Hash, currentHash)
	}

	return nil
}

// calculateFileHash calculates SHA256 hash of a file
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
