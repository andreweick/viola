// Package testkeys provides pre-generated age keys for testing purposes.
// WARNING: These keys are for testing only and should NEVER be used in production.
package testkeys

import (
	"bytes"
	"io"

	"filippo.io/age"
	"filippo.io/age/armor"
)

// Test identities (private keys) - NEVER use in production
const (
	TestIdentity1 = "AGE-SECRET-KEY-19MT3XUES3V0X2HYXSF7JLXFMRHZCKNUCZSHRXTR0R5HZ03D4CQDSXZ0Q0J"
	TestIdentity2 = "AGE-SECRET-KEY-10VFRLJM2QF6W85WYLT858348MJYVF3CP2C49K5A4DKY4N5FVUEPSQ8Y2KE"
	TestIdentity3 = "AGE-SECRET-KEY-1ZQPCU77PSTHUAQRG9ZP3KQKYV7Y46YDWYKF872GGW7Y8WV87Z07SKFJY77"
)

// Corresponding recipients (public keys)
const (
	TestRecipient1 = "age1nfgr67hmk5pynqhqwaqa9y0zkppr0dl9s2stdm4wjq3cn3nx4g2s5qvkrk"
	TestRecipient2 = "age1ukl0hf5sgpu0kvedsxdgn0krcx3vh3r9fmc38dpvqz8tvft3dskspweh55"
	TestRecipient3 = "age1dx0v6a24uaf7af0l60zw7zw6390es3g0kzc7jnzyqyeee4rsxumssxhvl6"
)

// Test passphrase for age-scrypt - NEVER use in production
const TestPassphrase = "test-passphrase-never-use-in-production-12345"

// GetTestIdentities returns parsed test identities for encryption/decryption
func GetTestIdentities() ([]age.Identity, error) {
	var identities []age.Identity

	id1, err := age.ParseX25519Identity(TestIdentity1)
	if err != nil {
		return nil, err
	}
	identities = append(identities, id1)

	id2, err := age.ParseX25519Identity(TestIdentity2)
	if err != nil {
		return nil, err
	}
	identities = append(identities, id2)

	id3, err := age.ParseX25519Identity(TestIdentity3)
	if err != nil {
		return nil, err
	}
	identities = append(identities, id3)

	return identities, nil
}

// GetTestRecipients returns parsed test recipients for encryption
func GetTestRecipients() ([]age.Recipient, error) {
	var recipients []age.Recipient

	r1, err := age.ParseX25519Recipient(TestRecipient1)
	if err != nil {
		return nil, err
	}
	recipients = append(recipients, r1)

	r2, err := age.ParseX25519Recipient(TestRecipient2)
	if err != nil {
		return nil, err
	}
	recipients = append(recipients, r2)

	r3, err := age.ParseX25519Recipient(TestRecipient3)
	if err != nil {
		return nil, err
	}
	recipients = append(recipients, r3)

	return recipients, nil
}

// EncryptTestData encrypts data with test recipients for use in golden tests
func EncryptTestData(data []byte) (string, error) {
	recipients, err := GetTestRecipients()
	if err != nil {
		return "", err
	}

	// Encrypt the data
	var buf bytes.Buffer
	armorWriter := armor.NewWriter(&buf)
	ageWriter, err := age.Encrypt(armorWriter, recipients...)
	if err != nil {
		return "", err
	}

	if _, err := ageWriter.Write(data); err != nil {
		return "", err
	}

	if err := ageWriter.Close(); err != nil {
		return "", err
	}

	if err := armorWriter.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// DecryptTestData decrypts armored data using test identities
func DecryptTestData(armoredData string) ([]byte, error) {
	identities, err := GetTestIdentities()
	if err != nil {
		return nil, err
	}

	// Decrypt the data
	armorReader := armor.NewReader(bytes.NewReader([]byte(armoredData)))
	ageReader, err := age.Decrypt(armorReader, identities...)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(ageReader)
}
