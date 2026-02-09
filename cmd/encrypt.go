/*
Copyright © 2025 Ravi Pina <ravi@pina.org>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ravinald/wifimgr/internal/cmdutils"
	"github.com/ravinald/wifimgr/internal/encryption"
)

var encryptCmd = &cobra.Command{
	Use:   "encrypt [psk]",
	Short: "Encrypt a secret for use in configuration files",
	Annotations: map[string]string{
		cmdutils.AnnotationNoInit: "true",
	},
	Long: `Interactively encrypt a secret value for use in configuration files.

This command prompts for both the secret and the encryption password with
terminal echo disabled, ensuring sensitive values are not visible on screen
or saved in shell history.

Arguments:
  psk    Optional. Validate the secret as a WPA2/WPA3 PSK (8-63 printable ASCII chars)

The output can be used in configuration files for:
  - WLAN PSK passwords (use 'psk' argument for validation)
  - RADIUS shared secrets
  - API tokens
  - Any other sensitive configuration values

Example usage:
  wifimgr encrypt        # Generic secret (non-empty)
  wifimgr encrypt psk    # Validate as WiFi PSK (8-63 chars, printable ASCII)

The encrypted output will have the 'enc:' prefix and can be pasted directly
into configuration files. When the application reads these values, it will
prompt for the decryption password.`,
	Args: cobra.MaximumNArgs(1),
	Run:  runEncrypt,
}

func init() {
	rootCmd.AddCommand(encryptCmd)
}

func runEncrypt(_ *cobra.Command, args []string) {
	// Determine if PSK validation is requested
	isPSK := len(args) > 0 && strings.EqualFold(args[0], "psk")

	// Step 1: Prompt for the secret value
	if isPSK {
		fmt.Println("Enter the WiFi PSK to encrypt.")
		fmt.Println("Requirements: 8-63 printable ASCII characters")
	} else {
		fmt.Println("Enter the secret value to encrypt.")
	}

	secret, err := promptSecretWithConfirmation(isPSK)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Prompt for the encryption password
	fmt.Println("\nEnter the encryption password.")
	fmt.Println("This password will be required to decrypt the value later.")
	password, err := encryption.PromptForPassword("Password (min 8 chars, input hidden): ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		os.Exit(1)
	}

	if len(password) < 8 {
		fmt.Fprintf(os.Stderr, "Error: password must be at least 8 characters\n")
		os.Exit(1)
	}

	// Confirm password
	confirm, err := encryption.PromptForPassword("Confirm password: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password confirmation: %v\n", err)
		os.Exit(1)
	}

	if password != confirm {
		fmt.Fprintf(os.Stderr, "Error: passwords do not match\n")
		os.Exit(1)
	}

	// Step 3: Encrypt the secret
	encrypted, err := encryption.Encrypt(secret, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encrypting secret: %v\n", err)
		os.Exit(1)
	}

	// Step 4: Output the encrypted value
	fmt.Println("\nEncrypted value:")
	fmt.Println(encrypted)
}

// promptSecretWithConfirmation prompts for a secret with confirmation
func promptSecretWithConfirmation(isPSK bool) (string, error) {
	promptText := "Secret (input hidden): "
	if isPSK {
		promptText = "PSK (input hidden): "
	}

	secret, err := encryption.PromptForPassword(promptText)
	if err != nil {
		return "", fmt.Errorf("failed to read secret: %w", err)
	}

	// Validate the secret
	if err := validateSecret(secret, isPSK); err != nil {
		return "", err
	}

	confirmText := "Confirm secret: "
	if isPSK {
		confirmText = "Confirm PSK: "
	}

	confirm, err := encryption.PromptForPassword(confirmText)
	if err != nil {
		return "", fmt.Errorf("failed to read confirmation: %w", err)
	}

	if secret != confirm {
		return "", fmt.Errorf("values do not match")
	}

	return secret, nil
}

// validateSecret validates a secret based on its type
func validateSecret(secret string, isPSK bool) error {
	if secret == "" {
		return fmt.Errorf("secret cannot be empty")
	}

	if isPSK {
		return validatePSK(secret)
	}

	return nil
}

// validatePSK validates a WiFi PSK according to IEEE 802.11i requirements
// PSK must be 8-63 printable ASCII characters (codes 32-126)
func validatePSK(psk string) error {
	length := len(psk)

	// Check length
	if length < 8 {
		return fmt.Errorf("PSK must be at least 8 characters (got %d)", length)
	}
	if length > 63 {
		return fmt.Errorf("PSK must be at most 63 characters (got %d)", length)
	}

	// Check for printable ASCII only (codes 32-126)
	for i, r := range psk {
		if r < 32 || r > 126 {
			return fmt.Errorf("PSK contains invalid character at position %d (must be printable ASCII)", i+1)
		}
	}

	return nil
}
