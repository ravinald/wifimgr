package encryption

import (
	"context"
	"errors"
	"testing"
)

// MockConfig implements the Config interface for testing
type MockConfig struct {
	apiToken     string
	orgID        string
	currentToken string
	saveErr      error
}

func (m *MockConfig) GetAPIToken() string          { return m.apiToken }
func (m *MockConfig) SetAPIToken(token string)     { m.apiToken = token }
func (m *MockConfig) GetAPIURL() string            { return "https://test-api.com" }
func (m *MockConfig) GetOrgID() string             { return m.orgID }
func (m *MockConfig) SetOrgID(id string)           { m.orgID = id }
func (m *MockConfig) Save(configPath string) error { return m.saveErr }
func (m *MockConfig) SetCurrentToken(token string) { m.currentToken = token }
func (m *MockConfig) GetCurrentToken() string      { return m.currentToken }

// MockTokenValidator implements the TokenValidator interface for testing
type MockTokenValidator struct {
	shouldSucceed bool
	validationErr error
	result        *TokenValidationResult
}

func (v *MockTokenValidator) ValidateToken(_ context.Context, _, _ string) (*TokenValidationResult, error) {
	if v.validationErr != nil {
		return nil, v.validationErr
	}

	if v.result != nil {
		return v.result, nil
	}

	if v.shouldSucceed {
		return &TokenValidationResult{
			Valid:    true,
			UserName: "Test User",
			OrgID:    "org123",
			OrgName:  "Test Org",
		}, nil
	}

	return &TokenValidationResult{
		Valid:   false,
		Message: "Invalid token",
	}, errors.New("invalid token")
}

func TestTokenManager_HandleNoToken(t *testing.T) {
	// Create mock config, validator, and IO
	mockConfig := &MockConfig{}
	mockValidator := &MockTokenValidator{shouldSucceed: true}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"test-token", "test-password", "test-password"}
	mockIO.InputResponses = []string{"y"}

	// Create token manager with mock IO
	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	// Test handleNoToken
	err := tm.handleNoToken(context.Background())

	// Check results
	if err != nil {
		t.Errorf("handleNoToken() error = %v, want nil", err)
	}

	// Token should be encrypted (has enc: prefix) and saved
	if !IsEncrypted(mockConfig.apiToken) {
		t.Errorf("handleNoToken() did not encrypt token (missing enc: prefix)")
	}

	if mockConfig.currentToken != "test-token" {
		t.Errorf("handleNoToken() did not set current token")
	}

	if mockConfig.orgID != "org123" {
		t.Errorf("handleNoToken() did not set org ID")
	}
}

func TestTokenManager_HandleEncryptedToken(t *testing.T) {
	// Create encrypted token for testing
	encryptedToken, _ := Encrypt("test-token", "test-password")

	// Create mock config, validator, and IO
	mockConfig := &MockConfig{apiToken: encryptedToken}
	mockValidator := &MockTokenValidator{shouldSucceed: true}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"test-password"}

	// Create token manager
	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	// Test handleEncryptedToken
	err := tm.handleEncryptedToken(context.Background(), encryptedToken)

	// Check results
	if err != nil {
		t.Errorf("handleEncryptedToken() error = %v, want nil", err)
	}

	// Token should be decrypted for current session
	if mockConfig.currentToken != "test-token" {
		t.Errorf("handleEncryptedToken() did not set current token")
	}
}

func TestTokenManager_HandlePlaintextToken(t *testing.T) {
	// Create mock config, validator, and IO
	mockConfig := &MockConfig{apiToken: "test-token"}
	mockValidator := &MockTokenValidator{shouldSucceed: true}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"test-password", "test-password"}

	// Create token manager with mock IO
	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	// Test handlePlaintextToken
	err := tm.handlePlaintextToken(context.Background(), "test-token")

	// Check results
	if err != nil {
		t.Errorf("handlePlaintextToken() error = %v, want nil", err)
	}

	// Token should be encrypted (has enc: prefix) and saved
	if !IsEncrypted(mockConfig.apiToken) {
		t.Errorf("handlePlaintextToken() did not encrypt token (missing enc: prefix)")
	}

	if mockConfig.currentToken != "test-token" {
		t.Errorf("handlePlaintextToken() did not set current token")
	}
}

func TestTokenManager_InitializeToken(t *testing.T) {
	tests := []struct {
		name              string
		token             string
		shouldEncrypt     bool
		validatorSucceeds bool
		wantErr           bool
	}{
		{
			name:              "No token",
			token:             "",
			shouldEncrypt:     true,
			validatorSucceeds: true,
			wantErr:           false,
		},
		{
			name:              "Plaintext token",
			token:             "test-token",
			shouldEncrypt:     true,
			validatorSucceeds: true,
			wantErr:           false,
		},
		{
			name:              "Invalid plaintext token",
			token:             "invalid-token",
			shouldEncrypt:     true,
			validatorSucceeds: false,
			wantErr:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock config, validator, and IO
			mockConfig := &MockConfig{apiToken: tt.token}
			mockValidator := &MockTokenValidator{shouldSucceed: tt.validatorSucceeds}
			mockIO := NewMockIO()
			mockIO.PasswordResponses = []string{"test-token", "test-password", "test-password"}
			mockIO.InputResponses = []string{"y"}

			// Create token manager with mock IO
			tm := NewTokenManager(mockConfig, mockValidator)
			tm.SetIO(mockIO)

			// Test InitializeToken
			err := tm.InitializeToken(context.Background())

			// Check results
			if (err != nil) != tt.wantErr {
				t.Errorf("InitializeToken() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If no error and we should encrypt, check token was processed correctly
			if err == nil && tt.shouldEncrypt && tt.validatorSucceeds {
				// Token should be encrypted (has enc: prefix)
				if !IsEncrypted(mockConfig.apiToken) {
					t.Errorf("InitializeToken() did not encrypt token (missing enc: prefix)")
				}

				// Token should be set
				if tt.token == "" {
					if mockConfig.currentToken != "test-token" {
						t.Errorf("InitializeToken() did not set current token")
					}
				} else {
					if mockConfig.currentToken != tt.token {
						t.Errorf("InitializeToken() did not set current token")
					}
				}
			}
		})
	}
}
