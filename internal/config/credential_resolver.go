package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/ravinald/wifimgr/internal/encryption"
	"github.com/ravinald/wifimgr/internal/logging"
)

// ResolveCredential resolves a credential from environment variables or config.
// The configPath should be a dot-separated path like "api.mist.credentials.api_token".
//
// Resolution order:
//  1. Check for environment variable (WIFIMGR_ + path with dots as underscores)
//  2. Fall back to Viper config value
//  3. If value has "enc:" prefix, decrypt using WIFIMGR_PASSWORD
//
// Examples:
//
//	ResolveCredential("api.mist.credentials.api_token")
//	-> Checks WIFIMGR_API_MIST_CREDENTIALS_API_TOKEN, then config
func ResolveCredential(configPath string) (string, error) {
	// Convert path to env var name: api.mist.credentials.api_token -> WIFIMGR_API_MIST_CREDENTIALS_API_TOKEN
	envVar := "WIFIMGR_" + strings.ToUpper(strings.ReplaceAll(configPath, ".", "_"))

	// Check environment variable first (takes precedence when -e flag is used)
	if value := os.Getenv(envVar); value != "" {
		logging.Debugf("Resolved credential from env: %s", envVar)
		// Env values may also be encrypted
		if encryption.IsEncrypted(value) {
			return decryptValue(value, envVar)
		}
		return value, nil
	}

	// Fall back to config file
	value := viper.GetString(configPath)
	if value == "" {
		return "", fmt.Errorf("credential not found: %s (checked env: %s)", configPath, envVar)
	}

	logging.Debugf("Resolved credential from config: %s", configPath)

	// Check if encrypted
	if encryption.IsEncrypted(value) {
		return decryptValue(value, configPath)
	}

	return value, nil
}

// decryptValue decrypts an encrypted value using the password from environment.
func decryptValue(encryptedValue, source string) (string, error) {
	password := encryption.GetPasswordFromEnv()
	if password == "" {
		return "", fmt.Errorf("%s required to decrypt %s", encryption.PasswordEnvVar, source)
	}

	decrypted, err := encryption.Decrypt(encryptedValue, password)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt %s: %w", source, err)
	}

	logging.Debugf("Decrypted credential: %s", source)
	return decrypted, nil
}

// IsCredentialAvailable checks if a credential can be resolved (exists and can be decrypted).
func IsCredentialAvailable(configPath string) bool {
	_, err := ResolveCredential(configPath)
	return err == nil
}

// DecryptIfNeeded decrypts a value if it has the "enc:" prefix, otherwise returns it as-is.
// Used for decrypting template values like PSKs before sending to the API.
func DecryptIfNeeded(value, source string) (string, error) {
	if !encryption.IsEncrypted(value) {
		return value, nil
	}
	return decryptValue(value, source)
}

// HasEncryptedCredentials checks if any credentials in the config need decryption.
// This is useful for determining if WIFIMGR_PASSWORD will be required.
func HasEncryptedCredentials() bool {
	// Check API section for encrypted values
	apiSection := viper.GetStringMap("api")
	for _, apiValue := range apiSection {
		nested, ok := apiValue.(map[string]any)
		if !ok {
			continue
		}

		credsMap, ok := nested["credentials"].(map[string]any)
		if !ok {
			continue
		}

		for _, credValue := range credsMap {
			if str, ok := credValue.(string); ok {
				if encryption.IsEncrypted(str) {
					return true
				}
			}
		}
	}

	return false
}
