package vendors

import "time"

// SchemaChange represents a detected change in API schema.
type SchemaChange struct {
	ChangeType string // "added", "removed", "type_changed", "frequency_changed"
	Field      string
	OldType    string  // for type_changed
	NewType    string  // for type_changed
	OldFreq    float64 // for frequency_changed
	NewFreq    float64 // for frequency_changed
}

// FieldSchema represents the schema information for a single field.
type FieldSchema struct {
	Type      string  `json:"type"`      // "string", "int", "bool", "float", "object", "array"
	Optional  bool    `json:"optional"`  // appears in <100% of samples
	Frequency float64 `json:"frequency"` // 0.0-1.0
}

// SchemaSnapshot represents a snapshot of API response schema at a point in time.
type SchemaSnapshot struct {
	Vendor      string                 `json:"vendor"`
	DeviceType  string                 `json:"device_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Fields      map[string]FieldSchema `json:"fields"`
	SampleCount int                    `json:"sample_count"`
}
