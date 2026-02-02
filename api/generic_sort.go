package api

import (
	"sort"

	"github.com/maruel/natural"
)

// SortField defines a field extractor function for sorting
type SortField[T any] func(item T) string

// SortConfig defines the configuration for multi-level sorting
type SortConfig[T any] struct {
	// Fields defines the sort fields in priority order
	Fields []SortField[T]
	// EmptyLast determines if empty values should be sorted last
	// Set to false to sort empty values first
	EmptyLast bool
}

// MultiLevelSort performs generic multi-level natural sorting on any slice
// The sorting is stable and uses natural ordering for strings
// Empty values are placed at the end of their respective groups when EmptyLast is true
func MultiLevelSort[T any](items []T, config SortConfig[T]) []T {
	if len(items) == 0 || len(config.Fields) == 0 {
		return items
	}

	// Create a copy to avoid modifying the original slice
	result := make([]T, len(items))
	copy(result, items)

	sort.SliceStable(result, func(i, j int) bool {
		// Compare items using each field in priority order
		for _, field := range config.Fields {
			valI := field(result[i])
			valJ := field(result[j])

			// Handle empty values based on configuration
			if valI == "" && valJ == "" {
				// Both empty, continue to next field
				continue
			} else if valI == "" && valJ != "" {
				// i is empty, j is not
				return !config.EmptyLast // if EmptyLast is true, empty comes after (return false)
			} else if valI != "" && valJ == "" {
				// i is not empty, j is empty
				return config.EmptyLast // if EmptyLast is true, non-empty comes first (return true)
			}

			// Both have values - compare them
			if valI != valJ {
				return natural.Less(valI, valJ)
			}
			// Values are equal, continue to next field
		}

		// All fields are equal
		return false
	})

	return result
}

// Common field extractors that can be reused across different types

// GetStringField creates a field extractor for simple string pointer fields
func GetStringField[T any](getter func(T) *string) SortField[T] {
	return func(item T) string {
		val := getter(item)
		if val == nil {
			return ""
		}
		return *val
	}
}

// SortDeviceProfiles sorts device profiles by name
func SortDeviceProfiles(profiles []*DeviceProfile) []*DeviceProfile {
	config := SortConfig[*DeviceProfile]{
		Fields: []SortField[*DeviceProfile]{
			// Sort by profile name
			GetStringField(func(p *DeviceProfile) *string { return p.Name }),
		},
		EmptyLast: true,
	}

	return MultiLevelSort(profiles, config)
}
