package api

import (
	"context"
)

// Token validation
// ============================================================================

// ValidateAPIToken validates the API token
func (m *MockClient) ValidateAPIToken(ctx context.Context) (*SelfResponse, error) {
	m.logRequest("GET", "/self", nil)

	// Mock API call with rate limiting
	if m.rateLimiter != nil {
		if err := m.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Return mock user info
	email := "mock-user@example.com"
	firstName := "Mock"
	lastName := "User"
	id := "mock-user-id"

	return &SelfResponse{
		Email:     &email,
		FirstName: &firstName,
		LastName:  &lastName,
		ID:        &id,
		Name:      "Mock User",
		Privileges: []Privilege{
			{
				OrgID: m.config.Organization,
				Name:  "Mock Organization",
				Role:  "admin",
				Scope: "org",
			},
		},
	}, nil
}

// GetAPIUserInfo retrieves information about the current API user
func (m *MockClient) GetAPIUserInfo(ctx context.Context) (*SelfResponse, error) {
	return m.ValidateAPIToken(ctx)
}
