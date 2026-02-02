package validation

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/config"
)

func TestNewConfigLinter(t *testing.T) {
	linter := NewConfigLinter(nil)
	if linter == nil {
		t.Fatal("NewConfigLinter returned nil")
	}
}

func TestValidateSyntax(t *testing.T) {
	linter := NewConfigLinter(nil)

	tests := []struct {
		name          string
		configMap     map[string]any
		expectIssues  bool
		issueContains string
	}{
		{
			name: "valid config",
			configMap: map[string]any{
				"name": "test-ap",
			},
			expectIssues: false,
		},
		{
			name:          "missing name",
			configMap:     map[string]any{},
			expectIssues:  true,
			issueContains: "Required field 'name'",
		},
		{
			name: "empty name",
			configMap: map[string]any{
				"name": "",
			},
			expectIssues:  true,
			issueContains: "Required field 'name'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := linter.validateSyntax(tt.configMap)

			if tt.expectIssues && len(issues) == 0 {
				t.Error("expected issues but got none")
			}
			if !tt.expectIssues && len(issues) > 0 {
				t.Errorf("expected no issues but got: %v", issues)
			}
			if tt.issueContains != "" && len(issues) > 0 {
				found := false
				for _, issue := range issues {
					if contains(issue.Message, tt.issueContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected issue containing '%s' but didn't find it", tt.issueContains)
				}
			}
		})
	}
}

func TestValidateRanges(t *testing.T) {
	linter := NewConfigLinter(nil)

	tests := []struct {
		name         string
		configMap    map[string]any
		deviceType   string
		expectIssues bool
	}{
		{
			name: "valid VLAN ID",
			configMap: map[string]any{
				"vlan_id": 100,
			},
			deviceType:   "ap",
			expectIssues: false,
		},
		{
			name: "VLAN ID too low",
			configMap: map[string]any{
				"vlan_id": 0,
			},
			deviceType:   "ap",
			expectIssues: true,
		},
		{
			name: "VLAN ID too high",
			configMap: map[string]any{
				"vlan_id": 5000,
			},
			deviceType:   "ap",
			expectIssues: true,
		},
		{
			name: "valid tx_power",
			configMap: map[string]any{
				"tx_power": 15,
			},
			deviceType:   "ap",
			expectIssues: false,
		},
		{
			name: "tx_power out of range",
			configMap: map[string]any{
				"tx_power": 25,
			},
			deviceType:   "ap",
			expectIssues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := linter.validateRanges(tt.configMap, tt.deviceType)

			if tt.expectIssues && len(issues) == 0 {
				t.Error("expected range validation issues but got none")
			}
			if !tt.expectIssues && len(issues) > 0 {
				t.Errorf("expected no range validation issues but got: %v", issues)
			}
		})
	}
}

func TestGetTargetVendor(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.SiteConfigObj
		expected string
	}{
		{
			name: "mist API",
			config: &config.SiteConfigObj{
				API: "mist-prod",
			},
			expected: "mist",
		},
		{
			name: "meraki API",
			config: &config.SiteConfigObj{
				API: "meraki-main",
			},
			expected: "meraki",
		},
		{
			name:     "no API specified",
			config:   &config.SiteConfigObj{},
			expected: "mist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTargetVendor(tt.config)
			if result != tt.expected {
				t.Errorf("getTargetVendor() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestGetTypeString(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{"string", "test", "string"},
		{"int", 42, "int"},
		{"bool", true, "bool"},
		{"float", 3.14, "float"},
		{"array", []any{1, 2, 3}, "array"},
		{"object", map[string]any{"key": "value"}, "object"},
		{"nil", nil, "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTypeString(tt.value)
			if result != tt.expected {
				t.Errorf("getTypeString(%v) = %s, want %s", tt.value, result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
