package encryption

import (
	"context"
	"errors"
	"os"
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

func TestTokenManager_InitializeToken_EncryptedPath(t *testing.T) {
	// An encrypted token in config should route through handleEncryptedToken.
	encryptedToken, err := Encrypt("real-token", "real-password")
	if err != nil {
		t.Fatalf("Encrypt setup failed: %v", err)
	}

	mockConfig := &MockConfig{apiToken: encryptedToken}
	mockValidator := &MockTokenValidator{shouldSucceed: true}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"real-password"}

	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	if err := tm.InitializeToken(context.Background()); err != nil {
		t.Fatalf("InitializeToken() error = %v", err)
	}
	if mockConfig.currentToken != "real-token" {
		t.Errorf("InitializeToken() did not set current token to decrypted value, got %q", mockConfig.currentToken)
	}
}

func TestTokenManager_HandleNoToken_DeclineRetry(t *testing.T) {
	// Validator fails, user declines retry → ErrInvalidAPIToken sentinel.
	mockConfig := &MockConfig{}
	mockValidator := &MockTokenValidator{validationErr: errors.New("backend down")}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"bogus-token"}
	mockIO.InputResponses = []string{"n"}

	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	err := tm.handleNoToken(context.Background())
	if !errors.Is(err, ErrInvalidAPIToken) {
		t.Errorf("handleNoToken() err = %v, want errors.Is(ErrInvalidAPIToken)", err)
	}
}

func TestTokenManager_HandleEncryptedToken_DeclineEverything(t *testing.T) {
	// Decrypt fails (wrong password), user declines retry, declines new-token → ErrTokenDecryptFailed.
	encryptedToken, _ := Encrypt("real-token", "right-password")
	mockConfig := &MockConfig{apiToken: encryptedToken}
	mockValidator := &MockTokenValidator{shouldSucceed: true}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"wrong-password"}
	mockIO.InputResponses = []string{"n", "n"}

	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	err := tm.handleEncryptedToken(context.Background(), encryptedToken)
	if !errors.Is(err, ErrTokenDecryptFailed) {
		t.Errorf("handleEncryptedToken() err = %v, want errors.Is(ErrTokenDecryptFailed)", err)
	}
}

func TestTokenManager_HandleEncryptedToken_RetryThenSucceed(t *testing.T) {
	// Decrypt fails, user retries with the right password, decrypt succeeds.
	encryptedToken, _ := Encrypt("real-token", "right-password")
	mockConfig := &MockConfig{apiToken: encryptedToken}
	mockValidator := &MockTokenValidator{shouldSucceed: true}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"wrong-password", "right-password"}
	mockIO.InputResponses = []string{"y"}

	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	if err := tm.handleEncryptedToken(context.Background(), encryptedToken); err != nil {
		t.Fatalf("handleEncryptedToken() error = %v", err)
	}
	if mockConfig.currentToken != "real-token" {
		t.Errorf("did not set current token to decrypted value, got %q", mockConfig.currentToken)
	}
}

func TestTokenManager_HandleEncryptedToken_ValidatorFailsAfterDecrypt(t *testing.T) {
	// Decrypt succeeds but validator rejects the token, user declines new-token → ErrInvalidAPIToken.
	encryptedToken, _ := Encrypt("stale-token", "the-password")
	mockConfig := &MockConfig{apiToken: encryptedToken}
	mockValidator := &MockTokenValidator{validationErr: errors.New("token revoked")}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"the-password"}
	mockIO.InputResponses = []string{"n"}

	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	err := tm.handleEncryptedToken(context.Background(), encryptedToken)
	if !errors.Is(err, ErrInvalidAPIToken) {
		t.Errorf("handleEncryptedToken() err = %v, want errors.Is(ErrInvalidAPIToken)", err)
	}
}

func TestTokenManager_ForceUpdateToken(t *testing.T) {
	// ForceUpdateToken just delegates to handleNoToken; success path confirms the wiring.
	mockConfig := &MockConfig{apiToken: "ignored-existing-token"}
	mockValidator := &MockTokenValidator{shouldSucceed: true}
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"new-token", "new-password", "new-password"}

	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	if err := tm.ForceUpdateToken(context.Background()); err != nil {
		t.Fatalf("ForceUpdateToken() error = %v", err)
	}
	if !IsEncrypted(mockConfig.apiToken) {
		t.Errorf("ForceUpdateToken() did not encrypt token (missing enc: prefix)")
	}
	if mockConfig.currentToken != "new-token" {
		t.Errorf("ForceUpdateToken() did not set current token, got %q", mockConfig.currentToken)
	}
}

func TestTokenManager_PromptForExistingPassword_FromEnv(t *testing.T) {
	// When WIFIMGR_PASSWORD is set, promptForExistingPassword should return it
	// without touching the IO layer.
	const envPassword = "env-supplied-password"
	prev, hadPrev := os.LookupEnv(PasswordEnvVar)
	if err := os.Setenv(PasswordEnvVar, envPassword); err != nil {
		t.Fatalf("setenv failed: %v", err)
	}
	t.Cleanup(func() {
		if hadPrev {
			_ = os.Setenv(PasswordEnvVar, prev)
		} else {
			_ = os.Unsetenv(PasswordEnvVar)
		}
	})

	mockIO := NewMockIO()
	// Deliberately empty: if promptForExistingPassword falls through to IO, the
	// MockIO PromptPassword call will fail and surface the bug.

	tm := NewTokenManager(&MockConfig{}, &MockTokenValidator{shouldSucceed: true})
	tm.SetIO(mockIO)

	got, err := tm.promptForExistingPassword()
	if err != nil {
		t.Fatalf("promptForExistingPassword() error = %v", err)
	}
	if got != envPassword {
		t.Errorf("promptForExistingPassword() = %q, want %q", got, envPassword)
	}
}
