package filehash

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"

	"github.com/ravinald/wifimgr/internal/logging"
)

// FileVerificationStatus represents the result of a file verification operation
//
// This type is used to indicate the outcome of file integrity verification operations.
// It follows the same pattern as CacheVerificationStatus for consistency.
type FileVerificationStatus int

const (
	// FileOK indicates the file is valid and hash matches
	FileOK FileVerificationStatus = iota

	// FileNew indicates a new file was created or metadata was regenerated
	// This is returned when a metadata file didn't exist and was created
	FileNew

	// FileRegenerated indicates the file metadata was regenerated
	// This is returned when file hash didn't match and metadata was updated
	FileRegenerated

	// FileFailed indicates the file integrity check failed but user chose to proceed
	// This is returned when verification failed and user elected not to regenerate
	FileFailed
)

// FileVerifier provides an interface for verifying different file types
//
// This interface allows for specialized verification behavior for different file types
// while maintaining a consistent API. Specialized verifiers can implement custom
// behavior for handling verification failures or generating metadata.
type FileVerifier interface {
	// VerifyIntegrity checks if the file content hash matches its metadata
	// and handles any necessary actions if it doesn't.
	VerifyIntegrity(filePath string) (FileVerificationStatus, error)

	// GenerateMetadata creates or updates metadata for a file
	// with the appropriate description and file-specific information.
	GenerateMetadata(filePath string, description string) error

	// HandleCorruption handles the case where file integrity check fails
	// by prompting the user for action or taking automatic remediation steps.
	HandleCorruption(filePath string, description string) (FileVerificationStatus, error)
}

// GenericFileVerifier provides a default implementation for file verification
type GenericFileVerifier struct {
	// Optional field to track if any file integrity was compromised
	integrityCompromised bool
}

// NewGenericFileVerifier creates a new generic file verifier
func NewGenericFileVerifier() *GenericFileVerifier {
	return &GenericFileVerifier{
		integrityCompromised: false,
	}
}

// IsIntegrityCompromised returns whether any file verification has failed
func (v *GenericFileVerifier) IsIntegrityCompromised() bool {
	return v.integrityCompromised
}

// VerifyIntegrity checks if the file exists and has a valid content hash
func (v *GenericFileVerifier) VerifyIntegrity(filePath string) (FileVerificationStatus, error) {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		logging.Warnf("File does not exist at %s", filePath)
		return FileFailed, fmt.Errorf("file does not exist: %w", err)
	} else if err != nil {
		return FileFailed, fmt.Errorf("failed to check file: %w", err)
	}

	// Check if file is empty
	if fileInfo.Size() == 0 {
		logging.Warnf("File is empty at %s", filePath)
		return FileFailed, fmt.Errorf("file is empty")
	}

	// Check if metadata file exists
	metaPath := filePath + ".meta"
	metadataInfo, err := os.Stat(metaPath)
	if os.IsNotExist(err) {
		logging.Warnf("Metadata file does not exist at %s", metaPath)
		// Create new metadata file
		if err := CreateMetadataFile(filePath, "File metadata"); err != nil {
			return FileFailed, fmt.Errorf("failed to create metadata file: %w", err)
		}
		return FileNew, nil
	} else if err != nil {
		return FileFailed, fmt.Errorf("failed to check metadata file: %w", err)
	}

	// Check if metadata file is empty
	if metadataInfo.Size() == 0 {
		logging.Warnf("Metadata file is empty at %s", metaPath)
		// Create new metadata file
		if err := CreateMetadataFile(filePath, "File metadata"); err != nil {
			return FileFailed, fmt.Errorf("failed to create metadata file: %w", err)
		}
		return FileNew, nil
	}

	// Verify file integrity
	valid, err := VerifyFileIntegrity(filePath)
	if err != nil {
		return FileFailed, fmt.Errorf("failed to verify file integrity: %w", err)
	}

	// If hash matches, file is valid
	if valid {
		logging.Debugf("File integrity verified successfully at %s", filePath)
		return FileOK, nil
	}

	// Hash mismatch detected, handle corruption
	logging.Warnf("File hash verification failed - hash mismatch detected for %s", filePath)
	return v.HandleCorruption(filePath, "File metadata")
}

