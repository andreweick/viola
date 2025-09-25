# 🎭 viola

`viola` is a command-line utility for maintaining and creating **encrypted TOML configuration files** using [age](https://github.com/FiloSottile/age).

## Why Viola?

The name comes from Shakespeare's *Twelfth Night*. Viola is a character who **conceals her identity** in order to safely move between worlds. She appears in one form publicly, while her true self is hidden, revealed only to those who can see past the disguise.

That metaphor is exactly what `viola` does for your configs:
- **Public face** → easy to version-control, store, and share (encrypted TOML)
- **True identity** → the cleartext secrets, visible only when unlocked with your `age` key
- **Transformation** → like Viola's disguise, `viola` allows config files to take on a safe, portable form

## ✨ Features

- **Library-first design**: Use as a Go library or command-line tool
- Encrypts and decrypts TOML config files with **age**
- **Smart field detection**: Automatically encrypts fields starting with `private_`
- **Multiple recipients**: Encrypt to multiple age public keys
- **Passphrase support**: Optional age-scrypt passphrase recipients
- **Type preservation**: Handles strings, numbers, booleans, arrays, and objects
- **Idempotent saves**: Won't re-encrypt unchanged values
- Keeps secrets safe while enabling them to be committed to Git
- Designed for **immutable infrastructure**: generate once, deploy everywhere
- Lightweight: a single binary with no external dependencies

## 🔐 Acronym

`viola` also works as a backronym that reflects its purpose:

**V.I.O.L.A.**
- **V**ersatile
- **I**mmutable
- **O**bscured
- **L**oader for
- **A**rchives

## 🚀 Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/andreweick/viola.git
cd viola

# Build the CLI
just build

# Install locally (optional)
just install
```

### Using Go

```bash
go install github.com/andreweick/viola/cmd/viola@latest
```

## 📚 Usage

### As a Go Library

Import viola into your Go project:

```go
import "github.com/andreweick/viola/pkg/viola"
```

#### Basic Example

```go
package main

import (
    "fmt"
    "log"

    "github.com/andreweick/viola/pkg/viola"
    "github.com/andreweick/viola/pkg/enc"
)

func main() {
    // Your configuration with private fields
    config := map[string]any{
        "app_name": "myapp",
        "database": map[string]any{
            "host":                      "localhost",
            "port":                      5432,
            "private_password":          "secret123",
            "private_connection_string": "postgresql://user:secret@localhost/db",
        },
        "private_api_key": "super-secret-key",
    }

    // Set up encryption options
    opts := viola.Options{
        Keys: enc.KeySources{
            Recipients: []string{
                "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p", // your age public key
            },
        },
    }

    // Save with encryption (fields starting with "private_" will be encrypted)
    encryptedTOML, fieldMeta, err := viola.Save(config, opts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Encrypted TOML:\n%s\n", encryptedTOML)
    fmt.Printf("Encrypted %d fields\n", len(fieldMeta))

    // Load with decryption
    opts.Keys.IdentitiesData = []string{"AGE-SECRET-KEY-..."} // your age private key

    result, err := viola.Load(encryptedTOML, opts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Decrypted config: %+v\n", result.Tree)
}
```

#### Advanced Usage

```go
// Custom encryption rules
opts := viola.Options{
    Keys: enc.KeySources{
        Recipients: []string{"age1..."},
        IdentitiesData: []string{"AGE-SECRET-KEY-..."},
    },
    // Custom function to determine which fields to encrypt
    ShouldEncrypt: func(path []string, key string, value any) bool {
        return strings.Contains(key, "secret") || strings.Contains(key, "password")
    },
}

// Transform existing configuration
newTOML, meta, err := viola.Transform(existingTOML, opts, func(tree any) error {
    config := tree.(map[string]any)
    config["new_private_field"] = "new secret value"
    return nil
})
```

#### Key Sources

Viola supports multiple ways to specify age keys:

```go
opts := viola.Options{
    Keys: enc.KeySources{
        // From explicit recipients (for encryption)
        Recipients: []string{"age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"},

        // From explicit identities (for decryption)
        IdentitiesData: []string{"AGE-SECRET-KEY-1..."},

        // From files
        RecipientsFile: "/path/to/recipients.txt",
        IdentitiesFile: "/path/to/identities.txt",

        // With passphrase
        PassphraseProvider: func() (string, error) {
            return "my-secure-passphrase", nil
        },
    },
}
```

### API Reference

#### Core Functions

- **`viola.Load(data []byte, opts Options) (*Result, error)`**
  - Loads and decrypts a TOML configuration
  - Returns decrypted config tree and field metadata

- **`viola.Save(tree any, opts Options) ([]byte, []FieldMeta, error)`**
  - Saves a configuration with encryption
  - Returns encrypted TOML bytes and field metadata

- **`viola.Transform(data []byte, opts Options, transform func(any) error) ([]byte, []FieldMeta, error)`**
  - Loads, transforms, and saves a configuration
  - Convenient for making changes to encrypted configs

#### Key Types

```go
type Options struct {
    Keys           enc.KeySources // Age key sources
    PrivatePrefix  string         // Field prefix to encrypt (default: "private_")
    ShouldEncrypt  func(path []string, key string, value any) bool // Custom encryption logic
}

type Result struct {
    Tree   map[string]any // Decrypted configuration
    Fields []FieldMeta    // Metadata about encrypted fields
}

type FieldMeta struct {
    Path           []string // Field path (e.g. ["database", "private_password"])
    WasEncrypted   bool     // Whether this field was encrypted
    Armored        string   // ASCII-armored ciphertext
    UsedRecipients []string // Recipients used for encryption
    UsedPassphrase bool     // Whether passphrase was used
}
```

## 🔒 How It Works

### Field Detection

By default, viola encrypts any TOML field whose name starts with `private_`:

```toml
# Original configuration
username = "alice"
private_password = "secret123"

[database]
host = "localhost"
port = 5432
private_connection_string = "postgresql://user:pass@localhost/db"

[[servers]]
name = "prod"
private_api_key = "key123"
```

### Encrypted Output

After encryption, private fields become ASCII-armored age blocks:

```toml
username = "alice"

private_password = """
-----BEGIN AGE ENCRYPTED FILE-----
YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBxbWNhYzhmcXpwUTJzWnlm
...
-----END AGE ENCRYPTED FILE-----
"""

[database]
host = "localhost"
port = 5432

private_connection_string = """
-----BEGIN AGE ENCRYPTED FILE-----
YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBjVU12WXNoZmdxTXJWQ3pr
...
-----END AGE ENCRYPTED FILE-----
"""

[[servers]]
name = "prod"

private_api_key = """
-----BEGIN AGE ENCRYPTED FILE-----
YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBjVU12WXNoZmdxTXJWQ3pr
...
-----END AGE ENCRYPTED FILE-----
"""
```

### Type Preservation

Viola handles various data types intelligently:

- **Strings**: Encrypted directly as UTF-8 bytes
- **Numbers, booleans, arrays, objects**: Serialized to JSON before encryption
- **Nested structures**: Recursively processes all levels
- **Arrays of tables**: Encrypts private fields within each table

### Command Line Usage

The viola CLI provides four main commands for working with encrypted TOML files:

#### Encrypt Plain TOML Files

```bash
# Basic encryption with recipients file
viola encrypt config.toml -r recipients.txt -o config-encrypted.toml

# Encrypt to stdout
viola encrypt config.toml -r recipients.txt

# Use inline recipients
viola encrypt config.toml --recipients-inline "age1abc...,age1xyz..." -o encrypted.toml

# Custom field prefix for encryption
viola encrypt config.toml -r recipients.txt --private-prefix "secret_"

# Dry run to see what would be encrypted
viola encrypt config.toml -r recipients.txt --dry-run

# Show encryption statistics
viola encrypt config.toml -r recipients.txt -o encrypted.toml --stats

# Overwrite existing output file
viola encrypt config.toml -r recipients.txt -o existing.toml --force
```

#### Read and Decrypt Files

```bash
# Basic decryption with identity file
viola read config.toml -i ~/.age/keys.txt

# Decrypt and output as JSON
viola read config.toml -i identity.key -o json

# Show only encrypted fields
viola read config.toml -i identity.key --private-only

# Show only non-encrypted fields
viola read config.toml --public-only

# Extract specific path
viola read config.toml -i identity.key --path "database.private_password"

# Use inline identity key (testing only)
viola read config.toml -k "AGE-SECRET-KEY-..."

# Use passphrase authentication
viola read config.toml --passphrase

# Show raw encrypted values without decryption
viola read config.toml --raw
```

#### Inspect File Metadata

```bash
# Show basic info about encrypted fields
viola inspect config.toml

# List all encrypted field paths
viola inspect config.toml --fields

# Show recipients for each encrypted field
viola inspect config.toml --recipients

# Display encryption statistics
viola inspect config.toml --stats

# Show QR code for specific field
viola inspect config.toml --qr "api.private_key"
```

#### Verify File Integrity

```bash
# Verify TOML format is valid
viola verify config.toml --check-format

# Verify armor blocks are well-formed
viola verify config.toml --check-armor

# Verify all fields can be decrypted with provided identity
viola verify config.toml --check-all -i identity.key

# Run all verification checks
viola verify config.toml --check-all -i identity.key
```

## 🏗️ Development

### Prerequisites

- Go 1.21 or later
- [just](https://github.com/casey/just) for task automation

### Building

```bash
# Download dependencies
just deps

# Build the CLI
just build

# Run with arguments
just run read config.toml

# Run tests
just test

# Clean build artifacts
just clean
```

### Project Structure

```
viola/
├── cmd/viola/          # CLI application
│   └── main.go         # Entry point and command definitions
├── pkg/
│   ├── viola/          # Main library API
│   │   ├── viola.go    # Load, Save, Transform functions
│   │   └── viola_test.go
│   └── enc/            # Age encryption helpers
│       ├── enc.go      # KeySources, Encrypt, Decrypt
│       └── enc_test.go
├── internal/
│   ├── testkeys/       # Test key constants
│   │   ├── keys.go     # Hardcoded age keys for testing
│   │   └── keys_test.go
│   └── walk/           # TOML tree traversal
│       ├── walk.go     # Generic tree walker
│       └── walk_test.go
├── docs/               # Documentation
├── justfile            # Task automation
├── go.mod              # Go module definition
└── README.md           # This file
```
## 📋 Complete Command Reference

### viola encrypt

Encrypt a plain TOML configuration file.

```
viola encrypt [options] <file>
```

#### Options

| Flag | Alias | Type | Description |
|------|-------|------|-------------|
| `--recipients` | `-r` | string[] | Path to recipients file containing age public keys (can be specified multiple times) |
| `--recipients-inline` | | string | Comma-separated age public keys for encryption |
| `--output` | `-o` | string | Output file path (default: stdout) |
| `--force` | `-f` | bool | Overwrite output file if it exists |
| `--private-prefix` | | string | Prefix for fields to encrypt (default: `private_`) |
| `--dry-run` | | bool | Show what would be encrypted without doing it |
| `--stats` | | bool | Show encryption statistics |
| `--quiet` | `-q` | bool | Suppress non-essential output |
| `--verbose` | `-v` | bool | Show detailed encryption info |

### viola read

Read and decrypt TOML configuration files.

```
viola read [options] <file>
```

#### Options

| Flag | Alias | Type | Description |
|------|-------|------|-------------|
| `--identity` | `-i` | string[] | Path to age identity file (can be specified multiple times) |
| `--key` | `-k` | string | Inline age identity key (insecure, for testing only) |
| `--passphrase` | | bool | Prompt for passphrase interactively |
| `--passphrase-file` | | string | Read passphrase from file (first line) |
| `--passphrase-env` | | string | Read passphrase from environment variable |
| `--output` | `-o` | string | Output format: `toml`, `json`, `yaml`, `env`, `flat` (default: `toml`) |
| `--raw` | | bool | Show raw encrypted values without decrypting |
| `--path` | | string | Extract specific path (dot notation: `server.private_key`) |
| `--private-only` | | bool | Show only encrypted fields |
| `--public-only` | | bool | Show only non-encrypted fields |
| `--show-qr` | | bool | Display QR codes alongside values (not implemented) |
| `--no-color` | | bool | Disable colored output |
| `--quiet` | `-q` | bool | Suppress non-essential output |
| `--verbose` | `-v` | bool | Show detailed decryption info |

### viola inspect

Inspect encrypted file metadata without decrypting.

```
viola inspect [options] <file>
```

#### Options

| Flag | Description |
|------|-------------|
| `--fields` | List all encrypted field paths |
| `--recipients` | Show recipients for each field |
| `--stats` | Show encryption statistics |
| `--qr` | Display QR for specific encrypted field |
| `--check-recipient` | Check if recipient can decrypt |

### viola verify

Verify file integrity and decryptability.

```
viola verify [options] <file>
```

#### Options

| Flag | Alias | Description |
|------|-------|-------------|
| `--identity` | `-i` | Identity to verify against (can be specified multiple times) |
| `--check-all` | | Verify all encrypted fields are decryptable |
| `--check-format` | | Verify TOML format is valid |
| `--check-armor` | | Verify armor blocks are valid |

### Global Options

These options are available for all commands:

| Flag | Alias | Description |
|------|-------|-------------|
| `--help` | `-h` | Show help |
| `--version` | | Show version information |

### Examples

#### Basic Usage

```bash
# Generate age key pair
age-keygen > identity.key
echo "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p" > recipients.txt

# Encrypt plain configuration
viola encrypt plain-config.toml --recipients recipients.txt -o encrypted-config.toml

# Read encrypted configuration
viola read encrypted-config.toml --identity identity.key

# Inspect without decrypting
viola inspect encrypted-config.toml --fields --recipients

# Verify integrity
viola verify encrypted-config.toml --check-all --identity identity.key
```

#### Advanced Usage

```bash
# Encryption workflows
viola encrypt config.toml -r recipients.txt --dry-run        # Preview what gets encrypted
viola encrypt config.toml -r alice.txt -r bob.txt -o enc.toml # Multiple recipients
viola encrypt config.toml --recipients-inline "age1abc..." -o enc.toml # Inline recipients
viola encrypt config.toml -r recipients.txt --private-prefix "secret_" # Custom prefix
viola encrypt config.toml -r recipients.txt -o enc.toml --stats --verbose # With stats

# Multiple output formats
viola read config.toml -i key.txt -o json > config.json
viola read config.toml -i key.txt -o yaml > config.yaml
viola read config.toml -i key.txt -o env > config.env

# Field filtering
viola read config.toml -i key.txt --private-only    # Only show encrypted fields
viola read config.toml --public-only                # Only show non-encrypted fields

# Path extraction
viola read config.toml -i key.txt --path "database.private_password"
viola read config.toml -i key.txt --path "api.private_key" -o json

# Passphrase authentication
viola read config.toml --passphrase                 # Interactive prompt
viola read config.toml --passphrase-env VIOLA_PASS  # From environment
viola read config.toml --passphrase-file pass.txt   # From file

# Multiple identities
viola read config.toml -i alice.key -i bob.key -i charlie.key

# Verification workflows
viola verify config.toml --check-format             # Just format
viola verify config.toml --check-armor              # Just armor blocks
viola verify config.toml --check-all -i identity.key # Full verification
```

## 🎭 Acknowledgments

- Shakespeare's *Twelfth Night* for the inspiration
- [age](https://github.com/FiloSottile/age) for the encryption
- [Charm](https://charm.sh/) for beautiful CLI components
