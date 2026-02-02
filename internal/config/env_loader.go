package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ravinald/wifimgr/internal/logging"
	"github.com/ravinald/wifimgr/internal/xdg"
)

// LoadEnvFile loads environment variables from a file without using external dependencies
// This is more secure than using Viper for secrets as it:
// 1. Doesn't persist values in Viper's config tree
// 2. Only loads into environment variables
// 3. Clears values after use if needed
func LoadEnvFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open env file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logging.Warnf("Failed to close env file %s: %v", filename, err)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			logging.Warnf("Skipping malformed line %d in %s", lineNum, filename)
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes if present
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = value[1 : len(value)-1]
		}

		// Set environment variable - add WIFIMGR_ prefix if not present
		envKey := key
		if !strings.HasPrefix(key, "WIFIMGR_") {
			envKey = "WIFIMGR_" + key
		}

		if err := os.Setenv(envKey, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", envKey, err)
		}

		// Log that we loaded a key (but not its value for security)
		logging.Debugf("Loaded environment variable %s from %s", envKey, filename)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading env file: %w", err)
	}

	return nil
}

// ClearSensitiveEnvVars removes sensitive environment variables after use
// This is important for long-running processes to avoid memory inspection attacks
func ClearSensitiveEnvVars() {
	// Clear all multi-vendor credential variables (WIFIMGR_API_*_CREDENTIALS_*)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		if strings.HasPrefix(key, "WIFIMGR_API_") && strings.Contains(key, "_CREDENTIALS_") {
			if err := os.Unsetenv(key); err != nil {
				logging.Warnf("Failed to unset environment variable %s: %v", key, err)
			}
		}
	}

	logging.Debug("Cleared sensitive environment variables")
}

// SecureLoadEnvFile loads the env file and returns a cleanup function.
// If the file is not found at the specified path, it searches using XDG paths.
// Usage:
//
//	cleanup, err := SecureLoadEnvFile(".env.wifimgr")
//	if err != nil { ... }
//	defer cleanup()
func SecureLoadEnvFile(filename string) (func(), error) {
	// Try the specified filename first
	if _, err := os.Stat(filename); err == nil {
		if err := LoadEnvFile(filename); err != nil {
			return nil, err
		}
		return ClearSensitiveEnvVars, nil
	}

	// Try to find using XDG paths
	envPath := xdg.FindEnvFile()
	if envPath == "" {
		return nil, fmt.Errorf("env file not found: %s", filename)
	}

	if err := LoadEnvFile(envPath); err != nil {
		return nil, err
	}

	return ClearSensitiveEnvVars, nil
}
