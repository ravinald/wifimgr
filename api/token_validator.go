package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/ravinald/wifimgr/internal/encryption"
)

// ClientTokenValidator implements the encryption.TokenValidator interface
// using the Mist API client for token validation
type ClientTokenValidator struct{}

// ValidateToken validates an API token by making a request to the Mist API
// This follows the requirements in encryption.txt:
// 1. GET /api/v1/self with token in Authorization header
// 2. Check for 200 response and proper content-type
// 3. Extract user and org information from response
func (v *ClientTokenValidator) ValidateToken(ctx context.Context, apiBaseURL, token string) (*encryption.TokenValidationResult, error) {
	// Create a temporary client with the token to validate
	tempClient := NewClientWithOptions(token, apiBaseURL, "", WithCacheTTL(0))

	// Call the API to validate the token using GET /api/v1/self
	resp, err := tempClient.ValidateAPIToken(ctx)
	if err != nil {
		// Format error message to be user-friendly
		var message string
		if errors.Is(err, ErrUnauthorized) {
			message = "Invalid token: authentication failed"
		} else {
			message = fmt.Sprintf("Token validation failed: %v", err)
		}

		return &encryption.TokenValidationResult{
			Valid:   false,
			Message: message,
		}, err
	}

	// Token is valid, extract user information
	result := &encryption.TokenValidationResult{
		Valid:    true,
		UserName: resp.Name,
	}

	// Extract organization information if available
	if len(resp.Privileges) > 0 {
		result.OrgID = resp.Privileges[0].OrgID
		result.OrgName = resp.Privileges[0].Name

		// Log which organization is being used
		fmt.Printf("Authenticated to organization: %s (%s)\n",
			result.OrgName, result.OrgID)
	} else {
		fmt.Println("Warning: No organization privileges found for this token")
	}

	return result, nil
}

// NewClientTokenValidator creates a new ClientTokenValidator
func NewClientTokenValidator() *ClientTokenValidator {
	return &ClientTokenValidator{}
}
