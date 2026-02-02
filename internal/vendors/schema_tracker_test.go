package vendors

import (
	"os"
	"testing"
	"time"
)

func TestNewSchemaTracker(t *testing.T) {
	tracker := NewSchemaTracker()
	if tracker == nil {
		t.Fatal("NewSchemaTracker returned nil")
	}
	if tracker.snapshots == nil {
		t.Error("snapshots map not initialized")
	}
}

func TestRecordAPIResponse(t *testing.T) {
	tracker := NewSchemaTracker()

	// Mock the now function for deterministic testing
	mockTime := time.Date(2025, 1, 26, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return mockTime }
	defer func() { now = time.Now }()

	data := map[string]any{
		"name":   "test-ap",
		"mac":    "aabbccddeeff",
		"status": "online",
	}

	tracker.RecordAPIResponse("mist", "ap", data)

	snapshot := tracker.GetSnapshot("mist", "ap")
	if snapshot == nil {
		t.Fatal("snapshot not created")
	}

	if snapshot.Vendor != "mist" {
		t.Errorf("expected vendor 'mist', got '%s'", snapshot.Vendor)
	}
	if snapshot.DeviceType != "ap" {
		t.Errorf("expected device type 'ap', got '%s'", snapshot.DeviceType)
	}
	if snapshot.SampleCount != 1 {
		t.Errorf("expected sample count 1, got %d", snapshot.SampleCount)
	}
	if len(snapshot.Fields) == 0 {
		t.Error("no fields recorded")
	}
}

func TestSaveAndLoadSnapshots(t *testing.T) {
	tracker := NewSchemaTracker()

	// Record some data
	data := map[string]any{
		"name": "test-ap",
		"mac":  "aabbccddeeff",
	}
	tracker.RecordAPIResponse("mist", "ap", data)

	// Save to temporary file
	tmpFile := "/tmp/schema_tracker_test.json"
	defer os.Remove(tmpFile)

	err := tracker.SaveSnapshots(tmpFile)
	if err != nil {
		t.Fatalf("SaveSnapshots failed: %v", err)
	}

	// Load into new tracker
	tracker2 := NewSchemaTracker()
	err = tracker2.LoadSnapshots(tmpFile)
	if err != nil {
		t.Fatalf("LoadSnapshots failed: %v", err)
	}

	snapshot := tracker2.GetSnapshot("mist", "ap")
	if snapshot == nil {
		t.Fatal("snapshot not loaded")
	}
	if snapshot.Vendor != "mist" {
		t.Errorf("expected vendor 'mist', got '%s'", snapshot.Vendor)
	}
}

func TestGetFieldType(t *testing.T) {
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
			result := getFieldType(tt.value)
			if result != tt.expected {
				t.Errorf("getFieldType(%v) = %s, want %s", tt.value, result, tt.expected)
			}
		})
	}
}
