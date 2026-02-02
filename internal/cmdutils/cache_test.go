package cmdutils

import (
	"testing"
)

func TestGetNestedValue(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		path     string
		want     interface{}
		wantOK   bool
	}{
		{
			name:   "nil data returns false",
			data:   nil,
			path:   "foo",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "empty path returns false",
			data:   map[string]interface{}{"foo": "bar"},
			path:   "",
			want:   nil,
			wantOK: false,
		},
		{
			name:   "simple field access",
			data:   map[string]interface{}{"name": "test-ap"},
			path:   "name",
			want:   "test-ap",
			wantOK: true,
		},
		{
			name: "nested field access",
			data: map[string]interface{}{
				"radio_config": map[string]interface{}{
					"band_5": map[string]interface{}{
						"channel": 36,
						"power":   17,
					},
				},
			},
			path:   "radio_config.band_5.channel",
			want:   36,
			wantOK: true,
		},
		{
			name: "deeply nested field",
			data: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": map[string]interface{}{
							"d": "deep-value",
						},
					},
				},
			},
			path:   "a.b.c.d",
			want:   "deep-value",
			wantOK: true,
		},
		{
			name: "missing intermediate key",
			data: map[string]interface{}{
				"radio_config": map[string]interface{}{
					"band_24": map[string]interface{}{
						"channel": 6,
					},
				},
			},
			path:   "radio_config.band_5.channel",
			want:   nil,
			wantOK: false,
		},
		{
			name: "path through non-map value",
			data: map[string]interface{}{
				"name": "test-ap",
			},
			path:   "name.foo",
			want:   nil,
			wantOK: false,
		},
		{
			name: "numeric value at leaf",
			data: map[string]interface{}{
				"ip_config": map[string]interface{}{
					"vlan_id": float64(100),
				},
			},
			path:   "ip_config.vlan_id",
			want:   float64(100),
			wantOK: true,
		},
		{
			name: "boolean value at leaf",
			data: map[string]interface{}{
				"led": map[string]interface{}{
					"enabled": true,
				},
			},
			path:   "led.enabled",
			want:   true,
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := GetNestedValue(tt.data, tt.path)
			if ok != tt.wantOK {
				t.Errorf("GetNestedValue() ok = %v, wantOK %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("GetNestedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatNestedValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  string
	}{
		{
			name:  "nil returns empty string",
			value: nil,
			want:  "",
		},
		{
			name:  "string passthrough",
			value: "hello",
			want:  "hello",
		},
		{
			name:  "integer from float64",
			value: float64(42),
			want:  "42",
		},
		{
			name:  "decimal float64",
			value: float64(3.14),
			want:  "3.14",
		},
		{
			name:  "boolean true",
			value: true,
			want:  "true",
		},
		{
			name:  "boolean false",
			value: false,
			want:  "false",
		},
		{
			name:  "array of strings",
			value: []interface{}{"a", "b", "c"},
			want:  "a, b, c",
		},
		{
			name:  "array of numbers",
			value: []interface{}{float64(1), float64(2), float64(3)},
			want:  "1, 2, 3",
		},
		{
			name:  "nested map summary",
			value: map[string]interface{}{"a": 1, "b": 2},
			want:  "{2 fields}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatNestedValue(tt.value)
			if got != tt.want {
				t.Errorf("FormatNestedValue() = %q, want %q", got, tt.want)
			}
		})
	}
}
