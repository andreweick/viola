package viola

import (
	"reflect"
	"strings"
	"testing"

	"github.com/andreweick/viola/internal/testkeys"
	"github.com/andreweick/viola/pkg/enc"
)

func TestLoadDecryption(t *testing.T) {
	// Create some test data with encrypted fields
	testData := map[string]any{
		"username":         "alice",
		"private_password": "secret123",
		"database": map[string]any{
			"host":                      "localhost",
			"port":                      5432,
			"private_connection_string": "postgresql://user:pass@localhost/db",
		},
	}

	// First save it to get encrypted TOML
	opts := Options{
		Keys: enc.KeySources{
			Recipients: []string{testkeys.TestRecipient1},
		},
	}

	encryptedTOML, _, err := Save(testData, opts)
	if err != nil {
		t.Fatalf("Failed to save test data: %v", err)
	}

	// Now test loading with decryption
	opts.Keys = enc.KeySources{
		IdentitiesData: []string{testkeys.TestIdentity1},
	}

	result, err := Load(encryptedTOML, opts)
	if err != nil {
		t.Fatalf("Failed to load encrypted data: %v", err)
	}

	// Check that data was decrypted correctly
	if result.Tree["username"] != "alice" {
		t.Errorf("Expected username=alice, got %v", result.Tree["username"])
	}

	if result.Tree["private_password"] != "secret123" {
		t.Errorf("Expected private_password=secret123, got %v", result.Tree["private_password"])
	}

	dbMap := result.Tree["database"].(map[string]any)
	if dbMap["host"] != "localhost" {
		t.Errorf("Expected database.host=localhost, got %v", dbMap["host"])
	}

	if dbMap["private_connection_string"] != "postgresql://user:pass@localhost/db" {
		t.Errorf("Expected decrypted connection string, got %v", dbMap["private_connection_string"])
	}

	// Check that we have field metadata
	if len(result.Fields) < 2 {
		t.Errorf("Expected at least 2 encrypted fields in metadata, got %d", len(result.Fields))
	}

	// Find the private_password field in metadata
	var passwordField *FieldMeta
	for _, field := range result.Fields {
		if len(field.Path) == 1 && field.Path[0] == "private_password" {
			passwordField = &field
			break
		}
	}

	if passwordField == nil {
		t.Error("Expected to find private_password in field metadata")
	} else {
		if !passwordField.WasEncrypted {
			t.Error("Expected private_password to be marked as encrypted")
		}
		if !strings.Contains(passwordField.Armored, "-----BEGIN AGE ENCRYPTED FILE-----") {
			t.Error("Expected armored data to contain age header")
		}
	}
}

func TestSaveEncryption(t *testing.T) {
	testData := map[string]any{
		"username":         "alice",
		"private_password": "secret123",
		"database": map[string]any{
			"host":                      "localhost",
			"private_connection_string": "postgresql://user:pass@localhost/db",
		},
		"servers": []any{
			map[string]any{
				"name":            "prod",
				"private_api_key": "key123",
			},
		},
	}

	opts := Options{
		Keys: enc.KeySources{
			Recipients: []string{testkeys.TestRecipient1, testkeys.TestRecipient2},
		},
	}

	tomlData, fields, err := Save(testData, opts)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	if len(tomlData) == 0 {
		t.Fatal("Expected non-empty TOML data")
	}

	// Check that private fields were encrypted
	tomlStr := string(tomlData)
	if !strings.Contains(tomlStr, "-----BEGIN AGE ENCRYPTED FILE-----") {
		t.Error("Expected TOML to contain encrypted data")
	}

	// Username should remain plaintext
	if !strings.Contains(tomlStr, `username = "alice"`) {
		t.Error("Expected username to remain plaintext")
	}

	// Private fields should NOT contain plaintext
	if strings.Contains(tomlStr, "secret123") {
		t.Error("Expected private_password not to contain plaintext")
	}

	if strings.Contains(tomlStr, "postgresql://user:pass@localhost/db") {
		t.Error("Expected private_connection_string not to contain plaintext")
	}

	// Check field metadata
	expectedEncryptedFields := []string{"private_password", "private_connection_string", "private_api_key"}
	encryptedCount := 0

	for _, field := range fields {
		if field.WasEncrypted {
			encryptedCount++

			// Check that recipients were recorded
			if len(field.UsedRecipients) != 2 {
				t.Errorf("Expected 2 recipients for field %v, got %d", field.Path, len(field.UsedRecipients))
			}

			expectedRecipients := []string{testkeys.TestRecipient1, testkeys.TestRecipient2}
			if !reflect.DeepEqual(field.UsedRecipients, expectedRecipients) {
				t.Errorf("Field %v: expected recipients %v, got %v", field.Path, expectedRecipients, field.UsedRecipients)
			}

			if field.UsedPassphrase {
				t.Errorf("Field %v: expected no passphrase, but UsedPassphrase=true", field.Path)
			}
		}
	}

	if encryptedCount != len(expectedEncryptedFields) {
		t.Errorf("Expected %d encrypted fields, got %d", len(expectedEncryptedFields), encryptedCount)
	}
}

