package encryption

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/ravinald/wifimgr/internal/logging"
)

// PromptForAPIToken prompts the user to enter an API token
// Implements requirement from encryption.txt to not echo the input
var PromptForAPIToken = func() (string, error) {
	fmt.Print("Enter your Mist API token (input will not be displayed): ")
	// Use ReadPassword to hide the input
	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("failed to read API token: %w", err)
	}
	fmt.Println() // Add newline after hidden input

	// Convert to string and trim whitespace
	token := strings.TrimSpace(string(tokenBytes))

	// Basic validation
	if token == "" {
		return "", fmt.Errorf("API token cannot be empty")
	}

	return token, nil
}

// PromptForPassword prompts for a password with the given prompt message
func PromptForPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	// Use ReadPassword to hide the input
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // Add newline after hidden input

	// Convert to string and trim whitespace
	password := strings.TrimSpace(string(passwordBytes))

	return password, nil
}

// promptWithConfirmation prompts for a password with confirmation
func promptWithConfirmation(initialPrompt, confirmPrompt string) (string, error) {
	// First prompt
	password, err := PromptForPassword(initialPrompt)
	if err != nil {
		return "", err
	}

	// Basic validation
	if len(password) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}

	// Confirm password
	confirm, err := PromptForPassword(confirmPrompt)
	if err != nil {
		return "", err
	}

	// Check if passwords match
	if password != confirm {
		return "", fmt.Errorf("passwords do not match")
	}

	logging.Debug("Password confirmed successfully")
	return password, nil
}

// PromptForNewPassword prompts the user to enter a new password
// Implements requirement from encryption.txt #3 to verify password with double entry
var PromptForNewPassword = func() (string, error) {
	fmt.Println("Please set a password to encrypt your API token.")
	password, err := promptWithConfirmation(
		"Enter a password (input will not be displayed): ",
		"Confirm your password (input will not be displayed): ",
	)
	if err != nil {
		return "", err
	}

	fmt.Println("Password confirmed. Your API token will be encrypted and saved.")
	return password, nil
}

// PromptForExistingPassword prompts the user to enter an existing password
var PromptForExistingPassword = func() (string, error) {
	return PromptForPassword("Enter your password to decrypt the API token (input will not be displayed): ")
}

// IsTerminal determines if the given file descriptor is a terminal
func IsTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

// PasswordEnvVar is the environment variable name for the decryption password
const PasswordEnvVar = "WIFIMGR_PASSWORD"

// GetPasswordFromEnv returns the decryption password from environment variable if set.
// Returns empty string if not set.
func GetPasswordFromEnv() string {
	return os.Getenv(PasswordEnvVar)
}

// GetPasswordOrPrompt returns the decryption password from environment variable,
// or prompts the user interactively if not set.
// The prompt parameter is used for the interactive prompt message.
func GetPasswordOrPrompt(prompt string) (string, error) {
	// First check environment variable
	if password := GetPasswordFromEnv(); password != "" {
		logging.Debug("Using decryption password from environment variable")
		return password, nil
	}

	// Fall back to interactive prompt
	return PromptForPassword(prompt)
}
