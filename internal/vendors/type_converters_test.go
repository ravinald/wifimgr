package vendors

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func getTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	return logger
}

func TestSafeString(t *testing.T) {
	logger := getTestLogger()

	tests := []struct {
		name        string
		data        map[string]any
		field       string
		wantValue   string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid string",
			data:      map[string]any{"name": "AP-01"},
			field:     "name",
			wantValue: "AP-01",
			wantErr:   false,
		},
		{
			name:      "missing field",
			data:      map[string]any{},
			field:     "name",
			wantValue: "",
			wantErr:   false,
		},
		{
			name:        "wrong type - int",
			data:        map[string]any{"name": 123},
			field:       "name",
			wantValue:   "",
			wantErr:     true,
			errContains: "expected string but got int",
		},
		{
			name:        "wrong type - bool",
			data:        map[string]any{"name": true},
			field:       "name",
			wantValue:   "",
			wantErr:     true,
			errContains: "expected string but got bool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := SafeString(tt.data, tt.field, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if value != tt.wantValue {
				t.Errorf("SafeString() = %v, want %v", value, tt.wantValue)
			}
			if err != nil && tt.errContains != "" {
				if fme, ok := err.(*FieldMappingError); ok {
					if fme.ExpectedType != "string" {
						t.Errorf("FieldMappingError.ExpectedType = %v, want 'string'", fme.ExpectedType)
					}
				} else {
					t.Errorf("Expected FieldMappingError, got %T", err)
				}
			}
		})
	}
}

func TestSafeInt(t *testing.T) {
	logger := getTestLogger()

	tests := []struct {
		name      string
		data      map[string]any
		field     string
		wantValue *int
		wantErr   bool
	}{
		{
			name:      "valid int",
			data:      map[string]any{"power": 15},
			field:     "power",
			wantValue: intPtr(15),
			wantErr:   false,
		},
		{
			name:      "valid float64 (JSON unmarshaling)",
			data:      map[string]any{"power": float64(15)},
			field:     "power",
			wantValue: intPtr(15),
			wantErr:   false,
		},
		{
			name:      "missing field",
			data:      map[string]any{},
			field:     "power",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "wrong type - string",
			data:      map[string]any{"power": "high"},
			field:     "power",
			wantValue: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := SafeInt(tt.data, tt.field, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeInt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !intPtrEqual(value, tt.wantValue) {
				t.Errorf("SafeInt() = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestSafeBool(t *testing.T) {
	logger := getTestLogger()

	tests := []struct {
		name      string
		data      map[string]any
		field     string
		wantValue *bool
		wantErr   bool
	}{
		{
			name:      "valid bool - true",
			data:      map[string]any{"enabled": true},
			field:     "enabled",
			wantValue: boolPtr(true),
			wantErr:   false,
		},
		{
			name:      "valid bool - false",
			data:      map[string]any{"enabled": false},
			field:     "enabled",
			wantValue: boolPtr(false),
			wantErr:   false,
		},
		{
			name:      "missing field",
			data:      map[string]any{},
			field:     "enabled",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "wrong type - string",
			data:      map[string]any{"enabled": "true"},
			field:     "enabled",
			wantValue: nil,
			wantErr:   true,
		},
		{
			name:      "wrong type - int",
			data:      map[string]any{"enabled": 1},
			field:     "enabled",
			wantValue: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := SafeBool(tt.data, tt.field, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeBool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !boolPtrEqual(value, tt.wantValue) {
				t.Errorf("SafeBool() = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestSafeMap(t *testing.T) {
	logger := getTestLogger()

	tests := []struct {
		name      string
		data      map[string]any
		field     string
		wantValue map[string]any
		wantErr   bool
	}{
		{
			name: "valid map",
			data: map[string]any{
				"config": map[string]any{"key": "value"},
			},
			field:     "config",
			wantValue: map[string]any{"key": "value"},
			wantErr:   false,
		},
		{
			name:      "missing field",
			data:      map[string]any{},
			field:     "config",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "wrong type - string",
			data:      map[string]any{"config": "not-a-map"},
			field:     "config",
			wantValue: nil,
			wantErr:   true,
		},
		{
			name:      "wrong type - array",
			data:      map[string]any{"config": []string{"a", "b"}},
			field:     "config",
			wantValue: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := SafeMap(tt.data, tt.field, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !mapsEqual(value, tt.wantValue) {
				t.Errorf("SafeMap() = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

func TestSafeStringSlice(t *testing.T) {
	logger := getTestLogger()

	tests := []struct {
		name      string
		data      map[string]any
		field     string
		wantValue []string
		wantErr   bool
	}{
		{
			name: "valid string slice",
			data: map[string]any{
				"tags": []interface{}{"tag1", "tag2", "tag3"},
			},
			field:     "tags",
			wantValue: []string{"tag1", "tag2", "tag3"},
			wantErr:   false,
		},
		{
			name:      "empty slice",
			data:      map[string]any{"tags": []interface{}{}},
			field:     "tags",
			wantValue: []string{},
			wantErr:   false,
		},
		{
			name:      "missing field",
			data:      map[string]any{},
			field:     "tags",
			wantValue: nil,
			wantErr:   false,
		},
		{
			name:      "wrong type - string",
			data:      map[string]any{"tags": "not-a-slice"},
			field:     "tags",
			wantValue: nil,
			wantErr:   true,
		},
		{
			name: "wrong element type - int in slice",
			data: map[string]any{
				"tags": []interface{}{"tag1", 123, "tag3"},
			},
			field:     "tags",
			wantValue: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := SafeStringSlice(tt.data, tt.field, logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeStringSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !stringSliceEqual(value, tt.wantValue) {
				t.Errorf("SafeStringSlice() = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

// Helper functions for tests

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func mapsEqual(a, b map[string]any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || v != bv {
			return false
		}
	}
	return true
}

func stringSliceEqual(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
