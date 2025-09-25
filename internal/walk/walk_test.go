package walk

import (
	"reflect"
	"strings"
	"testing"
)

func TestWalk(t *testing.T) {
	// Test data representing a parsed TOML structure
	testData := map[string]any{
		"username":         "alice",
		"private_password": "secret123",
		"database": map[string]any{
			"host":                       "localhost",
			"port":                       5432,
			"private_connection_string":  "postgresql://...",
			"private_encryption_enabled": true,
		},
		"servers": []any{
			map[string]any{
				"name":            "prod",
				"private_api_key": "key123",
			},
			map[string]any{
				"name":            "staging",
				"private_api_key": "key456",
				"settings": map[string]any{
					"private_token": "token789",
					"public":        true,
				},
			},
		},
	}

	t.Run("should visit all fields", func(t *testing.T) {
		var visitedKeys []string

		Walk(testData, func(path []string, key string, value any) (any, bool) {
			if key != "" {
				visitedKeys = append(visitedKeys, key)
			}
			return value, true
		})

		expectedKeys := []string{
			"username", "private_password", "database",
			"host", "port", "private_connection_string", "private_encryption_enabled",
			"servers",
			"[0]", "name", "private_api_key", // first server
			"[1]", "name", "private_api_key", "settings", // second server
			"private_token", "public", // settings in second server
		}

		if len(visitedKeys) != len(expectedKeys) {
			t.Errorf("Expected %d keys, got %d: %v", len(expectedKeys), len(visitedKeys), visitedKeys)
		}

		// Check that all expected keys were visited (order might vary)
		keySet := make(map[string]int)
		for _, key := range visitedKeys {
			keySet[key]++
		}

		expectedKeySet := map[string]int{
			"username": 1, "private_password": 1, "database": 1,
			"host": 1, "port": 1, "private_connection_string": 1, "private_encryption_enabled": 1,
			"servers": 1,
			"[0]":     1, "[1]": 1, // array indices
			"name": 2, "private_api_key": 2, "settings": 1, // name and private_api_key appear twice
			"private_token": 1, "public": 1,
		}

		if !reflect.DeepEqual(keySet, expectedKeySet) {
			t.Errorf("Key counts don't match.\nExpected: %v\nGot: %v", expectedKeySet, keySet)
		}
	})

	t.Run("should modify values", func(t *testing.T) {
		result := Walk(testData, func(path []string, key string, value any) (any, bool) {
			// Replace all private_* values with "ENCRYPTED"
			if strings.HasPrefix(key, "private_") {
				return "ENCRYPTED", true
			}
			return value, true
		})

		// Check that private values were replaced
		resultMap := result.(map[string]any)
		if resultMap["private_password"] != "ENCRYPTED" {
			t.Errorf("Expected private_password to be ENCRYPTED, got %v", resultMap["private_password"])
		}

		dbMap := resultMap["database"].(map[string]any)
		if dbMap["private_connection_string"] != "ENCRYPTED" {
			t.Errorf("Expected database.private_connection_string to be ENCRYPTED, got %v", dbMap["private_connection_string"])
		}

		servers := resultMap["servers"].([]any)
		server1 := servers[0].(map[string]any)
		if server1["private_api_key"] != "ENCRYPTED" {
			t.Errorf("Expected servers[0].private_api_key to be ENCRYPTED, got %v", server1["private_api_key"])
		}

		server2 := servers[1].(map[string]any)
		settings := server2["settings"].(map[string]any)
		if settings["private_token"] != "ENCRYPTED" {
			t.Errorf("Expected servers[1].settings.private_token to be ENCRYPTED, got %v", settings["private_token"])
		}

		// Check that non-private values weren't modified
		if resultMap["username"] != "alice" {
			t.Errorf("Expected username to remain alice, got %v", resultMap["username"])
		}
	})

	t.Run("should stop traversal when visitor returns false", func(t *testing.T) {
		var visitedKeys []string

		Walk(testData, func(path []string, key string, value any) (any, bool) {
			if key != "" {
				visitedKeys = append(visitedKeys, key)
			}
			// Stop at database
			if key == "database" {
				return value, false
			}
			return value, true
		})

		// Should visit root level keys but not recurse into database
		foundDatabase := false
		foundHost := false
		for _, key := range visitedKeys {
			if key == "database" {
				foundDatabase = true
			}
			if key == "host" {
				foundHost = true
			}
		}

		if !foundDatabase {
			t.Error("Expected to visit database key")
		}
		if foundHost {
			t.Error("Expected NOT to visit host key (inside database)")
		}
	})
}

