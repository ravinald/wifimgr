package filehash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ravinald/wifimgr/internal/logging"
)

// FileMetadata represents the metadata stored in the .meta file
//
// This structure maintains information about a file's contents, including
// its hash, path, and last modification time. It follows the same format
// used by the cache system for consistency across the codebase.
type FileMetadata struct {
	// Version of the metadata format
	Version int `json:"version"`
	// When the metadata was last updated
	LastUpdated string `json:"last_updated"`
	// File information
	File struct {
		// Absolute or relative path to the tracked file
		Path string `json:"path"`
		// SHA-256 hash of the file contents
		Hash string `json:"hash"`
		// Last modification time of the file in RFC3339 format
		LastModified string `json:"last_modified"`
		// Human-readable description of the file
		Description string `json:"description"`
		// Optional version of the file content
		Version interface{} `json:"version"`
	} `json:"file"`
}

// CalculateFileHash calculates the SHA-256 hash of any file
//
// This function reads the entire file and computes its SHA-256 hash. It returns
// the hash as a hex-encoded string, which is used to validate file integrity.
// Unlike the cache-specific hash function, this works with any file type.
//
// Parameters:
//   - filePath: The absolute path to the file to hash
//
// Returns:
//   - string: The hex-encoded SHA-256 hash of the file contents
//   - error: Any error encountered during file reading or hash calculation
func CalculateFileHash(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Create SHA-256 hasher
	hasher := sha256.New()

	// Copy file content to hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file for hashing: %w", err)
	}

	// Get hash and convert to hex string
	hashBytes := hasher.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}

// CreateMetadataFile creates or updates a metadata file for any file
//
// This function generates a metadata file (.meta) containing the file's hash,
// path, modification time, and description. The metadata file follows the same
// format used by the cache system for consistency.
//
// Parameters:
//   - filePath: The absolute path to the file for which to create metadata
//   - description: A human-readable description of the file
//
// Returns:
//   - error: Any error encountered during metadata creation or writing
func CreateMetadataFile(filePath, description string) error {
	// Calculate hash of the file
	hash, err := CalculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Get modification time of the file
	var lastModified string
	fileInfo, err := os.Stat(filePath)
	if err == nil {
		lastModified = fileInfo.ModTime().UTC().Format(time.RFC3339)
	} else {
		// If file doesn't exist yet, return error
		logging.Warnf("File %s not found when creating metadata: %v", filePath, err)
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Create metadata object
	metadata := &FileMetadata{
		Version:     1,
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
		File: struct {
			Path         string      `json:"path"`
			Hash         string      `json:"hash"`
			LastModified string      `json:"last_modified"`
			Description  string      `json:"description"`
			Version      interface{} `json:"version"`
		}{
			Path:         filePath,
			Hash:         hash,
			LastModified: lastModified,
			Description:  description,
			Version:      nil,
		},
	}

	// Create metadata file path
	metaPath := filePath + ".meta"

	// Ensure directory exists
	dir := filepath.Dir(metaPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for metadata file: %w", err)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp(dir, "meta-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary metadata file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write metadata to temp file
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(metadata); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write metadata to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temporary metadata file: %w", err)
	}

	// Rename temp file to actual metadata file (atomic operation)
	if err := os.Rename(tmpPath, metaPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to rename temporary metadata file: %w", err)
	}

	logging.Debugf("Created/updated metadata file at %s", metaPath)
	return nil
}

// VerifyFileIntegrity checks if a file's content hash matches what's in its metadata file
//
// This function calculates the current hash of a file and compares it to the hash
// stored in its metadata file. It returns true if the hashes match, indicating the
// file has not been modified since the metadata was last updated.
//
// Parameters:
//   - filePath: The absolute path to the file to verify
//
// Returns:
//   - bool: True if the file's hash matches the stored hash, false otherwise
//   - error: Any error encountered during verification
func VerifyFileIntegrity(filePath string) (bool, error) {
	// Check if file exists
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false, fmt.Errorf("file does not exist: %w", err)
	} else if err != nil {
		return false, fmt.Errorf("failed to check file: %w", err)
	}

	// Check if metadata file exists
	metaPath := filePath + ".meta"
	_, err = os.Stat(metaPath)
	if os.IsNotExist(err) {
		return false, fmt.Errorf("metadata file does not exist: %w", err)
	} else if err != nil {
		return false, fmt.Errorf("failed to check metadata file: %w", err)
	}

	// Load metadata file
	metaFile, err := os.Open(metaPath)
	if err != nil {
		return false, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer func() { _ = metaFile.Close() }()

	// Parse metadata
	var metadata FileMetadata
	if err := json.NewDecoder(metaFile).Decode(&metadata); err != nil {
		return false, fmt.Errorf("failed to parse metadata file: %w", err)
	}

	// Calculate current hash of file
	currentHash, err := CalculateFileHash(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to calculate current hash: %w", err)
	}

	logging.Debugf("Stored hash in metadata: %s", metadata.File.Hash)
	logging.Debugf("Calculated hash: %s", currentHash)

	// Compare calculated hash with stored hash
	return currentHash == metadata.File.Hash, nil
}

// SkipIntegrityChecks is a flag to indicate whether integrity checks should be skipped
// This is controlled by the API package's cache refresh flag
var SkipIntegrityChecks bool

// UpdateMetadataIfNeeded checks if a file's metadata needs updating and updates it if necessary
func UpdateMetadataIfNeeded(filePath, description string) error {
	metaPath := filePath + ".meta"

	// If SkipIntegrityChecks is set, always create a new metadata file
	// This is used during cache refresh operations
	if SkipIntegrityChecks {
		logging.Debugf("Skipping integrity check and updating metadata for %s", filePath)
		return CreateMetadataFile(filePath, description)
	}

	// Check if metadata file exists
	_, err := os.Stat(metaPath)
	if os.IsNotExist(err) {
		// Create new metadata file
		return CreateMetadataFile(filePath, description)
	}

	// Check if file integrity is still valid
	valid, err := VerifyFileIntegrity(filePath)
	if err != nil {
		logging.Warnf("Failed to verify file integrity: %v", err)
		// Create new metadata file if there was an error checking
		return CreateMetadataFile(filePath, description)
	}

	if !valid {
		logging.Debugf("File hash mismatch detected, updating metadata for %s", filePath)
		return CreateMetadataFile(filePath, description)
	}

	// Metadata is valid and up to date
	logging.Debugf("Metadata is valid and up to date for %s", filePath)
	return nil
}

// GetFileMetadata reads the metadata for a file
func GetFileMetadata(filePath string) (*FileMetadata, error) {
	metaPath := filePath + ".meta"

	// Check if metadata file exists
	_, err := os.Stat(metaPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("metadata file does not exist for %s", filePath)
	}

	// Open metadata file
	metaFile, err := os.Open(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer func() { _ = metaFile.Close() }()

	// Parse metadata
	var metadata FileMetadata
	if err := json.NewDecoder(metaFile).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata file: %w", err)
	}

	return &metadata, nil
}