// GenerateMetadata creates or updates metadata for a file
func (v *GenericFileVerifier) GenerateMetadata(filePath string, description string) error {
	return CreateMetadataFile(filePath, description)
}

// HandleCorruption handles the case where file integrity check fails
func (v *GenericFileVerifier) HandleCorruption(filePath string, description string) (FileVerificationStatus, error) {
	// Show file hash mismatch details
	fmt.Printf("\n%s WARNING: File integrity issue detected for %s!\n", color.RedString("!"), filePath)
	fmt.Printf("%s File hash doesn't match expected value in metadata file\n", color.YellowString(">"))
	fmt.Printf("\n%s Proceeding without regenerating the metadata may lead to using outdated or incorrect data\n", color.RedString("!"))

	// Prompt user for action
	fmt.Printf("\nWould you like to regenerate the metadata? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		logging.Errorf("Failed to read user input: %v", err)
		v.integrityCompromised = true
		return FileFailed, nil
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" || input == "y" || input == "yes" {
		// User chose to regenerate metadata
		logging.Infof("Regenerating metadata for %s...", filePath)

		// Create new metadata file
		if err := CreateMetadataFile(filePath, description); err != nil {
			v.integrityCompromised = true
			return FileFailed, fmt.Errorf("failed to create metadata file: %w", err)
		}

		logging.Infof("Metadata successfully regenerated for %s", filePath)
		return FileRegenerated, nil
	}

	// User chose not to regenerate metadata
	fmt.Printf("\n%s Proceeding with potentially compromised file data.\n", color.RedString("!"))
	fmt.Printf("%s CAUTION: Results may be inaccurate or incomplete.\n", color.RedString("!"))

	// Set global flag to indicate compromised integrity
	v.integrityCompromised = true
	return FileFailed, nil
}

// ConfigFileVerifier provides file verification specific to config files
type ConfigFileVerifier struct {
	GenericFileVerifier
}

// NewConfigFileVerifier creates a new config file verifier
func NewConfigFileVerifier() *ConfigFileVerifier {
	return &ConfigFileVerifier{
		GenericFileVerifier: *NewGenericFileVerifier(),
	}
}

// InventoryFileVerifier provides file verification specific to inventory files
type InventoryFileVerifier struct {
	GenericFileVerifier
}

// NewInventoryFileVerifier creates a new inventory file verifier
func NewInventoryFileVerifier() *InventoryFileVerifier {
	return &InventoryFileVerifier{
		GenericFileVerifier: *NewGenericFileVerifier(),
	}
}

// VerifyFile is a convenience function to verify any file with appropriate verifier
//
// This function automatically selects the appropriate FileVerifier implementation
// based on the file path and type. It's the recommended way to verify file integrity
// for most use cases, as it handles the selection logic for you.
//
// Parameters:
//   - filePath: The absolute path to the file to verify
//
// Returns:
//   - FileVerificationStatus: The status of the verification operation
//   - error: Any error encountered during verification
func VerifyFile(filePath string) (FileVerificationStatus, error) {
	// Choose appropriate verifier based on file path
	var verifier FileVerifier

	if strings.Contains(filePath, "inventory") {
		// Use specialized verifier for inventory files
		verifier = NewInventoryFileVerifier()
	} else if strings.HasSuffix(filePath, ".json") && strings.Contains(filePath, "config") {
		// Use specialized verifier for config files
		verifier = NewConfigFileVerifier()
	} else {
		// Default to generic verifier for any other file type
		verifier = NewGenericFileVerifier()
	}

	return verifier.VerifyIntegrity(filePath)
}