func TestFindFields(t *testing.T) {
	testData := map[string]any{
		"username":         "alice",
		"private_password": "secret123",
		"database": map[string]any{
			"host":                      "localhost",
			"private_connection_string": "postgresql://...",
		},
		"servers": []any{
			map[string]any{
				"name":            "prod",
				"private_api_key": "key123",
			},
		},
	}

	t.Run("should find fields with private_ prefix", func(t *testing.T) {
		fields := FindFields(testData, func(path []string, key string, value any) bool {
			return strings.HasPrefix(key, "private_")
		})

		expectedPaths := map[string][]string{
			"private_password":                   {"private_password"},
			"database.private_connection_string": {"database", "private_connection_string"},
			"servers.[0].private_api_key":        {"servers", "[0]", "private_api_key"},
		}

		if len(fields) != len(expectedPaths) {
			t.Errorf("Expected %d fields, got %d", len(expectedPaths), len(fields))
		}

		foundPaths := make(map[string][]string)
		for _, field := range fields {
			pathStr := field.GetFullPath()
			foundPaths[pathStr] = field.Path
		}

		for expectedPathStr, expectedPath := range expectedPaths {
			foundPath, exists := foundPaths[expectedPathStr]
			if !exists {
				t.Errorf("Expected to find path %s, but didn't", expectedPathStr)
				continue
			}
			if !reflect.DeepEqual(foundPath, expectedPath) {
				t.Errorf("Path %s: expected %v, got %v", expectedPathStr, expectedPath, foundPath)
			}
		}
	})

	t.Run("should find string values", func(t *testing.T) {
		fields := FindFields(testData, func(path []string, key string, value any) bool {
			_, isString := value.(string)
			return isString
		})

		// Should find all string values
		expectedCount := 4 // username, private_password, host, private_connection_string, name, private_api_key
		if len(fields) < expectedCount {
			t.Errorf("Expected at least %d string fields, got %d", expectedCount, len(fields))
		}
	})
}

func TestGetValue(t *testing.T) {
	testData := map[string]any{
		"username": "alice",
		"database": map[string]any{
			"host": "localhost",
			"port": 5432,
		},
		"servers": []any{
			map[string]any{
				"name": "prod",
			},
		},
	}

	tests := []struct {
		name     string
		path     []string
		expected any
		found    bool
	}{
		{"root level string", []string{"username"}, "alice", true},
		{"nested string", []string{"database", "host"}, "localhost", true},
		{"nested int", []string{"database", "port"}, 5432, true},
		{"array element", []string{"servers", "[0]", "name"}, "prod", true},
		{"nonexistent key", []string{"nonexistent"}, nil, false},
		{"nonexistent nested", []string{"database", "nonexistent"}, nil, false},
		{"invalid array index", []string{"servers", "[5]", "name"}, nil, false},
		{"empty path", []string{}, testData, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value, found := GetValue(testData, test.path)
			if found != test.found {
				t.Errorf("Expected found=%v, got found=%v", test.found, found)
			}
			if found && !reflect.DeepEqual(value, test.expected) {
				t.Errorf("Expected value=%v, got value=%v", test.expected, value)
			}
		})
	}
}

func TestSetValue(t *testing.T) {
	t.Run("should set values in map", func(t *testing.T) {
		testData := map[string]any{
			"username": "alice",
			"database": map[string]any{
				"host": "localhost",
			},
		}

		// Set root level value
		success := SetValue(testData, []string{"username"}, "bob")
		if !success {
			t.Error("Failed to set root level value")
		}
		if testData["username"] != "bob" {
			t.Errorf("Expected username=bob, got %v", testData["username"])
		}

		// Set nested value
		success = SetValue(testData, []string{"database", "host"}, "remotehost")
		if !success {
			t.Error("Failed to set nested value")
		}
		dbMap := testData["database"].(map[string]any)
		if dbMap["host"] != "remotehost" {
			t.Errorf("Expected database.host=remotehost, got %v", dbMap["host"])
		}

		// Set new value
		success = SetValue(testData, []string{"database", "port"}, 3306)
		if !success {
			t.Error("Failed to set new value")
		}
		if dbMap["port"] != 3306 {
			t.Errorf("Expected database.port=3306, got %v", dbMap["port"])
		}
	})

	t.Run("should set values in array", func(t *testing.T) {
		testData := map[string]any{
			"servers": []any{
				map[string]any{"name": "prod"},
				map[string]any{"name": "staging"},
			},
		}

		success := SetValue(testData, []string{"servers", "[0]", "name"}, "production")
		if !success {
			t.Error("Failed to set array element value")
		}

		servers := testData["servers"].([]any)
		server := servers[0].(map[string]any)
		if server["name"] != "production" {
			t.Errorf("Expected servers[0].name=production, got %v", server["name"])
		}
	})

	t.Run("should fail for invalid paths", func(t *testing.T) {
		testData := map[string]any{
			"username": "alice",
		}

		// Empty path
		success := SetValue(testData, []string{}, "value")
		if success {
			t.Error("Expected failure for empty path")
		}

		// Nonexistent parent
		success = SetValue(testData, []string{"nonexistent", "child"}, "value")
		if success {
			t.Error("Expected failure for nonexistent parent")
		}
	})
}

func TestIsScalarValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"string", "hello", true},
		{"int", 42, true},
		{"bool", true, true},
		{"float", 3.14, true},
		{"nil", nil, true},
		{"map", map[string]any{"key": "value"}, false},
		{"slice", []any{1, 2, 3}, false},
		{"array", [3]int{1, 2, 3}, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsScalarValue(test.value)
			if result != test.expected {
				t.Errorf("IsScalarValue(%v): expected %v, got %v", test.value, test.expected, result)
			}
		})
	}
}

func TestFieldInfo(t *testing.T) {
	field := FieldInfo{
		Path:  []string{"database", "config", "private_password"},
		Key:   "private_password",
		Value: "secret",
	}

	expectedPath := "database.config.private_password"
	actualPath := field.GetFullPath()
	if actualPath != expectedPath {
		t.Errorf("Expected full path %s, got %s", expectedPath, actualPath)
	}

	// Test with empty path
	field2 := FieldInfo{
		Path:  []string{},
		Key:   "root_key",
		Value: "value",
	}

	expectedPath2 := "root_key"
	actualPath2 := field2.GetFullPath()
	if actualPath2 != expectedPath2 {
		t.Errorf("Expected full path %s, got %s", expectedPath2, actualPath2)
	}
}
