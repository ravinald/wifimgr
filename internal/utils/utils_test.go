package utils

import (
	"context"
	"testing"

	"github.com/ravinald/wifimgr/api"
	"github.com/ravinald/wifimgr/internal/config"
)

// createMockClientWithSite creates a mock client that returns a specific site by ID
func createMockClientWithSite(siteID api.UUID, siteName string) api.Client {
	client := api.NewMockClient(api.Config{}).(*api.MockClient)
	// Add the site to the mock client's internal storage
	site := api.Site{Id: &siteID, Name: &siteName}
	client.AddMockSite(site)
	return client
}

// createMockClientWithSiteByName creates a mock client that returns a specific site by name
func createMockClientWithSiteByName(name string, siteID api.UUID, siteName string) api.Client {
	client := api.NewMockClient(api.Config{}).(*api.MockClient)
	// Add the site to the mock client's internal storage
	site := api.Site{Id: &siteID, Name: &name}
	client.AddMockSite(site)
	return client
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short string (<=8 chars)",
			input:    "short",
			expected: "********",
		},
		{
			name:     "Exact 8 chars",
			input:    "12345678",
			expected: "********",
		},
		{
			name:     "Long string",
			input:    "a-very-long-api-token-that-should-be-masked",
			expected: "a-ve***************************************",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "********",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MaskString(tc.input)
			if result != tc.expected {
				t.Errorf("MaskString(%s): expected '%s', got '%s'", tc.input, tc.expected, result)
			}
		})
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid UUID",
			input:    "01234567-89ab-cdef-0123-456789abcdef",
			expected: true,
		},
		{
			name:     "Mixed case UUID",
			input:    "01234567-89AB-CDEF-0123-456789ABCDEF",
			expected: true,
		},
		{
			name:     "Test site ID",
			input:    "site-123456789",
			expected: true,
		},
		{
			name:     "Test AP ID",
			input:    "ap-123456789",
			expected: true,
		},
		{
			name:     "Not a UUID",
			input:    "not-a-uuid",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Almost UUID",
			input:    "01234567-89ab-cdef-0123-456789abcde", // Missing last char
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsUUID(tc.input)
			if result != tc.expected {
				t.Errorf("IsUUID(%s): expected %v, got %v", tc.input, tc.expected, result)
			}
		})
	}
}

func TestIsSiteCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Valid site code",
			input:    "US-NYC-001",
			expected: true,
		},
		{
			name:     "Another valid site code",
			input:    "UK-LON-MAIN",
			expected: true,
		},
		{
			name:     "Invalid site code (too short)",
			input:    "A-B-C",
			expected: false, // Doesn't match the required pattern (2 chars-3/4 chars-1-10 chars)
		},
		{
			name:     "Invalid site code (no dashes)",
			input:    "USNYC001",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsSiteCode(tc.input)
			if result != tc.expected {
				t.Errorf("IsSiteCode(%s): expected %v, got %v", tc.input, tc.expected, result)
			}
		})
	}
}

func TestResolveSiteID(t *testing.T) {
	// Mock UUID and name
	uuid := api.UUID("01234567-89ab-cdef-0123-456789abcdef")
	siteName := "Test Site"

	// Create test config
	cfg := &config.Config{
		API: config.API{
			Credentials: config.Credentials{
				OrgID: "test-org-id",
			},
			URL:          "https://api.mist.com/api/v1",
			RateLimit:    1000,
			ResultsLimit: 100,
		},
		Files: config.Files{
			Cache:     "",
			Inventory: "",
		},
	}

	tests := []struct {
		name          string
		identifier    string
		mockClient    api.Client
		expectedID    string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid UUID",
			identifier:  "01234567-89ab-cdef-0123-456789abcdef",
			mockClient:  createMockClientWithSite(uuid, siteName),
			expectedID:  "01234567-89ab-cdef-0123-456789abcdef",
			expectError: false,
		},
		{
			name:        "Valid site name",
			identifier:  "Test Site",
			mockClient:  createMockClientWithSiteByName("Test Site", uuid, siteName),
			expectedID:  "01234567-89ab-cdef-0123-456789abcdef",
			expectError: false,
		},
		{
			name:        "Valid site code",
			identifier:  "US-NYC-001",
			mockClient:  createMockClientWithSiteByName("US-NYC-001", uuid, siteName),
			expectedID:  "01234567-89ab-cdef-0123-456789abcdef",
			expectError: false,
		},
		{
			name:          "Non-existent site name",
			identifier:    "thissitenameisnonexistent",
			mockClient:    api.NewMockClient(api.Config{}),
			expectedID:    "",
			expectError:   true,
			errorContains: "no site found matching name",
		},
		{
			name:          "UUID not found",
			identifier:    "01234567-89ab-cdef-0123-456789abcdef",
			mockClient:    api.NewMockClient(api.Config{}),
			expectedID:    "",
			expectError:   true,
			errorContains: "site with UUID",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create test context
			ctx := context.Background()

			// Call ResolveSiteID
			id, err := ResolveSiteID(ctx, tc.mockClient, cfg, tc.identifier)

			// Check error status
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tc.errorContains != "" && !contains(err.Error(), tc.errorContains) {
					t.Errorf("Error does not contain expected string '%s', got: '%s'", tc.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}

			// Check returned ID
			if !tc.expectError && id != tc.expectedID {
				t.Errorf("Expected ID '%s', got '%s'", tc.expectedID, id)
			}
		})
	}
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr && len(s) > len(substr) && s[0:len(substr)] == substr
}
