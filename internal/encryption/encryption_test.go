package encryption

import (
	"strings"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	testCases := []struct {
		name     string
		token    string
		password string
	}{
		{
			name:     "Standard API Token",
			token:    "abcdef1234567890abcdef1234567890",
			password: "securepassword",
		},
		{
			name:     "Empty Token",
			token:    "",
			password: "securepassword",
		},
		{
			name:     "Long Token",
			token:    strings.Repeat("x", 1024),
			password: "securepassword",
		},
		{
			name:     "Empty Password",
			token:    "abcdef1234567890abcdef1234567890",
			password: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := Encrypt(tc.token, tc.password)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Check prefix
			if !IsEncrypted(encrypted) {
				t.Errorf("Encrypted token doesn't have proper prefix: %s", encrypted)
			}

			// Decrypt with correct password
			decrypted, err := Decrypt(encrypted, tc.password)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != tc.token {
				t.Errorf("Decrypted token doesn't match original. Got %q, want %q", decrypted, tc.token)
			}

			// Decrypt with wrong password
			_, err = Decrypt(encrypted, tc.password+"wrong")
			if err == nil {
				t.Error("Decrypt should fail with wrong password")
			}
		})
	}
}

func TestDoubleEncryption(t *testing.T) {
	token := "abcdef1234567890"
	password := "password"

	// Encrypt once
	encrypted, err := Encrypt(token, password)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to encrypt again
	_, err = Encrypt(encrypted, password)
	if err == nil {
		t.Error("Should not be able to encrypt an already encrypted token")
	}
}

func TestIsEncrypted(t *testing.T) {
	testCases := []struct {
		token    string
		expected bool
	}{
		{Prefix + "abc", true},
		{"plaintoken", false},
		{"", false},
		{Prefix, true},
	}

	for i, tc := range testCases {
		if got := IsEncrypted(tc.token); got != tc.expected {
			t.Errorf("TestCase %d: IsEncrypted(%q) = %v, want %v", i, tc.token, got, tc.expected)
		}
	}
}

func TestDecryptNotEncrypted(t *testing.T) {
	token := "plaintoken"
	password := "password"

	_, err := Decrypt(token, password)
	if err != ErrNotEncrypted {
		t.Errorf("Expected ErrNotEncrypted, got: %v", err)
	}
}

func TestDecryptInvalidData(t *testing.T) {
	token := Prefix + "invalidbase64@#$"
	password := "password"

	_, err := Decrypt(token, password)
	if err == nil || !strings.Contains(err.Error(), ErrInvalidEncryptedData.Error()) {
		t.Errorf("Expected error containing %q, got: %v", ErrInvalidEncryptedData, err)
	}
}
