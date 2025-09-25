package enc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andreweick/viola/internal/testkeys"
)

func TestEncryptDecrypt(t *testing.T) {
	testData := []byte("Hello, World! This is test data for encryption.")

	recipients, err := testkeys.GetTestRecipients()
	if err != nil {
		t.Fatalf("Failed to get test recipients: %v", err)
	}

	// Test encryption
	encrypted, err := Encrypt(testData, recipients)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if encrypted == "" {
		t.Fatal("Encrypted data is empty")
	}

	// Check that it looks like armored age data
	if !strings.Contains(encrypted, "-----BEGIN AGE ENCRYPTED FILE-----") {
		t.Error("Encrypted data doesn't contain expected armor header")
	}

	if !strings.Contains(encrypted, "-----END AGE ENCRYPTED FILE-----") {
		t.Error("Encrypted data doesn't contain expected armor footer")
	}

	// Test decryption
	identities, err := testkeys.GetTestIdentities()
	if err != nil {
		t.Fatalf("Failed to get test identities: %v", err)
	}

	decrypted, err := Decrypt(encrypted, identities)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(testData) {
		t.Errorf("Decrypted data doesn't match original.\nOriginal: %s\nDecrypted: %s", testData, decrypted)
	}
}

func TestEncryptNoRecipients(t *testing.T) {
	testData := []byte("test data")

	_, err := Encrypt(testData, nil)
	if err == nil {
		t.Fatal("Expected error when encrypting with no recipients")
	}

	if !strings.Contains(err.Error(), "no recipients") {
		t.Errorf("Expected 'no recipients' error, got: %v", err)
	}
}

func TestDecryptNoIdentities(t *testing.T) {
	// Create some encrypted data first
	recipients, err := testkeys.GetTestRecipients()
	if err != nil {
		t.Fatalf("Failed to get test recipients: %v", err)
	}

	encrypted, err := Encrypt([]byte("test"), recipients)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// Try to decrypt with no identities
	_, err = Decrypt(encrypted, nil)
	if err == nil {
		t.Fatal("Expected error when decrypting with no identities")
	}

	if !strings.Contains(err.Error(), "no identities") {
		t.Errorf("Expected 'no identities' error, got: %v", err)
	}
}

func TestKeySourcesLoadIdentities(t *testing.T) {
	t.Run("load from data", func(t *testing.T) {
		ks := KeySources{
			IdentitiesData: []string{
				testkeys.TestIdentity1,
				testkeys.TestIdentity2,
			},
		}

		identities, err := ks.LoadIdentities()
		if err != nil {
			t.Fatalf("Failed to load identities: %v", err)
		}

		if len(identities) != 2 {
			t.Errorf("Expected 2 identities, got %d", len(identities))
		}
	})

	t.Run("load from file", func(t *testing.T) {
		// Create a temporary identity file
		tmpDir := t.TempDir()
		identityFile := filepath.Join(tmpDir, "identities.txt")

		content := testkeys.TestIdentity1 + "\n" + testkeys.TestIdentity2 + "\n"
		err := os.WriteFile(identityFile, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to write identity file: %v", err)
		}

		ks := KeySources{
			IdentitiesFile: identityFile,
		}

		identities, err := ks.LoadIdentities()
		if err != nil {
			t.Fatalf("Failed to load identities: %v", err)
		}

		if len(identities) != 2 {
			t.Errorf("Expected 2 identities, got %d", len(identities))
		}
	})

	t.Run("load with passphrase", func(t *testing.T) {
		ks := KeySources{
			IdentitiesData: []string{testkeys.TestIdentity1},
			PassphraseProvider: func() (string, error) {
				return testkeys.TestPassphrase, nil
			},
		}

		identities, err := ks.LoadIdentities()
		if err != nil {
			t.Fatalf("Failed to load identities: %v", err)
		}

		// Should have X25519 identity + scrypt identity
		if len(identities) != 2 {
			t.Errorf("Expected 2 identities (X25519 + scrypt), got %d", len(identities))
		}
	})
}

