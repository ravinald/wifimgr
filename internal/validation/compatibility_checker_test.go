package validation

import (
	"testing"

	"github.com/ravinald/wifimgr/internal/config"
	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestNewCompatibilityChecker(t *testing.T) {
	tracker := vendors.NewSchemaTracker()
	checker := NewCompatibilityChecker(tracker, nil)

	if checker == nil {
		t.Fatal("NewCompatibilityChecker returned nil")
	}
	if checker.schemaTracker != tracker {
		t.Error("schema tracker not set correctly")
	}
}

func TestCompatibilityResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		issues   []CompatibilityIssue
		expected bool
	}{
		{
			name:     "no issues",
			issues:   []CompatibilityIssue{},
			expected: false,
		},
		{
			name: "only warnings",
			issues: []CompatibilityIssue{
				{Severity: "warning", Message: "deprecated field"},
			},
			expected: false,
		},
		{
			name: "has errors",
			issues: []CompatibilityIssue{
				{Severity: "error", Message: "invalid field"},
			},
			expected: true,
		},
		{
			name: "mixed errors and warnings",
			issues: []CompatibilityIssue{
				{Severity: "warning", Message: "deprecated field"},
				{Severity: "error", Message: "invalid field"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CompatibilityResult{
				Issues: tt.issues,
			}
			if got := result.HasErrors(); got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompatibilityResult_HasWarnings(t *testing.T) {
	tests := []struct {
		name     string
		issues   []CompatibilityIssue
		expected bool
	}{
		{
			name:     "no issues",
			issues:   []CompatibilityIssue{},
			expected: false,
		},
		{
			name: "has warnings",
			issues: []CompatibilityIssue{
				{Severity: "warning", Message: "deprecated field"},
			},
			expected: true,
		},
		{
			name: "only errors",
			issues: []CompatibilityIssue{
				{Severity: "error", Message: "invalid field"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CompatibilityResult{
				Issues: tt.issues,
			}
			if got := result.HasWarnings(); got != tt.expected {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCompatibilityResult_Summary(t *testing.T) {
	tests := []struct {
		name       string
		result     *CompatibilityResult
		wantPrefix string
	}{
		{
			name: "fully compatible",
			result: &CompatibilityResult{
				Compatible: true,
				Issues:     []CompatibilityIssue{},
			},
			wantPrefix: "Fully compatible",
		},
		{
			name: "compatible with warnings",
			result: &CompatibilityResult{
				Compatible: true,
				Issues: []CompatibilityIssue{
					{Severity: "warning", Message: "test"},
				},
			},
			wantPrefix: "Compatible with",
		},
		{
			name: "not compatible",
			result: &CompatibilityResult{
				Compatible: false,
				Issues: []CompatibilityIssue{
					{Severity: "error", Message: "test"},
				},
			},
			wantPrefix: "Not compatible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.result.Summary()
			if len(summary) < len(tt.wantPrefix) || summary[:len(tt.wantPrefix)] != tt.wantPrefix {
				t.Errorf("Summary() = %s, want prefix %s", summary, tt.wantPrefix)
			}
		})
	}
}

func TestCompatibilityResult_FilterBySeverity(t *testing.T) {
	result := &CompatibilityResult{
		Issues: []CompatibilityIssue{
			{Severity: "error", Field: "field1"},
			{Severity: "warning", Field: "field2"},
			{Severity: "error", Field: "field3"},
		},
	}

	errors := result.FilterBySeverity("error")
	if len(errors) != 2 {
		t.Errorf("FilterBySeverity(error) returned %d issues, want 2", len(errors))
	}

	warnings := result.FilterBySeverity("warning")
	if len(warnings) != 1 {
		t.Errorf("FilterBySeverity(warning) returned %d issues, want 1", len(warnings))
	}
}

func TestCompatibilityResult_GroupByField(t *testing.T) {
	result := &CompatibilityResult{
		Issues: []CompatibilityIssue{
			{Severity: "error", Field: "field1", Message: "msg1"},
			{Severity: "warning", Field: "field1", Message: "msg2"},
			{Severity: "error", Field: "field2", Message: "msg3"},
		},
	}

	grouped := result.GroupByField()
	if len(grouped) != 2 {
		t.Errorf("GroupByField() returned %d groups, want 2", len(grouped))
	}

	if len(grouped["field1"]) != 2 {
		t.Errorf("field1 group has %d issues, want 2", len(grouped["field1"]))
	}
	if len(grouped["field2"]) != 1 {
		t.Errorf("field2 group has %d issues, want 1", len(grouped["field2"]))
	}
}

func TestCheckDeprecatedFields(t *testing.T) {
	tracker := vendors.NewSchemaTracker()
	checker := NewCompatibilityChecker(tracker, nil)

	tests := []struct {
		name         string
		device       config.APConfig
		expectIssues bool
	}{
		{
			name: "no deprecated fields",
			device: config.APConfig{
				MAC: "aabbccddeeff",
			},
			expectIssues: false,
		},
		{
			name: "has legacy VlanID",
			device: config.APConfig{
				MAC:    "aabbccddeeff",
				VlanID: 100,
			},
			expectIssues: true,
		},
		{
			name: "has legacy Config",
			device: config.APConfig{
				MAC: "aabbccddeeff",
				Config: config.APHWConfig{
					LEDEnabled: true,
				},
			},
			expectIssues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := checker.checkDeprecatedFields(tt.device, "mist", "ap")

			if tt.expectIssues && len(issues) == 0 {
				t.Error("expected deprecated field issues but got none")
			}
			if !tt.expectIssues && len(issues) > 0 {
				t.Errorf("expected no deprecated field issues but got: %v", issues)
			}
		})
	}
}
