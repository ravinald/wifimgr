package encryption

import (
	"context"
	"testing"
)

// TestTokenManagerWithCustomIO tests token manager functions with custom IO
func TestTokenManagerWithCustomIO(t *testing.T) {
	// Create a mock config
	mockConfig := &mockConfig{
		apiToken:        "test-token",
		tokenEncrypted:  false,
		apiURL:          "https://test.com",
		orgID:           "test-org",
		saveError:       nil,
		currentToken:    "",
		isTokenSaved:    false,
		isEncryptedFlag: false,
	}

	// Create a mock validator
	mockValidator := &mockValidator{
		validateFunc: func(ctx context.Context, apiURL, token string) (*TokenValidationResult, error) {
			return &TokenValidationResult{
				Valid:    true,
				Message:  "Valid token",
				OrgID:    "test-org",
				OrgName:  "Test Org",
				UserName: "test-user",
			}, nil
		},
	}

	// Create a mock IO
	mockIO := NewMockIO()
	mockIO.InputResponses = []string{"y"}
	mockIO.PasswordResponses = []string{"testpassword123", "testpassword123"}

	// Create token manager
	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	// Test prompt functions through mock IO
	token, err := tm.promptForAPIToken()
	if err != nil {
		t.Fatalf("Failed to prompt for API token: %v", err)
	}
	if token != "testpassword123" {
		t.Fatalf("Expected token 'testpassword123', got '%s'", token)
	}

	// Test password functions through mock IO
	password, err := tm.promptForExistingPassword()
	if err != nil {
		t.Fatalf("Failed to prompt for password: %v", err)
	}
	if password != "testpassword123" {
		t.Fatalf("Expected password 'testpassword123', got '%s'", password)
	}

	// Verify IO interaction
	if len(mockIO.PrintlnMessages) < 2 {
		t.Fatal("Not enough printouts from token manager")
	}
}

// TestTokenManagerPrintValidationSuccess tests validation success printout
func TestTokenManagerPrintValidationSuccess(t *testing.T) {
	// Create a mock config
	mockConfig := &mockConfig{
		apiToken:       "test-token",
		tokenEncrypted: false,
		apiURL:         "https://test.com",
		orgID:          "test-org",
	}

	// Create a mock validator
	mockValidator := &mockValidator{
		validateFunc: func(ctx context.Context, apiURL, token string) (*TokenValidationResult, error) {
			return &TokenValidationResult{}, nil
		},
	}

	// Create a mock IO
	mockIO := NewMockIO()

	// Create token manager
	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	// Test validation success with username and org info
	result := &TokenValidationResult{
		Valid:    true,
		UserName: "test-user",
		OrgID:    "test-org",
		OrgName:  "Test Org",
	}
	tm.printValidationSuccess(result)

	// Verify the validation success message
	if len(mockIO.PrintlnMessages) < 2 {
		t.Fatal("Not enough validation success messages")
	}
	if mockIO.PrintlnMessages[0] != "API token is valid for user 'test-user'" {
		t.Fatalf("Unexpected validation message: %s", mockIO.PrintlnMessages[0])
	}

	// Test validation success without username
	mockIO = NewMockIO()
	tm.SetIO(mockIO)
	result.UserName = ""
	tm.printValidationSuccess(result)

	// Verify the validation success message without username
	if len(mockIO.PrintlnMessages) < 1 {
		t.Fatal("No validation success message")
	}
	if mockIO.PrintlnMessages[0] != "API token is valid" {
		t.Fatalf("Unexpected validation message: %s", mockIO.PrintlnMessages[0])
	}
}

// TestTokenManagerWithContext tests token manager with context
func TestTokenManagerWithContext(t *testing.T) {
	// Create a mock config
	mockConfig := &mockConfig{
		apiToken:       "",
		tokenEncrypted: false,
		apiURL:         "https://test.com",
		orgID:          "",
	}

	// Create a mock validator
	mockValidator := &mockValidator{
		validateFunc: func(ctx context.Context, apiURL, token string) (*TokenValidationResult, error) {
			// Check that we received a valid context
			if ctx == nil {
				t.Fatal("Nil context passed to validator")
			}
			return &TokenValidationResult{
				Valid:    true,
				Message:  "Valid token",
				OrgID:    "test-org",
				OrgName:  "Test Org",
				UserName: "test-user",
			}, nil
		},
	}

	// Create a mock IO
	mockIO := NewMockIO()
	mockIO.PasswordResponses = []string{"test-token", "test-password", "test-password"}
	mockIO.InputResponses = []string{"y"}

	// Create token manager
	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	// Create a test context
	ctx := context.Background()

	// Test initialization with context
	// Note: We're only testing that the context is passed to validator
	// not the complete flow which would require more mocking
	_ = tm.InitializeToken(ctx)
}
