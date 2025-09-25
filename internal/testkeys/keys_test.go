package testkeys

import (
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	t.Helper()

	testData := []byte("Hello, World! This is test data for viola.")

	encrypted, err := EncryptTestData(testData)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	if encrypted == "" {
		t.Fatal("Encrypted data is empty")
	}

	// Check that it looks like armored age data
	if len(encrypted) < 100 {
		t.Errorf("Encrypted data seems too short: %d bytes", len(encrypted))
	}

	decrypted, err := DecryptTestData(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt test data: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Decrypted data doesn't match original.\nOriginal: %s\nDecrypted: %s", testData, decrypted)
	}
}

func TestGetTestIdentities(t *testing.T) {
	t.Helper()

	identities, err := GetTestIdentities()
	if err != nil {
		t.Fatalf("Failed to get test identities: %v", err)
	}

	if len(identities) != 3 {
		t.Errorf("Expected 3 test identities, got %d", len(identities))
	}
}

func TestGetTestRecipients(t *testing.T) {
	t.Helper()

	recipients, err := GetTestRecipients()
	if err != nil {
		t.Fatalf("Failed to get test recipients: %v", err)
	}

	if len(recipients) != 3 {
		t.Errorf("Expected 3 test recipients, got %d", len(recipients))
	}
}
