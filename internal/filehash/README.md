# File Hashing Utilities

This package provides utilities for generating, storing, and verifying file hashes to ensure data integrity.

## Overview

The `filehash` package implements a robust file integrity verification system using SHA-256 hashing. It allows tracking file modifications, validating file contents, and managing metadata for any file type.

## Key Components

### Core Hashing Functions

- `CalculateFileHash(filePath string) (string, error)`: Computes a SHA-256 hash for any file
- `CreateMetadataFile(filePath, description string) error`: Creates or updates a metadata file (.meta) with hash information
- `VerifyFileIntegrity(filePath string) (bool, error)`: Checks if a file's content matches its stored hash
- `UpdateMetadataIfNeeded(filePath, description string) error`: Updates metadata file if content has changed

### File Verification

- `FileVerifier` interface: Common interface for file verification implementations
- `GenericFileVerifier`: Default implementation for any file type
- `ConfigFileVerifier`: Specialized implementation for config files
- `InventoryFileVerifier`: Specialized implementation for inventory files
- `VerifyFile(filePath string) (FileVerificationStatus, error)`: Convenience function to verify any file

## Metadata Files

Each tracked file has a corresponding `.meta` file with the same name plus the `.meta` extension. For example:
- `inventory.json` → `inventory.json.meta`
- `us-sfo-lab.json` → `us-sfo-lab.json.meta`

The metadata file stores:
- File path
- SHA-256 hash of the content
- Last modification time
- File description
- Version information

## Usage Examples

### Calculating a File Hash

```go
hash, err := filehash.CalculateFileHash("/path/to/file.json")
if err != nil {
    log.Fatalf("Failed to calculate hash: %v", err)
}
fmt.Printf("File hash: %s\n", hash)
```

### Creating a Metadata File

```go
err := filehash.CreateMetadataFile("/path/to/file.json", "Configuration file")
if err != nil {
    log.Fatalf("Failed to create metadata: %v", err)
}
```

### Verifying File Integrity

```go
isValid, err := filehash.VerifyFileIntegrity("/path/to/file.json")
if err != nil {
    log.Fatalf("Error during verification: %v", err)
}
if !isValid {
    log.Println("File integrity check failed - hash mismatch detected")
} else {
    log.Println("File integrity verified successfully")
}
```

### Using File Verifiers

```go
verifier := filehash.NewConfigFileVerifier()
status, err := verifier.VerifyIntegrity("/path/to/config.json")
if err != nil {
    log.Fatalf("Error verifying file: %v", err)
}

switch status {
case filehash.FileOK:
    fmt.Println("File integrity verified")
case filehash.FileNew:
    fmt.Println("New metadata created")
case filehash.FileRegenerated:
    fmt.Println("Metadata regenerated")
case filehash.FileFailed:
    fmt.Println("File integrity check failed")
}
```

## Command-Line Tool

A command-line utility is available in `cmd/filehash/filehash.go` for generating and verifying file hashes:

```
# Generate metadata for all known files
./filehash-test --all --generate

# Verify integrity of specific file
./filehash-test --verify /path/to/file.json

# Show hash of a file
./filehash-test /path/to/file.json
```

## Supported File Types

This system is designed to work with any file type, but has specialized handling for:

1. Configuration files (e.g., `config/us-sfo-lab.json`)
2. Inventory files (e.g., `config/inventory.json`)
3. Cache files (e.g., `cache/id-cache.json`)