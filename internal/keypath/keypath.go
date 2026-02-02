// Package keypath provides utilities for parsing and working with dot-notation key paths.
// Key paths are used to specify nested fields in configuration objects, supporting
// managed_keys filtering and path-aware configuration updates.
package keypath

import (
	"fmt"
	"strings"
)

// KeyPath represents a parsed dot-notation key path.
type KeyPath struct {
	Segments    []string // Individual path segments
	HasWildcard bool     // True if path contains a wildcard (*)
	WildcardIdx int      // Index of wildcard segment (-1 if none)
	Original    string   // Original unparsed path string
}

// Parse parses a dot-notation key string into a KeyPath.
// Examples:
//   - "name" -> single segment
//   - "radio_config.band_24" -> two segments
//   - "port_config.*.vlan_id" -> three segments with wildcard
func Parse(key string) KeyPath {
	if key == "" {
		return KeyPath{Original: key, WildcardIdx: -1}
	}

	segments := strings.Split(key, ".")
	kp := KeyPath{
		Segments:    segments,
		HasWildcard: false,
		WildcardIdx: -1,
		Original:    key,
	}

	for i, seg := range segments {
		if seg == "*" {
			kp.HasWildcard = true
			kp.WildcardIdx = i
			break
		}
	}

	return kp
}

// Validate checks if a key path string is valid.
// Returns an error if the path is malformed.
func Validate(key string) error {
	if key == "" {
		return fmt.Errorf("key path cannot be empty")
	}

	segments := strings.Split(key, ".")

	for i, seg := range segments {
		if seg == "" {
			return fmt.Errorf("key path has empty segment at position %d: %q", i, key)
		}
		// Wildcard must not be the last segment (needs a field to match)
		if seg == "*" && i == len(segments)-1 {
			return fmt.Errorf("wildcard (*) cannot be the last segment: %q", key)
		}
	}

	return nil
}

// Depth returns the nesting depth of the key path.
func (kp KeyPath) Depth() int {
	return len(kp.Segments)
}

// IsNested returns true if the path has more than one segment.
func (kp KeyPath) IsNested() bool {
	return len(kp.Segments) > 1
}

// First returns the first segment of the path.
func (kp KeyPath) First() string {
	if len(kp.Segments) == 0 {
		return ""
	}
	return kp.Segments[0]
}

// Rest returns a new KeyPath with all segments except the first.
func (kp KeyPath) Rest() KeyPath {
	if len(kp.Segments) <= 1 {
		return KeyPath{}
	}

	rest := KeyPath{
		Segments:    kp.Segments[1:],
		HasWildcard: kp.HasWildcard && kp.WildcardIdx > 0,
		WildcardIdx: kp.WildcardIdx - 1,
		Original:    strings.Join(kp.Segments[1:], "."),
	}

	if rest.WildcardIdx < 0 {
		rest.HasWildcard = false
		rest.WildcardIdx = -1
	}

	return rest
}

// String returns the dot-notation string representation.
func (kp KeyPath) String() string {
	return strings.Join(kp.Segments, ".")
}

// StartsWith returns true if this path starts with the given prefix path.
func (kp KeyPath) StartsWith(prefix KeyPath) bool {
	if len(prefix.Segments) > len(kp.Segments) {
		return false
	}

	for i, seg := range prefix.Segments {
		if seg != "*" && kp.Segments[i] != seg {
			return false
		}
	}

	return true
}

// Matches returns true if this path matches the given pattern.
// Wildcards (*) in the pattern match any single segment.
func (kp KeyPath) Matches(pattern KeyPath) bool {
	if len(kp.Segments) != len(pattern.Segments) {
		return false
	}

	for i, seg := range pattern.Segments {
		if seg != "*" && kp.Segments[i] != seg {
			return false
		}
	}

	return true
}

// GetValueAtPath retrieves a value from a nested map using the path segments.
// Returns the value and true if found, nil and false otherwise.
func GetValueAtPath(data map[string]interface{}, path []string) (interface{}, bool) {
	if len(path) == 0 || data == nil {
		return nil, false
	}

	current := path[0]
	val, exists := data[current]
	if !exists {
		return nil, false
	}

	// If this is the last segment, return the value
	if len(path) == 1 {
		return val, true
	}

	// Otherwise, descend into nested map
	nested, ok := val.(map[string]interface{})
	if !ok {
		return nil, false
	}

	return GetValueAtPath(nested, path[1:])
}

// SetValueAtPath sets a value in a nested map at the given path.
// Creates intermediate maps as needed.
func SetValueAtPath(data map[string]interface{}, path []string, value interface{}) {
	if len(path) == 0 || data == nil {
		return
	}

	current := path[0]

	// If this is the last segment, set the value
	if len(path) == 1 {
		data[current] = value
		return
	}

	// Otherwise, get or create the nested map and descend
	existing, exists := data[current]
	var nested map[string]interface{}

	if exists {
		nested, _ = existing.(map[string]interface{})
	}

	if nested == nil {
		nested = make(map[string]interface{})
		data[current] = nested
	}

	SetValueAtPath(nested, path[1:], value)
}