func TestKeySourcesLoadRecipients(t *testing.T) {
	t.Run("load from explicit recipients", func(t *testing.T) {
		ks := KeySources{
			Recipients: []string{
				testkeys.TestRecipient1,
				testkeys.TestRecipient2,
			},
		}

		recipients, err := ks.LoadRecipients()
		if err != nil {
			t.Fatalf("Failed to load recipients: %v", err)
		}

		if len(recipients) != 2 {
			t.Errorf("Expected 2 recipients, got %d", len(recipients))
		}
	})

	t.Run("load from file", func(t *testing.T) {
		// Create a temporary recipients file
		tmpDir := t.TempDir()
		recipientsFile := filepath.Join(tmpDir, "recipients.txt")

		content := `# Test recipients file
` + testkeys.TestRecipient1 + `

# Second recipient
` + testkeys.TestRecipient2 + `
`
		err := os.WriteFile(recipientsFile, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write recipients file: %v", err)
		}

		ks := KeySources{
			RecipientsFile: recipientsFile,
		}

		recipients, err := ks.LoadRecipients()
		if err != nil {
			t.Fatalf("Failed to load recipients: %v", err)
		}

		if len(recipients) != 2 {
			t.Errorf("Expected 2 recipients, got %d", len(recipients))
		}
	})

	t.Run("load with passphrase", func(t *testing.T) {
		ks := KeySources{
			Recipients: []string{testkeys.TestRecipient1},
			PassphraseProvider: func() (string, error) {
				return testkeys.TestPassphrase, nil
			},
		}

		recipients, err := ks.LoadRecipients()
		if err != nil {
			t.Fatalf("Failed to load recipients: %v", err)
		}

		// Should have X25519 recipient + scrypt recipient
		if len(recipients) != 2 {
			t.Errorf("Expected 2 recipients (X25519 + scrypt), got %d", len(recipients))
		}
	})
}

func TestGetRecipientStrings(t *testing.T) {
	recipients, err := testkeys.GetTestRecipients()
	if err != nil {
		t.Fatalf("Failed to get test recipients: %v", err)
	}

	strs := GetRecipientStrings(recipients)
	if len(strs) != 3 {
		t.Errorf("Expected 3 recipient strings, got %d", len(strs))
	}

	// Check that we get the expected recipient strings
	expectedRecipients := []string{
		testkeys.TestRecipient1,
		testkeys.TestRecipient2,
		testkeys.TestRecipient3,
	}

	for i, expected := range expectedRecipients {
		if i >= len(strs) {
			t.Errorf("Missing recipient string at index %d", i)
			continue
		}
		if strs[i] != expected {
			t.Errorf("Recipient string %d: expected %s, got %s", i, expected, strs[i])
		}
	}
}

func TestHasPassphraseRecipient(t *testing.T) {
	t.Run("no passphrase recipient", func(t *testing.T) {
		recipients, err := testkeys.GetTestRecipients()
		if err != nil {
			t.Fatalf("Failed to get test recipients: %v", err)
		}

		if HasPassphraseRecipient(recipients) {
			t.Error("Expected no passphrase recipient, but found one")
		}
	})

	t.Run("with passphrase recipient", func(t *testing.T) {
		ks := KeySources{
			PassphraseProvider: func() (string, error) {
				return testkeys.TestPassphrase, nil
			},
		}

		recipients, err := ks.LoadRecipients()
		if err != nil {
			t.Fatalf("Failed to load recipients: %v", err)
		}

		if !HasPassphraseRecipient(recipients) {
			t.Error("Expected passphrase recipient, but found none")
		}
	})
}

func TestEncryptDecryptWithPassphrase(t *testing.T) {
	testData := []byte("Test data with passphrase encryption")

	t.Run("encrypt with X25519, decrypt with both X25519 and passphrase", func(t *testing.T) {
		// Encrypt with X25519 recipients
		x25519Recipients, err := testkeys.GetTestRecipients()
		if err != nil {
			t.Fatalf("Failed to get test recipients: %v", err)
		}

		encrypted, err := Encrypt(testData, x25519Recipients)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Decrypt with X25519 identity
		identities, err := testkeys.GetTestIdentities()
		if err != nil {
			t.Fatalf("Failed to get test identities: %v", err)
		}

		decrypted, err := Decrypt(encrypted, identities[:1]) // Only use first identity
		if err != nil {
			t.Fatalf("Failed to decrypt with X25519 identity: %v", err)
		}

		if string(decrypted) != string(testData) {
			t.Errorf("Decrypted data doesn't match original")
		}
	})

	t.Run("encrypt with passphrase only", func(t *testing.T) {
		// Set up key sources with passphrase only
		ks := KeySources{
			PassphraseProvider: func() (string, error) {
				return testkeys.TestPassphrase, nil
			},
		}

		recipients, err := ks.LoadRecipients()
		if err != nil {
			t.Fatalf("Failed to load recipients: %v", err)
		}

		// Encrypt with passphrase recipient
		encrypted, err := Encrypt(testData, recipients)
		if err != nil {
			t.Fatalf("Failed to encrypt: %v", err)
		}

		// Decrypt with passphrase identity
		identities, err := ks.LoadIdentities()
		if err != nil {
			t.Fatalf("Failed to load identities: %v", err)
		}

		decrypted, err := Decrypt(encrypted, identities)
		if err != nil {
			t.Fatalf("Failed to decrypt with passphrase identity: %v", err)
		}

		if string(decrypted) != string(testData) {
			t.Errorf("Decrypted data with passphrase doesn't match original")
		}
	})
}
