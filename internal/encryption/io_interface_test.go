package encryption

import (
	"context"
	"testing"
)

// MockIO implements TokenManagerIO for testing
type MockIO struct {
	PrintedMessages   []string
	PrintlnMessages   []string
	InputResponses    []string
	PasswordResponses []string
	PasswordIndex     int
	InputIndex        int
}

func NewMockIO() *MockIO {
	return &MockIO{
		PrintedMessages:   []string{},
		PrintlnMessages:   []string{},
		InputResponses:    []string{},
		PasswordResponses: []string{},
	}
}

// Print records a print message
func (io *MockIO) Print(msg string) {
	io.PrintedMessages = append(io.PrintedMessages, msg)
}

// Println records a println message
func (io *MockIO) Println(msg string) {
	io.PrintlnMessages = append(io.PrintlnMessages, msg)
}

// Scanln returns a predefined input response
func (io *MockIO) Scanln(value *string) error {
	if io.InputIndex < len(io.InputResponses) {
		*value = io.InputResponses[io.InputIndex]
		io.InputIndex++
		return nil
	}
	return nil
}

// PromptPassword returns a predefined password response
func (io *MockIO) PromptPassword(prompt string) (string, error) {
	io.PrintlnMessages = append(io.PrintlnMessages, prompt)
	if io.PasswordIndex < len(io.PasswordResponses) {
		response := io.PasswordResponses[io.PasswordIndex]
		io.PasswordIndex++
		return response, nil
	}
	return "", nil
}

// PromptWithConfirm returns a predefined password for confirmation
func (io *MockIO) PromptWithConfirm(initialPrompt, confirmPrompt string) (string, error) {
	io.PrintlnMessages = append(io.PrintlnMessages, initialPrompt)
	io.PrintlnMessages = append(io.PrintlnMessages, confirmPrompt)
	if io.PasswordIndex < len(io.PasswordResponses) {
		response := io.PasswordResponses[io.PasswordIndex]
		io.PasswordIndex++
		return response, nil
	}
	return "", nil
}

// TestDefaultIO tests the DefaultIO implementation
func TestDefaultIO(t *testing.T) {
	io := NewDefaultIO()

	// Verify the IO object is created
	if io == nil {
		t.Fatal("Failed to create DefaultIO")
	}

	// Note: We can't easily test the actual I/O operations
	// as they interact with os.Stdin and os.Stdout
	// We mainly verify the object structure here
}

// TestTokenManagerWithIO tests the token manager with a custom IO
func TestTokenManagerWithIO(t *testing.T) {
	// Create a mock config
	mockConfig := &mockConfig{
		apiToken:     "test-token",
		apiURL:       "https://test.com",
		orgID:        "test-org",
		saveError:    nil,
		currentToken: "",
		isTokenSaved: false,
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
	mockIO.InputResponses = []string{"y", "y"}
	mockIO.PasswordResponses = []string{"testpassword", "testpassword"}

	// Create token manager
	tm := NewTokenManager(mockConfig, mockValidator)
	tm.SetIO(mockIO)

	// Verify the IO is set
	if tm.IO == nil {
		t.Fatal("IO not set in token manager")
	}

	// Test using the custom IO
	// Note: We can't easily test the full token manager flow without
	// mocking more components, but we can verify that the token manager
	// uses our custom IO implementation
}

// mockConfig implements Config for testing
type mockConfig struct {
	apiToken     string
	apiURL       string
	orgID        string
	saveError    error
	currentToken string
	isTokenSaved bool
}

func (c *mockConfig) GetAPIToken() string {
	return c.apiToken
}

func (c *mockConfig) SetAPIToken(token string) {
	c.apiToken = token
	c.isTokenSaved = true
}

func (c *mockConfig) GetAPIURL() string {
	return c.apiURL
}

func (c *mockConfig) GetOrgID() string {
	return c.orgID
}

func (c *mockConfig) SetOrgID(id string) {
	c.orgID = id
}

func (c *mockConfig) Save(configPath string) error {
	return c.saveError
}

func (c *mockConfig) SetCurrentToken(token string) {
	c.currentToken = token
}

func (c *mockConfig) GetCurrentToken() string {
	return c.currentToken
}

// mockValidator implements TokenValidator for testing
type mockValidator struct {
	validateFunc func(ctx context.Context, apiURL, token string) (*TokenValidationResult, error)
}

func (v *mockValidator) ValidateToken(ctx context.Context, apiURL, token string) (*TokenValidationResult, error) {
	return v.validateFunc(ctx, apiURL, token)
}
