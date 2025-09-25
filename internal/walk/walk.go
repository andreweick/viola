// Package walk provides utilities for traversing TOML data structures.
package walk

import (
	"fmt"
	"reflect"
)

// VisitFunc is called for each field during traversal.
// path contains the keys leading to this field (e.g., ["database", "config", "private_password"])
// key is the current field's key
// value is the current field's value (may be modified by returning a new value)
// Returns the new value and whether to continue traversal
type VisitFunc func(path []string, key string, value any) (newValue any, cont bool)

// Walk traverses a parsed TOML data structure (map[string]any) and calls the visitor
// function for each field. The visitor can modify values by returning a different value.
func Walk(data any, visit VisitFunc) any {
	return walkValue(nil, "", data, visit)
}

// walkValue recursively walks through any value type
func walkValue(path []string, key string, value any, visit VisitFunc) any {
	// Call the visitor for this value
	newValue, cont := visit(path, key, value)
	if !cont {
		return newValue
	}

	// If visitor modified the value, use the new value for further traversal
	value = newValue

	switch v := value.(type) {
	case map[string]any:
		return walkMap(path, key, v, visit)
	case []any:
		return walkSlice(path, key, v, visit)
	default:
		// Leaf value (string, int, bool, etc.)
		return value
	}
}

// walkMap walks through a map (TOML table)
func walkMap(parentPath []string, parentKey string, m map[string]any, visit VisitFunc) map[string]any {
	// Build the path for this level
	var currentPath []string
	if parentKey != "" {
		currentPath = append(parentPath, parentKey)
	} else {
		currentPath = parentPath
	}

	result := make(map[string]any)
	for k, v := range m {
		newValue := walkValue(currentPath, k, v, visit)
		result[k] = newValue
	}
	return result
}

// walkSlice walks through a slice (TOML array)
func walkSlice(parentPath []string, parentKey string, s []any, visit VisitFunc) []any {
	// Build the path for this level
	var currentPath []string
	if parentKey != "" {
		currentPath = append(parentPath, parentKey)
	} else {
		currentPath = parentPath
	}

	result := make([]any, len(s))
	for i, v := range s {
		// For arrays, use the index as the key
		indexKey := fmt.Sprintf("[%d]", i)
		newValue := walkValue(currentPath, indexKey, v, visit)
		result[i] = newValue
	}
	return result
}

// FindFields searches for fields matching a predicate function and returns their paths and values
func FindFields(data any, predicate func(path []string, key string, value any) bool) []FieldInfo {
	var results []FieldInfo

	Walk(data, func(path []string, key string, value any) (any, bool) {
		if predicate(path, key, value) {
			fullPath := append(path, key)
			results = append(results, FieldInfo{
				Path:  fullPath,
				Key:   key,
				Value: value,
			})
		}
		return value, true
	})

	return results
}

// FieldInfo contains information about a field found during traversal
type FieldInfo struct {
	Path  []string // Full path including the key
	Key   string   // Just the key
	Value any      // The value
}

// GetFullPath returns the full path as a string, e.g., "database.config.private_password"
func (fi FieldInfo) GetFullPath() string {
	if len(fi.Path) == 0 {
		return fi.Key
	}

	result := ""
	for i, part := range fi.Path {
		if i > 0 {
			result += "."
		}
		result += part
	}
	return result
}

// GetValue safely gets a value from the data structure using a path
func GetValue(data any, path []string) (any, bool) {
	if len(path) == 0 {
		return data, true
	}

	current := data
	for _, key := range path {
		switch v := current.(type) {
		case map[string]any:
			val, exists := v[key]
			if !exists {
				return nil, false
			}
			current = val
		case []any:
			// Handle array access like "[0]"
			if len(key) < 3 || key[0] != '[' || key[len(key)-1] != ']' {
				return nil, false
			}
			indexStr := key[1 : len(key)-1]
			var index int
			if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
				return nil, false
			}
			if index < 0 || index >= len(v) {
				return nil, false
			}
			current = v[index]
		default:
			return nil, false
		}
	}

	return current, true
}

// SetValue safely sets a value in the data structure using a path
func SetValue(data any, path []string, newValue any) bool {
	if len(path) == 0 {
		return false
	}

	if len(path) == 1 {
		// Setting at root level
		if m, ok := data.(map[string]any); ok {
			m[path[0]] = newValue
			return true
		}
		return false
	}

	// Navigate to parent
	parent, found := GetValue(data, path[:len(path)-1])
	if !found {
		return false
	}

	// Set the value in parent
	finalKey := path[len(path)-1]
	switch p := parent.(type) {
	case map[string]any:
		p[finalKey] = newValue
		return true
	case []any:
		// Handle array access like "[0]"
		if len(finalKey) < 3 || finalKey[0] != '[' || finalKey[len(finalKey)-1] != ']' {
			return false
		}
		indexStr := finalKey[1 : len(finalKey)-1]
		var index int
		if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil {
			return false
		}
		if index < 0 || index >= len(p) {
			return false
		}
		p[index] = newValue
		return true
	default:
		return false
	}
}

// IsScalarValue checks if a value is a scalar (not a map or slice)
func IsScalarValue(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array:
		return false
	case reflect.Ptr:
		if v.IsNil() {
			return true
		}
		return IsScalarValue(v.Elem().Interface())
	default:
		return true
	}
}
