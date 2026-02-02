package encryption

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
)

// We'll define a simple Config interface to avoid direct dependency on config package
type Config interface {
	GetAPIToken() string
	SetAPIToken(token string)
	SetKeyEncrypted(encrypted bool)
	GetAPIURL() string
	GetOrgID() string
	SetOrgID(id string)
	Save(configPath string) error
	SetCurrentToken(token string)
	GetCurrentToken() string
}

// TokenManager handles token validation, encryption, and decryption workflows
type TokenManager struct {
	Config    Config
	Validator TokenValidator
	APIURL    string
	Warnings  []string
	IO        TokenManagerIO
}

// NewTokenManager creates a new token manager
func NewTokenManager(cfg Config, validator TokenValidator) *TokenManager {
	return &TokenManager{
		Config:    cfg,
		Validator: validator,
		APIURL:    cfg.GetAPIURL(),
		Warnings:  []string{},
		IO:        NewDefaultIO(), // Default to standard IO
	}
}

// SetIO sets a custom IO implementation
func (tm *TokenManager) SetIO(io TokenManagerIO) {
	tm.IO = io
}

// InitializeToken handles the token validation and encryption workflow at application startup
func (tm *TokenManager) InitializeToken(ctx context.Context) error {
	token := tm.Config.GetAPIToken()

	// If no token is configured, try to read from .env.wifimgr
	if token == "" {
		// Check if we have a Config that implements ReadTokenFromEnvFile
		if envReader, ok := tm.Config.(interface {
			ReadTokenFromEnvFile() (string, error)
		}); ok {
			envToken, err := envReader.ReadTokenFromEnvFile()
			if err == nil && envToken != "" {
				logging.Debugf("Using API token from .env.wifimgr file")
				tm.IO.Println("Using API token from .env.wifimgr file")
				// Set the token for current session
				tm.updateClientToken(envToken)

				// Validate the token
				result, err := tm.Validator.ValidateToken(ctx, tm.APIURL, envToken)
				if err != nil || !result.Valid {
					tm.IO.Println("Error: The token from .env.wifimgr is invalid.")
					if result != nil && result.Message != "" {
						tm.IO.Println("Details: " + result.Message)
					}
					// Fall back to normal token handling workflow
					return tm.handleNoToken(ctx)
				}

				// Token is valid, print success message
				tm.printValidationSuccess(result)

				// Set org ID if not already set
				if tm.Config.GetOrgID() == "" && result.OrgID != "" {
					tm.Config.SetOrgID(result.OrgID)
					tm.IO.Println(fmt.Sprintf("Organization ID set to '%s' (%s)", result.OrgID, result.OrgName))
				}

				tm.IO.Println("API token from .env.wifimgr validated successfully.")
				return nil
			}
		}

		// If we couldn't get a token from .env.wifimgr, prompt for a new one
		return tm.handleNoToken(ctx)
	}

	// Check if token is already encrypted
	if IsEncrypted(token) {
		return tm.handleEncryptedToken(ctx, token)
	}

	// Handle plaintext token
	return tm.handlePlaintextToken(ctx, token)
}