// DeleteValueAtPath removes a value from a nested map at the given path.
// Returns true if the value was deleted, false if it didn't exist.
func DeleteValueAtPath(data map[string]interface{}, path []string) bool {
	if len(path) == 0 || data == nil {
		return false
	}

	current := path[0]

	// If this is the last segment, delete the value
	if len(path) == 1 {
		_, exists := data[current]
		if exists {
			delete(data, current)
			return true
		}
		return false
	}

	// Otherwise, descend into nested map
	existing, exists := data[current]
	if !exists {
		return false
	}

	nested, ok := existing.(map[string]interface{})
	if !ok {
		return false
	}

	return DeleteValueAtPath(nested, path[1:])
}

// CollectMatchingPaths finds all paths in a data map that match the given key path.
// Wildcards are expanded to all matching keys at that level.
func CollectMatchingPaths(data map[string]interface{}, kp KeyPath) [][]string {
	if len(kp.Segments) == 0 || data == nil {
		return nil
	}

	return collectMatchingPathsRecursive(data, kp.Segments, nil)
}

func collectMatchingPathsRecursive(data map[string]interface{}, segments []string, prefix []string) [][]string {
	if len(segments) == 0 {
		return nil
	}

	current := segments[0]
	remaining := segments[1:]
	var results [][]string

	if current == "*" {
		// Expand wildcard to all keys
		for key, val := range data {
			newPrefix := append(append([]string{}, prefix...), key)

			if len(remaining) == 0 {
				results = append(results, newPrefix)
			} else {
				nested, ok := val.(map[string]interface{})
				if ok {
					results = append(results, collectMatchingPathsRecursive(nested, remaining, newPrefix)...)
				}
			}
		}
	} else {
		// Match specific key
		val, exists := data[current]
		if exists {
			newPrefix := append(append([]string{}, prefix...), current)

			if len(remaining) == 0 {
				results = append(results, newPrefix)
			} else {
				nested, ok := val.(map[string]interface{})
				if ok {
					results = append(results, collectMatchingPathsRecursive(nested, remaining, newPrefix)...)
				}
			}
		}
	}

	return results
}

// ExpandWildcardPath expands a wildcard path against actual data.
// Returns all concrete paths that match the pattern.
func ExpandWildcardPath(data map[string]interface{}, pattern string) []string {
	kp := Parse(pattern)
	if !kp.HasWildcard {
		// No wildcard, return as-is if path exists
		if _, found := GetValueAtPath(data, kp.Segments); found {
			return []string{pattern}
		}
		return nil
	}

	paths := CollectMatchingPaths(data, kp)
	result := make([]string, len(paths))
	for i, p := range paths {
		result[i] = strings.Join(p, ".")
	}
	return result
}

// IsKeyManaged checks if a specific key path is covered by the managed keys list.
// A key is managed if:
//   - It appears directly in the managed keys list
//   - A parent path appears in the managed keys list (implicit inclusion)
//   - It matches a wildcard pattern in the managed keys list
func IsKeyManaged(key string, managedKeys []string) bool {
	if len(managedKeys) == 0 {
		return false
	}

	kp := Parse(key)

	for _, mk := range managedKeys {
		mkp := Parse(mk)

		// Direct match
		if kp.String() == mkp.String() {
			return true
		}

		// Parent path match - if managed key is parent of our key, key is managed
		if kp.StartsWith(mkp) {
			return true
		}

		// Wildcard pattern match
		if mkp.HasWildcard && kp.Matches(mkp) {
			return true
		}
	}

	return false
}

// FilterMapByManagedKeys filters a configuration map to only include managed keys.
// Returns a new map containing only the fields specified in managedKeys.
// Supports dot-notation paths and wildcards.
func FilterMapByManagedKeys(data map[string]interface{}, managedKeys []string) map[string]interface{} {
	if len(managedKeys) == 0 {
		return nil
	}

	result := make(map[string]interface{})

	for _, mk := range managedKeys {
		kp := Parse(mk)

		if kp.HasWildcard {
			// Expand wildcard and copy all matching paths
			paths := CollectMatchingPaths(data, kp)
			for _, path := range paths {
				if val, found := GetValueAtPath(data, path); found {
					SetValueAtPath(result, path, deepCopyValue(val))
				}
			}
		} else {
			// Copy specific path
			if val, found := GetValueAtPath(data, kp.Segments); found {
				SetValueAtPath(result, kp.Segments, deepCopyValue(val))
			}
		}
	}

	return result
}

// deepCopyValue creates a deep copy of a value.
func deepCopyValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			result[k] = deepCopyValue(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = deepCopyValue(item)
		}
		return result
	default:
		// Primitive types are immutable, return as-is
		return v
	}
}

// CompareValuesAtPath compares values at a specific path in two maps.
// Returns true if values differ, false if they are equal.
func CompareValuesAtPath(a, b map[string]interface{}, path string) bool {
	kp := Parse(path)
	valA, foundA := GetValueAtPath(a, kp.Segments)
	valB, foundB := GetValueAtPath(b, kp.Segments)

	if foundA != foundB {
		return true
	}

	if !foundA {
		return false
	}

	return !deepEqual(valA, valB)
}

// deepEqual compares two values for deep equality.
func deepEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch va := a.(type) {
	case map[string]interface{}:
		vb, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !deepEqual(v, vb[k]) {
				return false
			}
		}
		return true

	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !deepEqual(va[i], vb[i]) {
				return false
			}
		}
		return true

	default:
		return a == b
	}
}
