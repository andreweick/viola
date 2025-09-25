# Simplified Implementation Plan for Viola

## Overview
This is a simplified implementation plan that excludes QR code generation and identity management features. The system will rely on environment variables for private keys.

## Phase 1: Core Encryption Library (pkg/enc)

### Goals
- Create age encryption/decryption helpers
- Implement ASCII armor handling
- Support reading private key from environment variable
- Add recipient string parsing
- Add passphrase support (optional, for future use)

### Key Functions
- `Encrypt(plaintext []byte, recipients []string) ([]byte, error)`
- `Decrypt(ciphertext []byte, privateKey string) ([]byte, error)`
- `Armor(data []byte) string`
- `Dearmor(armored string) ([]byte, error)`

## Phase 2: TOML Processing (pkg/viola)

### Goals
- Implement TOML tree walker (internal/walk)
- Create Load, Save, and Transform functions

### Load Function
- Detects `private_` prefix fields at any depth
- Decrypts armored blocks using env key
- Deserializes JSON for complex types
- Returns decrypted tree and field metadata

### Save Function
- Encrypts `private_` fields
- Serializes complex types to JSON first
- Produces ASCII-armored blocks
- Maintains idempotency (no re-encrypt if unchanged)
- Returns TOML bytes and field metadata

### Transform Function
- Atomic load → modify → save operation
- Preserves encryption for unchanged fields

## Phase 3: CLI Implementation

### Commands to Implement
1. **read** - Decrypt and display TOML
2. **save** - Encrypt and save TOML
3. **edit/transform** - In-place modification
4. **inspect** - View metadata without decrypting
5. **verify** - Check file integrity

### Output Formats
- `toml` (default)
- `json`
- `yaml`
- `env`
- `flat` (dot notation)

### Key Handling
```bash
export AGE_PRIVATE_KEY="AGE-SECRET-KEY-..."
viola read config.toml
```

## Phase 4: Field Metadata Tracking

### FieldMeta Structure
```go
type FieldMeta struct {
    Path           []string  // Field path in tree
    WasEncrypted   bool      // Was already encrypted
    Armored        string    // ASCII-armored payload
    UsedRecipients []string  // Recipients used
    UsedPassphrase bool      // Passphrase was used
    // Note: ASCIIIQR field excluded (will be empty string)
}
```

## Phase 5: Testing

### Test Coverage Required
1. **Unit Tests**
   - Encryption/decryption round-trips
   - All TOML data types (string, int, bool, array, table)
   - Nested `private_` field detection

2. **Integration Tests**
   - Full workflow tests
   - Multiple encrypted fields
   - Complex nested structures

3. **Edge Cases**
   - Malformed armor blocks
   - Missing private keys
   - Invalid TOML structure
   - Idempotency verification

## Security Checklist

- ✅ Use age for all cryptography
- ✅ Validate armor blocks before decryption
- ✅ Never log plaintext values
- ✅ Clear sensitive buffers after use
- ✅ Include clear error messages with field paths
- ✅ Ensure idempotent encryption
- ✅ Reject invalid recipient formats
- ✅ Handle decryption failures gracefully

## Excluded Features

### Not Implementing
- QR code generation (internal/qr package)
- Identity management/generation
- XDG_CONFIG_HOME key storage
- Self-encryption fallback
- Automatic age key generation

### Simplifications
- Single environment variable for private key
- No filesystem key management
- No visual QR output (metadata field will be empty)
- No key rotation commands initially

## Dependencies Required

```go
// go.mod additions needed
require (
    filippo.io/age v1.1.1
    github.com/BurntSushi/toml v1.3.2
    // Existing: urfave/cli, charmbracelet/lipgloss
)
```

## Example Usage

```bash
# Set private key
export AGE_PRIVATE_KEY="AGE-SECRET-KEY-1234..."

# Read encrypted config
viola read config.toml

# Save with encryption
viola save config.toml --recipient age1abc...

# Transform in place
viola edit config.toml --set server.private_password=newsecret

# Inspect without decrypting
viola inspect config.toml --fields
```

## Next Steps
1. Create package structure
2. Implement core encryption helpers
3. Add TOML processing logic
4. Wire up CLI commands
5. Add comprehensive tests