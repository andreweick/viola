# Viola Library API Documentation

This document provides comprehensive API documentation for the viola library, including all functions, types, and usage examples.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core API](#core-api)
  - [viola.Load](#violaload)
  - [viola.Save](#violasave)
  - [viola.Transform](#violatransform)
- [Types](#types)
  - [Options](#options)
  - [Result](#result)
  - [FieldMeta](#fieldmeta)
  - [KeySources](#keysources)
- [Encryption Helpers](#encryption-helpers)
  - [enc.Encrypt](#encencrypt)
  - [enc.Decrypt](#encdecrypt)
  - [enc.KeySources methods](#enckeysources-methods)
- [Tree Walking](#tree-walking)
  - [walk.Walk](#walkwalk)
  - [walk.FindFields](#walkfindfields)
- [Examples](#examples)
  - [Basic Usage](#basic-usage)
  - [Multiple Recipients](#multiple-recipients)
  - [Custom Encryption Rules](#custom-encryption-rules)
  - [Passphrase Support](#passphrase-support)
  - [File-based Keys](#file-based-keys)
  - [Error Handling](#error-handling)
- [Best Practices](#best-practices)

## Installation

```bash
go get github.com/andreweick/viola
```

## Quick Start

```go
package main

import (
    "log"

    "github.com/andreweick/viola/pkg/viola"
    "github.com/andreweick/viola/pkg/enc"
)

func main() {
    config := map[string]any{
        "app": "myapp",
        "private_secret": "my-secret-value",
    }

    opts := viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{"age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"},
            IdentitiesData: []string{"AGE-SECRET-KEY-1..."},
        },
    }

    // Encrypt and save
    encrypted, _, err := viola.Save(config, opts)
    if err != nil {
        log.Fatal(err)
    }

    // Load and decrypt
    result, err := viola.Load(encrypted, opts)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Decrypted: %+v", result.Tree)
}
```

## Core API

### viola.Load

Parses and decrypts a TOML configuration file.

```go
func Load(data []byte, opts Options) (*Result, error)
```

#### Parameters
- `data []byte`: Raw TOML data as bytes
- `opts Options`: Configuration options including keys and encryption settings

#### Returns
- `*Result`: Contains decrypted configuration tree and field metadata
- `error`: Any error that occurred during parsing or decryption

#### Example

```go
tomlData := []byte(`
app_name = "myapp"
private_api_key = """
-----BEGIN AGE ENCRYPTED FILE-----
YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBjVU12WXNoZmdxTXJWQ3pr
...
-----END AGE ENCRYPTED FILE-----
"""
`)

opts := viola.Options{
    Keys: enc.KeySources{
        IdentitiesData: []string{"AGE-SECRET-KEY-1..."},
    },
}

result, err := viola.Load(tomlData, opts)
if err != nil {
    return err
}

fmt.Printf("App name: %s\n", result.Tree["app_name"])
fmt.Printf("API key: %s\n", result.Tree["private_api_key"])
fmt.Printf("Processed %d encrypted fields\n", len(result.Fields))
```

#### Behavior
- Parses TOML into a `map[string]any` structure
- Detects ASCII-armored age blocks and attempts to decrypt them
- Non-decryptable fields remain as encrypted strings (graceful degradation)
- Returns metadata about all processed encrypted fields

### viola.Save

Encrypts specified fields and serializes configuration to TOML.

```go
func Save(tree any, opts Options) ([]byte, []FieldMeta, error)
```

#### Parameters
- `tree any`: Configuration data as `map[string]any` or compatible structure
- `opts Options`: Configuration options including keys and encryption settings

#### Returns
- `[]byte`: Encrypted TOML data
- `[]FieldMeta`: Metadata about each encrypted field
- `error`: Any error that occurred during encryption or serialization

#### Example

```go
config := map[string]any{
    "database": map[string]any{
        "host": "localhost",
        "port": 5432,
        "private_password": "secret123",
        "private_connection_url": "postgresql://user:pass@localhost/db",
    },
    "private_api_tokens": []string{"token1", "token2"},
    "private_config": map[string]any{
        "debug": true,
        "max_connections": 100,
    },
}

opts := viola.Options{
    Keys: enc.KeySources{
        Recipients: []string{
            "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
            "age1lggyhqrw2nlhcxprm67z43rta9q6mzraxn7mx18fp0wlkqfk54aq8ce9hz",
        },
    },
}

tomlData, fieldMeta, err := viola.Save(config, opts)
if err != nil {
    return err
}

fmt.Printf("Generated TOML:\n%s\n", tomlData)
fmt.Printf("Encrypted %d fields\n", len(fieldMeta))

for _, field := range fieldMeta {
    if field.WasEncrypted {
        fmt.Printf("- %s (recipients: %v)\n",
            strings.Join(field.Path, "."), field.UsedRecipients)
    }
}
```

#### Behavior
- Fields matching encryption criteria are encrypted in-place
- Already encrypted fields are left unchanged (idempotent)
- Non-string values are JSON-serialized before encryption
- Generates ASCII-armored age blocks compatible with the age tool

### viola.Transform

Loads a configuration, applies a transformation function, and saves the result.

```go
func Transform(data []byte, opts Options, transform func(tree any) error) ([]byte, []FieldMeta, error)
```

#### Parameters
- `data []byte`: Original TOML data
- `opts Options`: Configuration options
- `transform func(tree any) error`: Function that modifies the configuration tree

#### Returns
- `[]byte`: Modified and re-encrypted TOML data
- `[]FieldMeta`: Metadata about encrypted fields in the result
- `error`: Any error from loading, transformation, or saving

#### Example

```go
originalTOML := []byte(`
app_name = "myapp"
version = "1.0.0"
private_api_key = """
-----BEGIN AGE ENCRYPTED FILE-----
...encrypted data...
-----END AGE ENCRYPTED FILE-----
"""
`)

opts := viola.Options{
    Keys: enc.KeySources{
        Recipients: []string{"age1..."},
        IdentitiesData: []string{"AGE-SECRET-KEY-1..."},
    },
}

newTOML, meta, err := viola.Transform(originalTOML, opts, func(tree any) error {
    config := tree.(map[string]any)

    // Update version
    config["version"] = "2.0.0"

    // Add new encrypted field
    config["private_db_password"] = "new-secret"

    // Rotate API key
    config["private_api_key"] = "new-api-key-value"

    return nil
})

if err != nil {
    return err
}

fmt.Printf("Updated configuration:\n%s\n", newTOML)
```

#### Use Cases
- Configuration updates with secrets rotation
- Adding new encrypted fields to existing configs
- Bulk modifications of encrypted configurations
- Migration scripts for configuration changes

## Types

### Options

Configuration options for viola operations.

```go
type Options struct {
    Keys           enc.KeySources
    PrivatePrefix  string
    ShouldEncrypt  func(path []string, key string, value any) bool
    EmitASCIIQR    bool
    QRCommentPrefix string
    Indent         string
}
```

#### Fields

- **`Keys`**: Sources for age identities and recipients
- **`PrivatePrefix`**: Field name prefix that triggers encryption (default: `"private_"`)
- **`ShouldEncrypt`**: Optional custom function to determine encryption (overrides `PrivatePrefix`)
- **`EmitASCIIQR`**: Generate QR codes for encrypted fields (default: `true`, **not implemented**)
- **`QRCommentPrefix`**: Comment prefix for QR codes (default: `"# "`, **not implemented**)
- **`Indent`**: TOML indentation (default: `"  "`)

#### Example

```go
opts := viola.Options{
    Keys: enc.KeySources{
        Recipients: []string{"age1..."},
        IdentitiesData: []string{"AGE-SECRET-KEY-..."},
    },
    PrivatePrefix: "secret_", // Custom prefix
    ShouldEncrypt: func(path []string, key string, value any) bool {
        // Custom encryption logic
        return strings.Contains(key, "password") ||
               strings.Contains(key, "token") ||
               strings.HasSuffix(key, "_secret")
    },
}
```

### Result

Contains the result of a Load operation.

```go
type Result struct {
    Tree   map[string]any
    Fields []FieldMeta
}
```

#### Fields

- **`Tree`**: Decrypted configuration as a nested map structure
- **`Fields`**: Metadata about all processed encrypted fields

#### Example

```go
result, err := viola.Load(data, opts)
if err != nil {
    return err
}

// Access decrypted values
dbConfig := result.Tree["database"].(map[string]any)
password := dbConfig["private_password"].(string)

// Examine field metadata
for _, field := range result.Fields {
    if field.WasEncrypted {
        fmt.Printf("Field %s was encrypted with %d recipients\n",
            strings.Join(field.Path, "."), len(field.UsedRecipients))
    }
}
```

### FieldMeta

Metadata about an encrypted field.

```go
type FieldMeta struct {
    Path           []string
    WasEncrypted   bool
    Armored        string
    ASCIIQR        string
    UsedRecipients []string
    UsedPassphrase bool
}
```

#### Fields

- **`Path`**: Full path to the field (e.g., `["database", "private_password"]`)
- **`WasEncrypted`**: Whether this field was encrypted during processing
- **`Armored`**: ASCII-armored ciphertext
- **`ASCIIQR`**: QR code as ASCII art (**not implemented**)
- **`UsedRecipients`**: List of recipients used for encryption
- **`UsedPassphrase`**: Whether a passphrase recipient was used

#### Example

```go
_, fieldMeta, err := viola.Save(config, opts)
if err != nil {
    return err
}

for _, field := range fieldMeta {
    if field.WasEncrypted {
        path := strings.Join(field.Path, ".")
        fmt.Printf("Encrypted field: %s\n", path)
        fmt.Printf("  Recipients: %v\n", field.UsedRecipients)
        fmt.Printf("  Used passphrase: %v\n", field.UsedPassphrase)
        fmt.Printf("  Ciphertext length: %d bytes\n", len(field.Armored))
    }
}
```

### KeySources

Specifies sources for age identities and recipients.

```go
type KeySources struct {
    IdentitiesFile     string
    IdentitiesData     []string
    RecipientsFile     string
    Recipients         []string
    PassphraseProvider func() (string, error)
}
```

#### Fields

- **`IdentitiesFile`**: Path to file containing age private keys (for decryption)
- **`IdentitiesData`**: Age private keys as strings (for decryption)
- **`RecipientsFile`**: Path to file containing age public keys (for encryption)
- **`Recipients`**: Age public keys as strings (for encryption)
- **`PassphraseProvider`**: Function that returns passphrase for age-scrypt

#### Examples

**File-based keys:**
```go
keys := enc.KeySources{
    IdentitiesFile: "/home/user/.age/keys.txt",
    RecipientsFile: "/home/user/.age/recipients.txt",
}
```

**Explicit keys:**
```go
keys := enc.KeySources{
    Recipients: []string{
        "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
        "age1lggyhqrw2nlhcxprm67z43rta9q6mzraxn7mx18fp0wlkqfk54aq8ce9hz",
    },
    IdentitiesData: []string{
        "AGE-SECRET-KEY-1GFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPKQHJQHP",
    },
}
```

**With passphrase:**
```go
keys := enc.KeySources{
    Recipients: []string{"age1..."},
    PassphraseProvider: func() (string, error) {
        return os.Getenv("VIOLA_PASSPHRASE"), nil
    },
}
```

## Encryption Helpers

The `enc` package provides lower-level encryption utilities.

### enc.Encrypt

Encrypts data with age recipients.

```go
func Encrypt(data []byte, recipients []age.Recipient) (string, error)
```

#### Example

```go
recipients, err := testkeys.GetTestRecipients() // or use enc.KeySources.LoadRecipients()
if err != nil {
    return err
}

encrypted, err := enc.Encrypt([]byte("secret data"), recipients)
if err != nil {
    return err
}

fmt.Printf("Encrypted data:\n%s\n", encrypted)
```

### enc.Decrypt

Decrypts ASCII-armored age data.

```go
func Decrypt(armoredData string, identities []age.Identity) ([]byte, error)
```

#### Example

```go
identities, err := testkeys.GetTestIdentities() // or use enc.KeySources.LoadIdentities()
if err != nil {
    return err
}

decrypted, err := enc.Decrypt(encryptedData, identities)
if err != nil {
    return err
}

fmt.Printf("Decrypted: %s\n", decrypted)
```

### enc.KeySources methods

#### LoadIdentities

Loads age identities for decryption.

```go
func (ks KeySources) LoadIdentities() ([]age.Identity, error)
```

#### LoadRecipients

Loads age recipients for encryption.

```go
func (ks KeySources) LoadRecipients() ([]age.Recipient, error)
```

#### Example

```go
ks := enc.KeySources{
    RecipientsFile: "/path/to/recipients.txt",
    IdentitiesFile: "/path/to/keys.txt",
}

recipients, err := ks.LoadRecipients()
if err != nil {
    return err
}

identities, err := ks.LoadIdentities()
if err != nil {
    return err
}

// Use with enc.Encrypt/Decrypt
encrypted, err := enc.Encrypt(data, recipients)
decrypted, err := enc.Decrypt(encrypted, identities)
```

## Tree Walking

The `walk` package provides utilities for traversing TOML data structures.

### walk.Walk

Recursively walks through a data structure, calling a visitor function for each field.

```go
func Walk(data any, visit VisitFunc) any
```

#### VisitFunc

```go
type VisitFunc func(path []string, key string, value any) (newValue any, cont bool)
```

#### Example

```go
data := map[string]any{
    "user": "alice",
    "private_password": "secret",
    "config": map[string]any{
        "debug": true,
        "private_api_key": "key123",
    },
}

result := walk.Walk(data, func(path []string, key string, value any) (any, bool) {
    if strings.HasPrefix(key, "private_") {
        fmt.Printf("Found private field at %s.%s\n", strings.Join(path, "."), key)
        return "***REDACTED***", true
    }
    return value, true
})

fmt.Printf("Result: %+v\n", result)
```

### walk.FindFields

Finds fields matching a predicate function.

```go
func FindFields(data any, predicate func(path []string, key string, value any) bool) []FieldInfo
```

#### Example

```go
fields := walk.FindFields(data, func(path []string, key string, value any) bool {
    return strings.HasPrefix(key, "private_")
})

for _, field := range fields {
    fmt.Printf("Private field: %s = %v\n", field.GetFullPath(), field.Value)
}
```

## Examples

### Basic Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/andreweick/viola/pkg/viola"
    "github.com/andreweick/viola/pkg/enc"
)

func basicExample() {
    // Configuration with secrets
    config := map[string]any{
        "app_name": "myapp",
        "version": "1.0.0",
        "database": map[string]any{
            "host": "localhost",
            "port": 5432,
            "private_password": "db_secret_123",
        },
        "private_api_key": "api_secret_456",
    }

    // Setup encryption
    opts := viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{
                "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
            },
            IdentitiesData: []string{
                "AGE-SECRET-KEY-1GFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPKQHJQHP",
            },
        },
    }

    // Encrypt and save
    encryptedTOML, fieldMeta, err := viola.Save(config, opts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Encrypted TOML:\n%s\n", encryptedTOML)
    fmt.Printf("Encrypted %d fields\n", len(fieldMeta))

    // Load and decrypt
    result, err := viola.Load(encryptedTOML, opts)
    if err != nil {
        log.Fatal(err)
    }

    // Access decrypted data
    fmt.Printf("App: %s\n", result.Tree["app_name"])
    fmt.Printf("API Key: %s\n", result.Tree["private_api_key"])

    db := result.Tree["database"].(map[string]any)
    fmt.Printf("DB Password: %s\n", db["private_password"])
}
```

### Multiple Recipients

```go
func multipleRecipientsExample() {
    config := map[string]any{
        "private_secret": "shared-secret-value",
    }

    opts := viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{
                "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p", // Alice's key
                "age1lggyhqrw2nlhcxprm67z43rta9q6mzraxn7mx18fp0wlkqfk54aq8ce9hz", // Bob's key
                "age1zx5hjyp6xz7zl7z8l8l9l0l1l2l3l4l5l6l7l8l9l0l1l2l3l4l5l6l7", // Charlie's key
            },
        },
    }

    // Encrypt to all recipients
    encryptedTOML, meta, err := viola.Save(config, opts)
    if err != nil {
        log.Fatal(err)
    }

    // Any of the recipients can decrypt
    // Alice decrypts:
    aliceOpts := viola.Options{
        Keys: enc.KeySources{
            IdentitiesData: []string{"AGE-SECRET-KEY-1ALICE..."},
        },
    }

    result, err := viola.Load(encryptedTOML, aliceOpts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Alice decrypted: %s\n", result.Tree["private_secret"])

    // Bob can also decrypt with his key:
    bobOpts := viola.Options{
        Keys: enc.KeySources{
            IdentitiesData: []string{"AGE-SECRET-KEY-1BOB..."},
        },
    }

    result2, err := viola.Load(encryptedTOML, bobOpts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Bob decrypted: %s\n", result2.Tree["private_secret"])
}
```

### Custom Encryption Rules

```go
func customRulesExample() {
    config := map[string]any{
        "username": "alice",
        "user_password": "should-be-encrypted",    // matches "password"
        "api_token": "should-be-encrypted",        // matches "token"
        "database_secret": "should-be-encrypted",  // matches "secret"
        "public_config": "not-encrypted",          // doesn't match any rule
        "private_old_field": "encrypted-by-prefix", // matches private_ prefix
    }

    opts := viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{"age1..."},
            IdentitiesData: []string{"AGE-SECRET-KEY-..."},
        },
        // Custom encryption rules
        ShouldEncrypt: func(path []string, key string, value any) bool {
            // Encrypt if key contains any of these words
            sensitiveWords := []string{"password", "token", "secret", "key"}

            keyLower := strings.ToLower(key)
            for _, word := range sensitiveWords {
                if strings.Contains(keyLower, word) {
                    return true
                }
            }

            // Also encrypt private_ prefix (fallback to default behavior)
            return strings.HasPrefix(key, "private_")
        },
    }

    encryptedTOML, fieldMeta, err := viola.Save(config, opts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Custom rules encrypted %d fields\n", len(fieldMeta))

    // Show which fields were encrypted
    for _, field := range fieldMeta {
        if field.WasEncrypted {
            fmt.Printf("  - %s\n", strings.Join(field.Path, "."))
        }
    }

    fmt.Printf("\nEncrypted TOML:\n%s\n", encryptedTOML)
}
```

### Passphrase Support

```go
func passphraseExample() {
    config := map[string]any{
        "private_backup_key": "emergency-access-key",
    }

    opts := viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{
                "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p", // Primary key
            },
            PassphraseProvider: func() (string, error) {
                // In practice, you might read from environment or prompt user
                return "my-secure-backup-passphrase", nil
            },
        },
    }

    // Encrypt with both recipient and passphrase
    encryptedTOML, meta, err := viola.Save(config, opts)
    if err != nil {
        log.Fatal(err)
    }

    // Can decrypt with primary key
    primaryKeyOpts := viola.Options{
        Keys: enc.KeySources{
            IdentitiesData: []string{"AGE-SECRET-KEY-1..."},
        },
    }

    result1, err := viola.Load(encryptedTOML, primaryKeyOpts)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Decrypted with primary key: %s\n", result1.Tree["private_backup_key"])

    // Can also decrypt with passphrase (emergency access)
    passphraseOpts := viola.Options{
        Keys: enc.KeySources{
            PassphraseProvider: func() (string, error) {
                return "my-secure-backup-passphrase", nil
            },
        },
    }

    result2, err := viola.Load(encryptedTOML, passphraseOpts)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Decrypted with passphrase: %s\n", result2.Tree["private_backup_key"])
}
```

### File-based Keys

```go
func fileBasedKeysExample() {
    // Create recipients file
    recipientsContent := `# Team recipients
age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p  # Alice
age1lggyhqrw2nlhcxprm67z43rta9q6mzraxn7mx18fp0wlkqfk54aq8ce9hz  # Bob

# Emergency access
age1zx5hjyp6xz7zl7z8l8l9l0l1l2l3l4l5l6l7l8l9l0l1l2l3l4l5l6l7  # Emergency key
`
    err := os.WriteFile("/tmp/recipients.txt", []byte(recipientsContent), 0644)
    if err != nil {
        log.Fatal(err)
    }

    // Create identity file (normally you'd use age-keygen)
    identityContent := `# Alice's private key
AGE-SECRET-KEY-1GFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPYYSJZGFPKQHJQHP
`
    err = os.WriteFile("/tmp/identity.txt", []byte(identityContent), 0600)
    if err != nil {
        log.Fatal(err)
    }

    config := map[string]any{
        "private_team_secret": "shared among team members",
    }

    opts := viola.Options{
        Keys: enc.KeySources{
            RecipientsFile: "/tmp/recipients.txt",
            IdentitiesFile: "/tmp/identity.txt",
        },
    }

    // Encrypt to all recipients in file
    encryptedTOML, meta, err := viola.Save(config, opts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Encrypted with file-based keys:\n%s\n", encryptedTOML)

    // Decrypt using identity file
    result, err := viola.Load(encryptedTOML, opts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Decrypted: %s\n", result.Tree["private_team_secret"])

    // Clean up
    os.Remove("/tmp/recipients.txt")
    os.Remove("/tmp/identity.txt")
}
```

### Error Handling

```go
func errorHandlingExample() {
    config := map[string]any{
        "private_secret": "secret-value",
    }

    // Test various error conditions

    // 1. No recipients provided
    opts := viola.Options{
        Keys: enc.KeySources{
            // No recipients or identities
        },
    }

    _, _, err := viola.Save(config, opts)
    if err != nil {
        fmt.Printf("Expected error (no recipients): %v\n", err)
    }

    // 2. Invalid recipient
    opts = viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{"invalid-age-key"},
        },
    }

    _, _, err = viola.Save(config, opts)
    if err != nil {
        fmt.Printf("Expected error (invalid recipient): %v\n", err)
    }

    // 3. Malformed TOML
    malformedTOML := []byte(`
    invalid toml syntax [[[
    `)

    opts = viola.Options{
        Keys: enc.KeySources{
            IdentitiesData: []string{"AGE-SECRET-KEY-1..."},
        },
    }

    _, err = viola.Load(malformedTOML, opts)
    if err != nil {
        fmt.Printf("Expected error (malformed TOML): %v\n", err)
    }

    // 4. Unable to decrypt (wrong key)
    validTOML := []byte(`
    private_secret = """
    -----BEGIN AGE ENCRYPTED FILE-----
    YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBjVU12WXNoZmdxTXJWQ3pr
    ...
    -----END AGE ENCRYPTED FILE-----
    """
    `)

    opts = viola.Options{
        Keys: enc.KeySources{
            IdentitiesData: []string{"AGE-SECRET-KEY-1WRONGKEY..."},
        },
    }

    result, err := viola.Load(validTOML, opts)
    if err != nil {
        fmt.Printf("Error loading with wrong key: %v\n", err)
    } else {
        // Graceful degradation - field remains encrypted
        encrypted := result.Tree["private_secret"].(string)
        if strings.Contains(encrypted, "AGE ENCRYPTED FILE") {
            fmt.Printf("Field remained encrypted (graceful degradation)\n")
        }
    }

    // 5. Proper error handling pattern
    opts = viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{"age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"},
            IdentitiesData: []string{"AGE-SECRET-KEY-1VALIDKEY..."},
        },
    }

    encryptedTOML, meta, err := viola.Save(config, opts)
    if err != nil {
        log.Printf("Encryption failed: %v", err)
        return
    }

    result, err := viola.Load(encryptedTOML, opts)
    if err != nil {
        log.Printf("Decryption failed: %v", err)
        return
    }

    // Check if all expected fields were decrypted
    for _, field := range meta {
        if field.WasEncrypted {
            path := strings.Join(field.Path, ".")
            value, exists := getNestedValue(result.Tree, field.Path)
            if !exists {
                log.Printf("Warning: Field %s was not found in decrypted result", path)
            } else if strVal, ok := value.(string); ok && strings.Contains(strVal, "AGE ENCRYPTED FILE") {
                log.Printf("Warning: Field %s remains encrypted", path)
            } else {
                fmt.Printf("Successfully decrypted: %s\n", path)
            }
        }
    }
}

// Helper function to get nested values safely
func getNestedValue(data map[string]any, path []string) (any, bool) {
    current := data
    for i, key := range path {
        if i == len(path)-1 {
            value, exists := current[key]
            return value, exists
        }
        next, exists := current[key]
        if !exists {
            return nil, false
        }
        nextMap, ok := next.(map[string]any)
        if !ok {
            return nil, false
        }
        current = nextMap
    }
    return nil, false
}
```

## Best Practices

### 1. Key Management

```go
// ✅ Good: Use file-based keys for production
opts := viola.Options{
    Keys: enc.KeySources{
        RecipientsFile: "/etc/viola/recipients.txt",
        IdentitiesFile: "/home/user/.age/keys.txt",
    },
}

// ❌ Avoid: Hardcoded keys in source code
opts := viola.Options{
    Keys: enc.KeySources{
        Recipients: []string{"age1..."},     // Hardcoded - security risk
        IdentitiesData: []string{"AGE-SECRET-KEY-1..."}, // Never commit private keys!
    },
}
```

### 2. Error Handling

```go
// ✅ Good: Handle errors gracefully
result, err := viola.Load(data, opts)
if err != nil {
    log.Printf("Failed to load config: %v", err)
    return fmt.Errorf("configuration error: %w", err)
}

// Check for partially encrypted fields
for _, field := range result.Fields {
    if field.WasEncrypted {
        path := strings.Join(field.Path, ".")
        value, _ := getNestedValue(result.Tree, field.Path)
        if strVal, ok := value.(string); ok && strings.Contains(strVal, "AGE ENCRYPTED FILE") {
            log.Printf("Warning: Field %s could not be decrypted", path)
        }
    }
}
```

### 3. Configuration Validation

```go
// ✅ Good: Validate configuration after loading
result, err := viola.Load(data, opts)
if err != nil {
    return err
}

config := result.Tree

// Validate required fields exist and are decrypted
requiredSecrets := []string{"database.private_password", "private_api_key"}
for _, secretPath := range requiredSecrets {
    path := strings.Split(secretPath, ".")
    value, exists := getNestedValue(config, path)
    if !exists {
        return fmt.Errorf("required secret %s not found", secretPath)
    }
    if strVal, ok := value.(string); ok && strings.Contains(strVal, "AGE ENCRYPTED FILE") {
        return fmt.Errorf("required secret %s could not be decrypted", secretPath)
    }
}
```

### 4. Idempotent Operations

```go
// ✅ Good: Save operations are idempotent
config := loadExistingConfig()

// Modify configuration
config["new_field"] = "value"
config["private_new_secret"] = "secret"

// Save - existing encrypted fields won't be re-encrypted
newTOML, _, err := viola.Save(config, opts)
if err != nil {
    return err
}

// Multiple saves of the same data produce equivalent results
toml1, _, _ := viola.Save(config, opts)
toml2, _, _ := viola.Save(config, opts)

// Decrypt both and verify they produce the same plaintext
result1, _ := viola.Load(toml1, opts)
result2, _ := viola.Load(toml2, opts)
// result1.Tree should equal result2.Tree
```

### 5. Testing

```go
// ✅ Good: Use test keys for consistent testing
func TestConfigEncryption(t *testing.T) {
    opts := viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{testkeys.TestRecipient1},
            IdentitiesData: []string{testkeys.TestIdentity1},
        },
    }

    config := map[string]any{
        "app": "test",
        "private_secret": "test-value",
    }

    // Test encryption
    encrypted, meta, err := viola.Save(config, opts)
    require.NoError(t, err)
    require.Len(t, meta, 1)

    // Test decryption
    result, err := viola.Load(encrypted, opts)
    require.NoError(t, err)
    assert.Equal(t, "test-value", result.Tree["private_secret"])
}
```

### 6. Production Deployment

```go
// ✅ Good: Environment-based configuration
func loadProductionConfig() (*viola.Result, error) {
    recipientsFile := os.Getenv("VIOLA_RECIPIENTS_FILE")
    if recipientsFile == "" {
        recipientsFile = "/etc/viola/recipients.txt"
    }

    identityFile := os.Getenv("VIOLA_IDENTITY_FILE")
    if identityFile == "" {
        identityFile = "/var/secrets/viola/identity.txt"
    }

    opts := viola.Options{
        Keys: enc.KeySources{
            RecipientsFile: recipientsFile,
            IdentitiesFile: identityFile,
        },
    }

    configPath := os.Getenv("CONFIG_FILE")
    if configPath == "" {
        configPath = "/etc/myapp/config.toml"
    }

    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    return viola.Load(data, opts)
}
```

---

This completes the comprehensive API documentation for the viola library. The documentation covers all public APIs, types, usage patterns, and best practices for secure configuration management with age encryption.