// handleNoToken handles case when no token is configured
// This implements the workflow from encryption.txt #1
func (tm *TokenManager) handleNoToken(ctx context.Context) error {
	tm.IO.Println("No API token found in configuration.")

	// Prompt for a new token, hidden from display
	token, err := tm.promptForAPIToken()
	if err != nil {
		return fmt.Errorf("failed to get API token: %w", err)
	}

	// Validate the token against the API
	tm.IO.Println("Validating API token...")
	result, err := tm.Validator.ValidateToken(ctx, tm.APIURL, token)
	if err != nil {
		// Check if result is not nil before accessing its fields
		if result != nil && result.Message != "" {
			tm.IO.Println("Error: " + result.Message)
			tm.IO.Println("Details: " + result.Message)
		}

		tm.IO.Println("Would you like to try again? (y/n)")
		var response string
		_ = tm.IO.Scanln(&response)

		if strings.ToLower(response) == "y" {
			return tm.handleNoToken(ctx)
		} else {
			return errors.New("invalid API token provided")
		}
	}

	if !result.Valid {
		tm.IO.Println("Error: " + result.Message)
		tm.IO.Println("Would you like to try again? (y/n)")
		var response string
		_ = tm.IO.Scanln(&response)

		if strings.ToLower(response) == "y" {
			return tm.handleNoToken(ctx)
		}

		return errors.New("invalid API token provided")
	}

	tm.IO.Println("API token validated successfully.")

	// If org_id not set in config, use the one from the API response
	if tm.Config.GetOrgID() == "" && result.OrgID != "" {
		tm.IO.Println(fmt.Sprintf("Setting organization ID to '%s' (%s)",
			result.OrgID, result.OrgName))
	}

	// Token is valid, encrypt it
	return tm.encryptAndSaveToken(ctx, token, result)
}

// promptForAPIToken prompts for an API token using our I/O interface
func (tm *TokenManager) promptForAPIToken() (string, error) {
	// Prompt for token with hidden input
	token, err := tm.IO.PromptPassword("Enter your Mist API token (input will not be displayed): ")
	if err != nil {
		return "", err
	}

	// Basic validation
	token = strings.TrimSpace(token)
	if token == "" {
		return "", fmt.Errorf("API token cannot be empty")
	}

	return token, nil
}

// handleEncryptedToken handles case when token is already encrypted
// Implements requirements from encryption.txt #2 Existing Token
func (tm *TokenManager) handleEncryptedToken(ctx context.Context, encryptedToken string) error {
	tm.IO.Println("Encrypted API token found in configuration.")

	// Prompt for password
	password, err := tm.promptForExistingPassword()
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Try to decrypt the token
	plainToken, err := Decrypt(encryptedToken, password)
	if err != nil {
		tm.IO.Println("Error: Failed to decrypt the API token.")

		// Offer to try again with a different password
		tm.IO.Println("Would you like to try again with a different password? (y/n)")
		var response string
		_ = tm.IO.Scanln(&response)

		if strings.ToLower(response) == "y" {
			return tm.handleEncryptedToken(ctx, encryptedToken)
		}

		// Offer to enter a new token
		tm.IO.Println("Would you like to enter a new API token instead? (y/n)")
		_ = tm.IO.Scanln(&response)

		if strings.ToLower(response) == "y" {
			return tm.handleNoToken(ctx)
		}

		return errors.New("unable to decrypt API token")
	}

	// Now validate the decrypted token against the API
	tm.IO.Println("Validating API token...")
	result, err := tm.Validator.ValidateToken(ctx, tm.APIURL, plainToken)
	if err != nil || !result.Valid {
		tm.IO.Println("Error: The decrypted token is invalid.")
		if result != nil && result.Message != "" {
			tm.IO.Println("Details: " + result.Message)
		}

		// Offer to enter a new token
		tm.IO.Println("Would you like to enter a new API token? (y/n)")
		var response string
		_ = tm.IO.Scanln(&response)

		if strings.ToLower(response) == "y" {
			return tm.handleNoToken(ctx)
		}

		return errors.New("invalid API token")
	}

	// Token is valid, update the client with the decrypted token
	tm.updateClientToken(plainToken)

	// Print success message with user info
	tm.printValidationSuccess(result)

	tm.IO.Println("API token decrypted and validated successfully.")

	return nil
}

// promptForExistingPassword prompts for the password to decrypt the token
func (tm *TokenManager) promptForExistingPassword() (string, error) {
	return tm.IO.PromptPassword("Enter your password to decrypt the API token (input will not be displayed): ")
}

