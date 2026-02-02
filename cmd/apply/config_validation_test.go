package apply

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ravinald/wifimgr/internal/vendors"
)

func TestDisplayConfigWarnings(t *testing.T) {
	tests := []struct {
		name            string
		warnings        []error
		deviceMAC       string
		expectCritical  bool
		expectOutput    bool
		expectedStrings []string
	}{
		{
			name:           "no warnings",
			warnings:       []error{},
			deviceMAC:      "5c:5b:35:8e:4c:f9",
			expectCritical: false,
			expectOutput:   false,
		},
		{
			name: "field mapping error (critical)",
			warnings: []error{
				&vendors.FieldMappingError{
					Vendor:       "mist",
					DeviceMAC:    "5c:5b:35:8e:4c:f9",
					Field:        "radio_config.band_5.power",
					ExpectedType: "integer",
					ActualType:   "string",
					ActualValue:  "high",
				},
			},
			deviceMAC:      "5c:5b:35:8e:4c:f9",
			expectCritical: true,
			expectOutput:   true,
			expectedStrings: []string{
				"Critical Configuration Errors",
				"Field Type Mismatch",
				"radio_config.band_5.power",
				"Expected: integer",
			},
		},
		{
			name: "unexpected field warning (non-critical)",
			warnings: []error{
				&vendors.UnexpectedFieldWarning{
					Vendor:    "mist",
					DeviceMAC: "5c:5b:35:8e:4c:f9",
					Field:     "new_api_field",
					Value:     "some_value",
				},
			},
			deviceMAC:      "5c:5b:35:8e:4c:f9",
			expectCritical: false,
			expectOutput:   true,
			expectedStrings: []string{
				"Configuration Warnings",
				"Unexpected Field",
				"new_api_field",
			},
		},
		{
			name: "missing field warning (non-critical)",
			warnings: []error{
				&vendors.MissingFieldWarning{
					Vendor:    "mist",
					DeviceMAC: "5c:5b:35:8e:4c:f9",
					Field:     "deprecated_field",
				},
			},
			deviceMAC:      "5c:5b:35:8e:4c:f9",
			expectCritical: false,
			expectOutput:   true,
			expectedStrings: []string{
				"Configuration Warnings",
				"Missing Expected Field",
				"deprecated_field",
			},
		},
		{
			name: "mixed warnings",
			warnings: []error{
				&vendors.FieldMappingError{
					Vendor:       "mist",
					DeviceMAC:    "5c:5b:35:8e:4c:f9",
					Field:        "power",
					ExpectedType: "integer",
					ActualType:   "string",
					ActualValue:  "auto",
				},
				&vendors.UnexpectedFieldWarning{
					Vendor:    "mist",
					DeviceMAC: "5c:5b:35:8e:4c:f9",
					Field:     "new_field",
					Value:     123,
				},
			},
			deviceMAC:      "5c:5b:35:8e:4c:f9",
			expectCritical: true,
			expectOutput:   true,
			expectedStrings: []string{
				"Critical Configuration Errors",
				"Configuration Warnings",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call the function
			hasCritical := DisplayConfigWarnings(tt.warnings, tt.deviceMAC)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check critical flag
			if hasCritical != tt.expectCritical {
				t.Errorf("DisplayConfigWarnings() hasCritical = %v, want %v", hasCritical, tt.expectCritical)
			}

			// Check output
			hasOutput := len(output) > 0
			if hasOutput != tt.expectOutput {
				t.Errorf("DisplayConfigWarnings() produced output = %v, want %v", hasOutput, tt.expectOutput)
			}

			// Check expected strings
			for _, expected := range tt.expectedStrings {
				if !strings.Contains(output, expected) {
					t.Errorf("DisplayConfigWarnings() output missing expected string %q\nGot output:\n%s", expected, output)
				}
			}
		})
	}
}

func TestDisplayConfigValidationErrors(t *testing.T) {
	tests := []struct {
		name            string
		errors          []error
		deviceMAC       string
		vendorName      string
		expectAbort     bool
		expectedStrings []string
	}{
		{
			name:        "no errors",
			errors:      []error{},
			deviceMAC:   "5c:5b:35:8e:4c:f9",
			vendorName:  "mist",
			expectAbort: false,
		},
		{
			name: "validation error",
			errors: []error{
				&vendors.ConfigValidationError{
					Field:   "meraki",
					Message: "configuration contains 'meraki:' block but device targets Mist API",
				},
			},
			deviceMAC:   "5c:5b:35:8e:4c:f9",
			vendorName:  "mist",
			expectAbort: true,
			expectedStrings: []string{
				"Configuration Validation Errors",
				"mist vendor",
				"Field: meraki",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call the function
			shouldAbort := DisplayConfigValidationErrors(tt.errors, tt.deviceMAC, tt.vendorName)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			// Check abort flag
			if shouldAbort != tt.expectAbort {
				t.Errorf("DisplayConfigValidationErrors() shouldAbort = %v, want %v", shouldAbort, tt.expectAbort)
			}

			// Check expected strings
			for _, expected := range tt.expectedStrings {
				if !strings.Contains(output, expected) {
					t.Errorf("DisplayConfigValidationErrors() output missing expected string %q\nGot output:\n%s", expected, output)
				}
			}
		})
	}
}