func TestRoundTrip(t *testing.T) {
	originalData := map[string]any{
		"username":         "alice",
		"private_password": "secret123",
		"config": map[string]any{
			"debug":             true,
			"private_api_token": "token456",
		},
		"numbers": map[string]any{
			"port":                  8080,
			"private_secret_number": 42,
		},
		"private_array": []any{"item1", "item2"},
		"private_complex": map[string]any{
			"nested": "value",
			"count":  123,
		},
	}

	opts := Options{
		Keys: enc.KeySources{
			Recipients:     []string{testkeys.TestRecipient1},
			IdentitiesData: []string{testkeys.TestIdentity1},
		},
	}

	// Save (encrypt)
	tomlData, _, err := Save(originalData, opts)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Load (decrypt)
	result, err := Load(tomlData, opts)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Compare original and decrypted data
	if !reflect.DeepEqual(result.Tree["username"], originalData["username"]) {
		t.Errorf("Username mismatch: expected %v, got %v", originalData["username"], result.Tree["username"])
	}

	if !reflect.DeepEqual(result.Tree["private_password"], originalData["private_password"]) {
		t.Errorf("Private password mismatch: expected %v, got %v", originalData["private_password"], result.Tree["private_password"])
	}

	// Check nested structures
	origConfig := originalData["config"].(map[string]any)
	resultConfig := result.Tree["config"].(map[string]any)

	if !reflect.DeepEqual(resultConfig["debug"], origConfig["debug"]) {
		t.Errorf("Config debug mismatch: expected %v, got %v", origConfig["debug"], resultConfig["debug"])
	}

	if !reflect.DeepEqual(resultConfig["private_api_token"], origConfig["private_api_token"]) {
		t.Errorf("Config private_api_token mismatch: expected %v, got %v", origConfig["private_api_token"], resultConfig["private_api_token"])
	}

	// Check numbers (JSON marshaling may change int to float64 for encrypted fields)
	origNumbers := originalData["numbers"].(map[string]any)
	resultNumbers := result.Tree["numbers"].(map[string]any)

	// Non-encrypted field - TOML parsing may convert int to int64
	origPort := origNumbers["port"]
	resultPort := resultNumbers["port"]

	// Convert to int64 for comparison since TOML may parse as int64
	if int64(origPort.(int)) != resultPort.(int64) {
		t.Errorf("Numbers port mismatch: expected %v, got %v", origNumbers["port"], resultNumbers["port"])
	}

	// Encrypted field might be converted to float64
	if resultNumbers["private_secret_number"] != float64(42) {
		t.Errorf("Numbers private_secret_number mismatch: expected %v, got %v", float64(42), resultNumbers["private_secret_number"])
	}

	// Check that complex types (arrays and objects) are preserved
	// Note: JSON marshaling/unmarshaling may change number types (int -> float64)
	origArray := originalData["private_array"].([]any)
	resultArray := result.Tree["private_array"].([]any)
	if len(origArray) != len(resultArray) {
		t.Errorf("Array length mismatch: expected %d, got %d", len(origArray), len(resultArray))
	} else {
		for i, origItem := range origArray {
			if origItem != resultArray[i] {
				t.Errorf("Array[%d] mismatch: expected %v, got %v", i, origItem, resultArray[i])
			}
		}
	}

	// For complex objects, JSON marshaling converts int to float64
	origComplex := originalData["private_complex"].(map[string]any)
	resultComplex := result.Tree["private_complex"].(map[string]any)

	if resultComplex["nested"] != origComplex["nested"] {
		t.Errorf("Complex nested mismatch: expected %v, got %v", origComplex["nested"], resultComplex["nested"])
	}

	// JSON marshaling converts int 123 to float64 123.0
	if resultComplex["count"] != float64(123) {
		t.Errorf("Complex count mismatch: expected %v, got %v", float64(123), resultComplex["count"])
	}
}

func TestCustomShouldEncrypt(t *testing.T) {
	testData := map[string]any{
		"username":     "alice",
		"password":     "should_be_encrypted",
		"secret_key":   "also_encrypted",
		"public_value": "not_encrypted",
	}

	opts := Options{
		Keys: enc.KeySources{
			Recipients:     []string{testkeys.TestRecipient1},
			IdentitiesData: []string{testkeys.TestIdentity1},
		},
		// Custom encryption rule: encrypt fields containing "password" or "secret"
		ShouldEncrypt: func(path []string, key string, value any) bool {
			return strings.Contains(key, "password") || strings.Contains(key, "secret")
		},
	}

	// Save with custom encryption rules
	tomlData, fields, err := Save(testData, opts)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Check that the right fields were encrypted
	encryptedFields := make(map[string]bool)
	for _, field := range fields {
		if field.WasEncrypted && len(field.Path) == 1 {
			encryptedFields[field.Path[0]] = true
		}
	}

	expectedEncrypted := map[string]bool{
		"password":   true,
		"secret_key": true,
	}

	if !reflect.DeepEqual(encryptedFields, expectedEncrypted) {
		t.Errorf("Expected encrypted fields %v, got %v", expectedEncrypted, encryptedFields)
	}

	// Verify round trip
	result, err := Load(tomlData, opts)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if !reflect.DeepEqual(result.Tree, testData) {
		t.Errorf("Round trip failed: expected %v, got %v", testData, result.Tree)
	}
}

