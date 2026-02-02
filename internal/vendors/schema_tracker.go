package vendors

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"
)

// SchemaTracker tracks API response schemas and detects changes over time.
type SchemaTracker struct {
	snapshots map[string]*SchemaSnapshot // key: "vendor:deviceType"
	mu        sync.RWMutex
}

// NewSchemaTracker creates a new schema tracker.
func NewSchemaTracker() *SchemaTracker {
	return &SchemaTracker{
		snapshots: make(map[string]*SchemaSnapshot),
	}
}

// RecordAPIResponse records an API response and updates the schema snapshot.
// This should be called after each API call to accumulate schema information.
func (t *SchemaTracker) RecordAPIResponse(vendor, deviceType string, data map[string]any) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := makeSnapshotKey(vendor, deviceType)
	snapshot := t.snapshots[key]

	if snapshot == nil {
		snapshot = &SchemaSnapshot{
			Vendor:      vendor,
			DeviceType:  deviceType,
			Timestamp:   now(),
			Fields:      make(map[string]FieldSchema),
			SampleCount: 0,
		}
		t.snapshots[key] = snapshot
	}

	// Update sample count
	snapshot.SampleCount++

	// Analyze fields in the data
	analyzeFields(data, "", snapshot)

	// Update timestamp
	snapshot.Timestamp = now()
}

// GetSnapshot returns the current schema snapshot for a vendor and device type.
func (t *SchemaTracker) GetSnapshot(vendor, deviceType string) *SchemaSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	key := makeSnapshotKey(vendor, deviceType)
	return t.snapshots[key]
}

// CompareWithPrevious compares the current snapshot with a previous one and returns detected changes.
// The previous snapshot should be loaded from disk using LoadSnapshots.
func (t *SchemaTracker) CompareWithPrevious(vendor, deviceType string) []SchemaChange {
	t.mu.RLock()
	defer t.mu.RUnlock()

	key := makeSnapshotKey(vendor, deviceType)
	current := t.snapshots[key]

	if current == nil {
		return nil
	}

	// For comparison, we need a baseline snapshot to compare against.
	// This would typically be loaded from disk and stored separately.
	// For now, we return an empty slice. Full implementation would require
	// storing baseline snapshots separately.
	return []SchemaChange{}
}

// SaveSnapshots saves all snapshots to a JSON file.
func (t *SchemaTracker) SaveSnapshots(path string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	data, err := json.MarshalIndent(t.snapshots, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal snapshots: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write snapshots to %s: %w", path, err)
	}

	return nil
}

// LoadSnapshots loads snapshots from a JSON file.
func (t *SchemaTracker) LoadSnapshots(path string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet - not an error
			return nil
		}
		return fmt.Errorf("failed to read snapshots from %s: %w", path, err)
	}

	snapshots := make(map[string]*SchemaSnapshot)
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return fmt.Errorf("failed to unmarshal snapshots: %w", err)
	}

	t.snapshots = snapshots
	return nil
}

// makeSnapshotKey creates a key for the snapshots map.
func makeSnapshotKey(vendor, deviceType string) string {
	return vendor + ":" + deviceType
}

// analyzeFields recursively analyzes fields in a map and updates the snapshot.
func analyzeFields(data map[string]any, prefix string, snapshot *SchemaSnapshot) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		fieldType := getFieldType(value)

		// Get or create field schema
		field, exists := snapshot.Fields[fullKey]
		if !exists {
			field = FieldSchema{
				Type:      fieldType,
				Optional:  false,
				Frequency: 0.0,
			}
		}

		// Update frequency
		// frequency = (previous_count * previous_freq + 1) / new_count
		oldCount := float64(snapshot.SampleCount - 1)
		newCount := float64(snapshot.SampleCount)
		field.Frequency = (oldCount*field.Frequency + 1.0) / newCount

		// Mark as optional if frequency < 1.0
		field.Optional = field.Frequency < 1.0

		// Detect type changes
		if exists && field.Type != fieldType {
			// Type changed - this should be logged/reported
			field.Type = fieldType
		}

		snapshot.Fields[fullKey] = field

		// Recurse into nested objects
		if fieldType == "object" {
			if nestedMap, ok := value.(map[string]any); ok {
				analyzeFields(nestedMap, fullKey, snapshot)
			}
		}
	}
}

// getFieldType returns the type name for a value.
func getFieldType(value any) string {
	if value == nil {
		return "null"
	}

	switch v := value.(type) {
	case bool:
		return "bool"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "int"
	case float32, float64:
		return "float"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		// Use reflection for other types
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			return "array"
		}
		if t.Kind() == reflect.Map || t.Kind() == reflect.Struct {
			return "object"
		}
		return "unknown"
	}
}

// now returns the current time. Factored out for testing.
var now = time.Now
