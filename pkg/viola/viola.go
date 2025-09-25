// Package viola provides encrypted TOML configuration management with age encryption.
package viola

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/andreweick/viola/internal/walk"
	"github.com/andreweick/viola/pkg/enc"
)

// Options configures viola behavior
type Options struct {
	// Keys specifies sources for age identities and recipients
	Keys enc.KeySources

	// PrivatePrefix is the key prefix that triggers encryption (default: "private_")
	PrivatePrefix string

	// ShouldEncrypt overrides the default prefix-based encryption detection
	ShouldEncrypt func(path []string, key string, value any) bool

	// EmitASCIIQR controls whether QR codes are generated (default: true)
	EmitASCIIQR bool

	// QRCommentPrefix is the prefix for QR code comments (default: "# ")
	QRCommentPrefix string

	// Indent is the TOML indentation (default: "  ")
	Indent string
}

// setDefaults applies default values to options
func (o *Options) setDefaults() {
	if o.PrivatePrefix == "" {
		o.PrivatePrefix = "private_"
	}
	if o.QRCommentPrefix == "" {
		o.QRCommentPrefix = "# "
	}
	if o.Indent == "" {
		o.Indent = "  "
	}
	// EmitASCIIQR defaults to true, but we can't set that here since false is zero value
	// We'll handle this in the calling functions
}

// shouldEncryptField determines if a field should be encrypted
func (o Options) shouldEncryptField(path []string, key string, value any) bool {
	if o.ShouldEncrypt != nil {
		return o.ShouldEncrypt(path, key, value)
	}
	return strings.HasPrefix(key, o.PrivatePrefix)
}

// FieldMeta contains metadata about an encrypted field
type FieldMeta struct {
	// Path is the full path to the field (e.g., ["database", "private_password"])
	Path []string

	// WasEncrypted indicates if this field was encrypted
	WasEncrypted bool

	// Armored is the ASCII-armored ciphertext
	Armored string

	// ASCIIQR is the QR code as ASCII art (if enabled)
	ASCIIQR string

	// UsedRecipients lists the recipients used for encryption
	UsedRecipients []string

	// UsedPassphrase indicates if a passphrase was used
	UsedPassphrase bool
}

// Result contains the decrypted configuration and metadata
type Result struct {
	// Tree is the decrypted configuration as a map
	Tree map[string]any

	// Fields contains metadata for each field that was processed
	Fields []FieldMeta
}

// Load parses and decrypts a TOML configuration
func Load(data []byte, opts Options) (*Result, error) {
	opts.setDefaults()

	// Parse TOML
	var tree map[string]any
	if err := toml.Unmarshal(data, &tree); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	// Load identities for decryption
	identities, err := opts.Keys.LoadIdentities()
	if err != nil {
		return nil, fmt.Errorf("failed to load identities: %w", err)
	}

	var fields []FieldMeta

	// Walk the tree and decrypt encrypted fields
	decryptedTree := walk.Walk(tree, func(path []string, key string, value any) (any, bool) {
		// Check if this looks like an encrypted field
		if strValue, ok := value.(string); ok && isArmoredData(strValue) {
			// This is encrypted data, decrypt it
			decrypted, err := enc.Decrypt(strValue, identities)
			if err != nil {
				// If we can't decrypt, leave as-is and record the error
				// This allows for partial decryption or mixed files
				fields = append(fields, FieldMeta{
					Path:         append(path, key),
					WasEncrypted: true,
					Armored:      strValue,
				})
				return value, true
			}

			// Try to decode as JSON (for non-string values)
			var jsonValue any
			if err := json.Unmarshal(decrypted, &jsonValue); err != nil {
				// Not JSON, treat as string
				jsonValue = string(decrypted)
			}

			fields = append(fields, FieldMeta{
				Path:         append(path, key),
				WasEncrypted: true,
				Armored:      strValue,
			})

			return jsonValue, true
		}

		return value, true
	})

	return &Result{
		Tree:   decryptedTree.(map[string]any),
		Fields: fields,
	}, nil
}

// Save encrypts and serializes a configuration to TOML
func Save(tree any, opts Options) ([]byte, []FieldMeta, error) {
	opts.setDefaults()

	// Load recipients for encryption
	recipients, err := opts.Keys.LoadRecipients()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load recipients: %w", err)
	}

	if len(recipients) == 0 {
		return nil, nil, fmt.Errorf("no recipients available for encryption")
	}

	var fields []FieldMeta

	// Walk the tree and encrypt fields that should be encrypted
	encryptedTree := walk.Walk(tree, func(path []string, key string, value any) (any, bool) {
		if opts.shouldEncryptField(path, key, value) {
			// Skip if already encrypted
			if strValue, ok := value.(string); ok && isArmoredData(strValue) {
				// Already encrypted, record metadata and leave as-is
				fields = append(fields, FieldMeta{
					Path:           append(path, key),
					WasEncrypted:   true,
					Armored:        strValue,
					UsedRecipients: enc.GetRecipientStrings(recipients),
					UsedPassphrase: enc.HasPassphraseRecipient(recipients),
				})
				return value, true
			}

			// Encrypt the value
			var dataToEncrypt []byte
			if strValue, ok := value.(string); ok {
				// String value, encrypt directly
				dataToEncrypt = []byte(strValue)
			} else {
				// Non-string value, serialize to JSON first
				jsonData, err := json.Marshal(value)
				if err != nil {
					// If we can't serialize, leave as-is
					return value, true
				}
				dataToEncrypt = jsonData
			}

			encrypted, err := enc.Encrypt(dataToEncrypt, recipients)
			if err != nil {
				// If we can't encrypt, leave as-is
				return value, true
			}

			fields = append(fields, FieldMeta{
				Path:           append(path, key),
				WasEncrypted:   true,
				Armored:        encrypted,
				UsedRecipients: enc.GetRecipientStrings(recipients),
				UsedPassphrase: enc.HasPassphraseRecipient(recipients),
			})

			return encrypted, true
		}

		return value, true
	})

	// Serialize back to TOML
	tomlData, err := tomlMarshal(encryptedTree)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal TOML: %w", err)
	}

	return tomlData, fields, nil
}

// Transform loads a configuration, applies a transformation function, and saves it back
func Transform(data []byte, opts Options, transform func(tree any) error) ([]byte, []FieldMeta, error) {
	// Load the configuration
	result, err := Load(data, opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply the transformation
	if err := transform(result.Tree); err != nil {
		return nil, nil, fmt.Errorf("transformation failed: %w", err)
	}

	// Save the modified configuration
	return Save(result.Tree, opts)
}

// isArmoredData checks if a string looks like ASCII-armored age data
func isArmoredData(s string) bool {
	return strings.Contains(s, "-----BEGIN AGE ENCRYPTED FILE-----") &&
		strings.Contains(s, "-----END AGE ENCRYPTED FILE-----")
}

// tomlMarshal marshals a value to TOML bytes
func tomlMarshal(v any) ([]byte, error) {
	var buf strings.Builder
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}