func TestTransform(t *testing.T) {
	originalTOML := `
username = "alice"
private_password = "old_secret"

[database]
host = "localhost"
private_connection_string = "old_connection"
`

	opts := Options{
		Keys: enc.KeySources{
			Recipients:     []string{testkeys.TestRecipient1},
			IdentitiesData: []string{testkeys.TestIdentity1},
		},
	}

	// Transform the configuration
	newTOML, fields, err := Transform([]byte(originalTOML), opts, func(tree any) error {
		treeMap := tree.(map[string]any)

		// Change the private password
		treeMap["private_password"] = "new_secret"

		// Change the database connection string
		db := treeMap["database"].(map[string]any)
		db["private_connection_string"] = "new_connection"

		// Add a new field
		treeMap["private_new_field"] = "new_value"

		return nil
	})

	if err != nil {
		t.Fatalf("Failed to transform: %v", err)
	}

	// Verify that changes were applied
	result, err := Load(newTOML, opts)
	if err != nil {
		t.Fatalf("Failed to load transformed data: %v", err)
	}

	if result.Tree["private_password"] != "new_secret" {
		t.Errorf("Expected private_password=new_secret, got %v", result.Tree["private_password"])
	}

	db := result.Tree["database"].(map[string]any)
	if db["private_connection_string"] != "new_connection" {
		t.Errorf("Expected new connection string, got %v", db["private_connection_string"])
	}

	if result.Tree["private_new_field"] != "new_value" {
		t.Errorf("Expected private_new_field=new_value, got %v", result.Tree["private_new_field"])
	}

	// Verify that we have field metadata for encrypted fields
	if len(fields) < 3 {
		t.Errorf("Expected at least 3 encrypted fields, got %d", len(fields))
	}
}

func TestSaveNoRecipients(t *testing.T) {
	testData := map[string]any{
		"private_password": "secret",
	}

	opts := Options{
		Keys: enc.KeySources{
			// No recipients provided
		},
	}

	_, _, err := Save(testData, opts)
	if err == nil {
		t.Fatal("Expected error when saving with no recipients")
	}

	if !strings.Contains(err.Error(), "no recipients") {
		t.Errorf("Expected 'no recipients' error, got: %v", err)
	}
}

func TestLoadMissingIdentities(t *testing.T) {
	// Create encrypted data
	testData := map[string]any{
		"private_password": "secret",
	}

	opts := Options{
		Keys: enc.KeySources{
			Recipients: []string{testkeys.TestRecipient1},
		},
	}

	encryptedTOML, _, err := Save(testData, opts)
	if err != nil {
		t.Fatalf("Failed to save test data: %v", err)
	}

	// Try to load without identities
	optsNoIdentities := Options{
		Keys: enc.KeySources{
			// No identities provided
		},
	}

	result, err := Load(encryptedTOML, optsNoIdentities)
	if err != nil {
		t.Fatalf("Load should not fail even without identities: %v", err)
	}

	// The encrypted field should remain encrypted (not decrypted)
	passwordValue := result.Tree["private_password"].(string)
	if !strings.Contains(passwordValue, "-----BEGIN AGE ENCRYPTED FILE-----") {
		t.Error("Expected password to remain encrypted when no identities available")
	}
}

func TestIdempotentSave(t *testing.T) {
	testData := map[string]any{
		"username":         "alice",
		"private_password": "secret123",
	}

	opts := Options{
		Keys: enc.KeySources{
			Recipients:     []string{testkeys.TestRecipient1},
			IdentitiesData: []string{testkeys.TestIdentity1},
		},
	}

	// First save
	firstSave, _, err := Save(testData, opts)
	if err != nil {
		t.Fatalf("Failed first save: %v", err)
	}

	// Load the encrypted data
	result, err := Load(firstSave, opts)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Save again without changes - should be idempotent
	secondSave, _, err := Save(result.Tree, opts)
	if err != nil {
		t.Fatalf("Failed second save: %v", err)
	}

	// The encrypted values should remain the same
	// (Note: age encryption includes randomness, so we can't compare bytes directly.
	//  Instead, we verify that both versions can be decrypted to the same value)

	result1, err := Load(firstSave, opts)
	if err != nil {
		t.Fatalf("Failed to load first save: %v", err)
	}

	result2, err := Load(secondSave, opts)
	if err != nil {
		t.Fatalf("Failed to load second save: %v", err)
	}

	if !reflect.DeepEqual(result1.Tree, result2.Tree) {
		t.Error("Expected idempotent save to produce the same decrypted result")
	}
}
