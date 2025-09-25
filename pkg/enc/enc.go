// Package enc provides age encryption and decryption helpers for viola.
package enc

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// KeySources contains various sources for age identities and recipients
type KeySources struct {
	// IdentitiesFile is the path to a file containing age private keys
	IdentitiesFile string

	// IdentitiesData contains age private keys as strings
	IdentitiesData []string

	// RecipientsFile is the path to a file containing age public keys
	RecipientsFile string

	// Recipients contains age public keys as strings
	Recipients []string

	// PassphraseProvider returns a passphrase for age-scrypt decryption
	PassphraseProvider func() (string, error)
}

// LoadIdentities loads age identities from the key sources
func (ks KeySources) LoadIdentities() ([]age.Identity, error) {
	var identities []age.Identity

	// Load from file
	if ks.IdentitiesFile != "" {
		fileIdentities, err := loadIdentitiesFromFile(ks.IdentitiesFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load identities from file %s: %w", ks.IdentitiesFile, err)
		}
		identities = append(identities, fileIdentities...)
	}

	// Load from data
	for _, identityStr := range ks.IdentitiesData {
		identity, err := age.ParseX25519Identity(identityStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse identity: %w", err)
		}
		identities = append(identities, identity)
	}

	// Add passphrase identity if provider exists
	if ks.PassphraseProvider != nil {
		passphrase, err := ks.PassphraseProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to get passphrase: %w", err)
		}
		scryptIdentity, err := age.NewScryptIdentity(passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to create scrypt identity: %w", err)
		}
		identities = append(identities, scryptIdentity)
	}

	return identities, nil
}

// LoadRecipients loads age recipients from the key sources
func (ks KeySources) LoadRecipients() ([]age.Recipient, error) {
	var recipients []age.Recipient

	// Load from file
	if ks.RecipientsFile != "" {
		fileRecipients, err := loadRecipientsFromFile(ks.RecipientsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load recipients from file %s: %w", ks.RecipientsFile, err)
		}
		recipients = append(recipients, fileRecipients...)
	}

	// Load from explicit recipients
	for _, recipientStr := range ks.Recipients {
		recipient, err := age.ParseX25519Recipient(recipientStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse recipient: %w", err)
		}
		recipients = append(recipients, recipient)
	}

	// Add passphrase recipient if provider exists
	if ks.PassphraseProvider != nil {
		passphrase, err := ks.PassphraseProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to get passphrase: %w", err)
		}
		scryptRecipient, err := age.NewScryptRecipient(passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to create scrypt recipient: %w", err)
		}
		recipients = append(recipients, scryptRecipient)
	}

	return recipients, nil
}

// Encrypt encrypts data with the given recipients and returns ASCII-armored ciphertext
func Encrypt(data []byte, recipients []age.Recipient) (string, error) {
	if len(recipients) == 0 {
		return "", fmt.Errorf("no recipients provided")
	}

	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)

	ageWriter, err := age.Encrypt(armorWriter, recipients...)
	if err != nil {
		return "", fmt.Errorf("failed to create age encryptor: %w", err)
	}

	if _, err := ageWriter.Write(data); err != nil {
		return "", fmt.Errorf("failed to write data: %w", err)
	}

	if err := ageWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close age writer: %w", err)
	}

	if err := armorWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close armor writer: %w", err)
	}

	return buf.String(), nil
}

// Decrypt decrypts ASCII-armored ciphertext using the given identities
func Decrypt(armoredData string, identities []age.Identity) ([]byte, error) {
	if len(identities) == 0 {
		return nil, fmt.Errorf("no identities provided")
	}

	armorReader := armor.NewReader(strings.NewReader(armoredData))
	ageReader, err := age.Decrypt(armorReader, identities...)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return io.ReadAll(ageReader)
}

// GetRecipientStrings extracts string representations of recipients for metadata
func GetRecipientStrings(recipients []age.Recipient) []string {
	var result []string
	for _, recipient := range recipients {
		// For X25519Recipient, we can get the string representation
		if x25519, ok := recipient.(*age.X25519Recipient); ok {
			result = append(result, x25519.String())
		}
		// For ScryptRecipient, we just note that passphrase was used
		if _, ok := recipient.(*age.ScryptRecipient); ok {
			result = append(result, "passphrase")
		}
	}
	return result
}

// HasPassphraseRecipient checks if any recipient is a passphrase recipient
func HasPassphraseRecipient(recipients []age.Recipient) bool {
	for _, recipient := range recipients {
		if _, ok := recipient.(*age.ScryptRecipient); ok {
			return true
		}
	}
	return false
}

// loadIdentitiesFromFile reads age identities from a file
func loadIdentitiesFromFile(filename string) ([]age.Identity, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return age.ParseIdentities(file)
}

// loadRecipientsFromFile reads age recipients from a file (one per line)
func loadRecipientsFromFile(filename string) ([]age.Recipient, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var recipients []age.Recipient
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		recipient, err := age.ParseX25519Recipient(line)
		if err != nil {
			return nil, fmt.Errorf("failed to parse recipient %s: %w", line, err)
		}

		recipients = append(recipients, recipient)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return recipients, nil
}
