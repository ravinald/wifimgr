package encryption

import (
	"context"
	"errors"
	"fmt"
)

// TokenValidationResult represents the result of a token validation operation
type TokenValidationResult struct {
	Valid    bool
	Message  string
	UserName string
	OrgID    string
	OrgName  string
}

// TokenValidator defines an interface for token validation
type TokenValidator interface {
	ValidateToken(ctx context.Context, apiBaseURL, token string) (*TokenValidationResult, error)
}

// DefaultTokenValidator provides a default implementation (to be replaced in main.go)
type DefaultTokenValidator struct{}

// ValidateToken validates an API token against the Mist API
// This is a stub that will be implemented in main.go to avoid circular dependencies
func (v *DefaultTokenValidator) ValidateToken(ctx context.Context, apiBaseURL, token string) (*TokenValidationResult, error) {
	return &TokenValidationResult{
		Valid:   false,
		Message: "Token validation not implemented in this context",
	}, errors.New("token validation not implemented")
}

// ValidateAndEncryptToken validates a token and encrypts it if valid
func ValidateAndEncryptToken(ctx context.Context, validator TokenValidator, apiBaseURL, token, password string) (string, *TokenValidationResult, error) {
	// First validate the token
	result, err := validator.ValidateToken(ctx, apiBaseURL, token)
	if err != nil {
		return "", result, err
	}

	// If token is not valid, return early
	if !result.Valid {
		return "", result, errors.New("cannot encrypt an invalid token")
	}

	// Encrypt the token
	encryptedToken, err := Encrypt(token, password)
	if err != nil {
		return "", result, fmt.Errorf("failed to encrypt token: %w", err)
	}

	return encryptedToken, result, nil
}

// ValidateEncryptedToken validates an encrypted token by decrypting and checking it
func ValidateEncryptedToken(ctx context.Context, validator TokenValidator, apiBaseURL, encryptedToken, password string) (*TokenValidationResult, error) {
	// Check if token is encrypted
	if !IsEncrypted(encryptedToken) {
		return &TokenValidationResult{
			Valid:   false,
			Message: "Token is not encrypted",
		}, ErrNotEncrypted
	}

	// Try to decrypt the token
	token, err := Decrypt(encryptedToken, password)
	if err != nil {
		return &TokenValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to decrypt token: %v", err),
		}, err
	}

	// Validate the decrypted token
	return validator.ValidateToken(ctx, apiBaseURL, token)
}