// handlePlaintextToken handles case when token is in plaintext
// This follows the requirements from encryption.txt for handling unencrypted tokens
func (tm *TokenManager) handlePlaintextToken(ctx context.Context, token string) error {
	tm.IO.Println("Unencrypted API token found in configuration. Validating...")

	// Validate the token
	result, err := tm.Validator.ValidateToken(ctx, tm.APIURL, token)
	if err != nil {
		tm.IO.Println("Error: Your API token is invalid.")
		if result != nil && result.Message != "" {
			tm.IO.Println("Details: " + result.Message)
		}

		// Offer to enter a new token
		tm.IO.Println("Would you like to enter a new API token? (y/n)")
		var response string
		_ = tm.IO.Scanln(&response)

		if strings.ToLower(response) == "y" {
			return tm.handleNoToken(ctx)
		}

		return errors.New("invalid API token provided")
	}

	if !result.Valid {
		tm.IO.Println("Error: " + result.Message)

		// Offer to enter a new token
		tm.IO.Println("Would you like to enter a new API token? (y/n)")
		var response string
		_ = tm.IO.Scanln(&response)

		if strings.ToLower(response) == "y" {
			return tm.handleNoToken(ctx)
		}

		return errors.New("invalid API token provided")
	}

	// Set org ID if not already set
	if tm.Config.GetOrgID() == "" && result.OrgID != "" {
		tm.Config.SetOrgID(result.OrgID)
		tm.IO.Println(fmt.Sprintf("Organization ID set to '%s' (%s)", result.OrgID, result.OrgName))
	}

	tm.IO.Println("API token validated successfully.")

	// Token is valid, encrypt it
	return tm.encryptAndSaveToken(ctx, token, result)
}

// encryptAndSaveToken encrypts and saves a valid token
// Implements requirements from encryption.txt #3 Password Encryption
func (tm *TokenManager) encryptAndSaveToken(_ context.Context, token string, result *TokenValidationResult) error {
	// Prompt for encryption password with confirmation
	tm.IO.Println("Please set a password to encrypt your API token.")
	password, err := tm.IO.PromptWithConfirm(
		"Enter a password (input will not be displayed): ",
		"Confirm your password (input will not be displayed): ",
	)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	tm.IO.Println("Password confirmed. Your API token will be encrypted and saved.")

	// Encrypt the token
	encryptedToken, err := Encrypt(token, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Update config with encrypted token
	tm.Config.SetAPIToken(encryptedToken)
	tm.Config.SetKeyEncrypted(true)

	// Set org ID if not already set
	if tm.Config.GetOrgID() == "" && result.OrgID != "" {
		tm.Config.SetOrgID(result.OrgID)
		tm.IO.Println(fmt.Sprintf("Organization ID set to '%s' (%s)", result.OrgID, result.OrgName))

		// Note which site name this corresponds to (not implemented fully yet,
		// would need to look up site name from cache)
		if result.OrgName != "" {
			tm.IO.Println(fmt.Sprintf("This corresponds to organization: %s", result.OrgName))
		}
	}

	// Update client with plaintext token for the current session
	tm.updateClientToken(token)

	// Save the updated config
	if err := tm.Config.Save(""); err != nil {
		return fmt.Errorf("failed to save config with encrypted token: %w", err)
	}

	// Print success message
	tm.printValidationSuccess(result)

	tm.IO.Println("API token has been encrypted and saved.")

	return nil
}

// updateClientToken updates the client with the plaintext token
func (tm *TokenManager) updateClientToken(token string) {
	// Store the decrypted token in the config for use by the client
	tm.Config.SetCurrentToken(token)
	logging.Debugf("API client updated with authenticated token")
}

// printValidationSuccess prints token validation success message
func (tm *TokenManager) printValidationSuccess(result *TokenValidationResult) {
	if result.UserName != "" {
		tm.IO.Println(fmt.Sprintf("API token is valid for user '%s'", result.UserName))
		if result.OrgName != "" {
			tm.IO.Println(fmt.Sprintf("Organization: %s (%s)", result.OrgName, result.OrgID))
		}
	} else {
		tm.IO.Println("API token is valid")
	}
}

// ForceUpdateToken forces the update of a token
func (tm *TokenManager) ForceUpdateToken(ctx context.Context) error {
	return tm.handleNoToken(ctx)
}
