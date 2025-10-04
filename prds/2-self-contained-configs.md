# PRD: Self-Contained Config Files with Field Setting

**Status**: Draft
**Created**: 2025-10-04
**GitHub Issue**: [#2](https://github.com/andreweick/viola/issues/2)
**Owner**: TBD
**Priority**: High

---

## Problem Statement

Currently, viola has two major workflow pain points that make it cumbersome to use in practice:

### Problem 1: Adding Encrypted Fields Requires Multi-Step Workflow

To add or update a single encrypted field in a viola TOML file, users must:

1. Decrypt the entire file: `viola read config.toml -i identity.key > decrypted.toml`
2. Manually edit the decrypted file in a text editor
3. Re-encrypt the entire file: `viola encrypt decrypted.toml -r recipients.txt -o config.toml`
4. Remember to delete the decrypted file: `rm decrypted.toml`

This is:
- **Error-prone**: Easy to forget step 4, leaving secrets in plaintext
- **Slow**: Multiple commands for a simple field update
- **Inconvenient**: Can't quickly add a field without context switching

### Problem 2: Files Require External Dependencies

Viola TOML files currently require external recipient files or CLI flags to decrypt:

```bash
# Need to remember/specify recipients every time
viola encrypt config.toml -r recipients.txt -o encrypted.toml
viola read encrypted.toml -i identity.key

# Or use inline recipients (hard to remember)
viola encrypt config.toml --recipients-inline "age1ql3z..." -o encrypted.toml
```

This creates several issues:
- **Not portable**: Can't share just the encrypted file
- **Requires documentation**: Users need to know which recipients file to use
- **Bootstrap friction**: Hard to use encrypted configs in new projects
- **Lost context**: Looking at a file doesn't show who can decrypt it

### User Impact

**Developer Pain Points**:
- "I just want to add one API key to my config without decrypting the whole thing"
- "I shared the encrypted config but forgot to share the recipients file"
- "Which recipients file do I use for this config again?"
- "I'm scared I'll forget to delete the decrypted file and commit secrets"

---

## Solution Overview

Add two complementary features that make viola files self-contained and easy to update:

### Feature 1: `viola set` Command

A single command to add or update encrypted fields directly:

```bash
# Add/update a single encrypted field
viola set config.toml database.private_password "new-secret"

# No decrypt-edit-encrypt dance needed
# No intermediate files to clean up
# Works on encrypted files directly
```

**Key Benefits**:
- One-line field updates
- No intermediate plaintext files
- Safe by default (no cleanup needed)
- Fast and convenient

### Feature 2: Embedded Recipients in `[_viola]` Section

Store recipients directly in the TOML file:

```toml
[_viola]
recipients = [
    "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
    "age1another..."
]

# Regular configuration below
database = "localhost"
private_password = "-----BEGIN AGE ENCRYPTED FILE-----..."
```

**Key Benefits**:
- Self-documenting (shows who can decrypt)
- Portable (file contains everything needed)
- No CLI flags needed for decryption
- Easy to bootstrap in new projects

### How They Work Together

```bash
# Initial setup: Create config with embedded recipients
viola encrypt config.toml --recipients-inline "age1..." -o encrypted.toml
# Recipients are auto-embedded in [_viola] section

# Later: Add a field (uses embedded recipients automatically)
viola set encrypted.toml private_api_key "secret-key"
# No need to specify recipients - they're in the file!

# Share the file: Others can decrypt without extra info
viola read encrypted.toml -i their-identity.key
# No recipients file needed - it's self-contained!
```

---

## Success Criteria

### Must Have
- [ ] `viola set` command works to add/update encrypted fields
- [ ] Fields with `private_` prefix are automatically encrypted
- [ ] `viola set` reads embedded recipients from `[_viola]` section
- [ ] `encrypt` command embeds recipients in `[_viola]` section
- [ ] `read` command uses embedded recipients when no CLI recipients provided
- [ ] Backward compatibility with files without `[_viola]` section
- [ ] CLI recipients take precedence over embedded recipients
- [ ] Documentation with clear examples

### Should Have
- [ ] `viola set` supports nested paths with dot notation (e.g., `db.auth.private_key`)
- [ ] `inspect` command shows embedded recipients
- [ ] Support for different value types (string, int, bool, json)
- [ ] Warning if trying to set field but no recipients available
- [ ] Examples showing self-contained workflow

### Could Have
- [ ] `viola unset` command to remove fields
- [ ] `viola get` command to read specific field
- [ ] Batch updates (`--set field1=val1 --set field2=val2`)
- [ ] Interactive mode for setting values

---

## User Stories

### Story 1: Quick Field Addition
**As a** developer
**I want** to add an encrypted field to my config with one command
**So that** I don't have to decrypt, edit, and re-encrypt the whole file

**Acceptance Criteria**:
- Run single `viola set` command
- Field is encrypted and added to file
- No intermediate files created
- Existing fields remain unchanged

**Example**:
```bash
viola set config.toml private_api_key "sk-abc123"
# Done! No cleanup needed.
```

### Story 2: Self-Contained Config for New Project
**As a** developer
**I want** to copy an encrypted config to a new project and use it immediately
**So that** I don't need to track down recipient files or remember CLI flags

**Acceptance Criteria**:
- Copy just the TOML file to new project
- Decrypt works with only identity file
- No external recipient files needed
- File shows who can decrypt it

**Example**:
```bash
# In new project
cp ~/configs/app.encrypted.toml ./config.toml
viola read config.toml -i ~/.age/keys.txt
# Just works! No recipients needed.
```

### Story 3: Sharing Config with Team
**As a** team lead
**I want** to share an encrypted config that shows who can decrypt it
**So that** team members know if they have access and who to ask if not

**Acceptance Criteria**:
- Open TOML file and see recipients in `[_viola]` section
- Recognize which age keys have access
- Know whether you can decrypt before trying

**Example**:
```toml
[_viola]
recipients = [
    "age1alice...",  # Alice's key
    "age1bob...",    # Bob's key
]
```

### Story 4: Bootstrap Development Environment
**As a** developer
**I want** to set up encrypted configs in a new environment quickly
**So that** I can start development without manual configuration

**Acceptance Criteria**:
- Clone repo with encrypted config
- Add local secrets with `viola set`
- No need to find/configure recipients
- Works immediately

**Example**:
```bash
git clone myproject
cd myproject
# Add my local DB password
viola set config.toml database.private_password "local-dev-pass"
# Start developing
```

---

## Technical Design

### Current State

**Library Support** (`pkg/viola`):
- ✅ `Load()` function decrypts TOML files
- ✅ `Save()` function encrypts TOML files
- ✅ `Transform()` function for decrypt-modify-encrypt workflow
- ✅ Support for multiple recipients
- ✅ Idempotent saves (won't re-encrypt unchanged values)

**CLI Support** (`cmd/viola/main.go`):
- ✅ `encrypt` command with recipient specification
- ✅ `read` command with identity specification
- ✅ `inspect` command for metadata
- ❌ No `set` command for direct field updates
- ❌ No embedded recipients support

### Proposed Changes

#### Change 1: Add `[_viola]` Metadata Section Support

**Purpose**: Store recipients and future metadata in the TOML file itself.

**Format**:
```toml
[_viola]
recipients = ["age1...", "age2..."]
# Future: version, cipher, etc.

# User's configuration
[database]
host = "localhost"
private_password = "-----BEGIN AGE ENCRYPTED FILE-----..."
```

**Implementation**:
1. Modify `Save()` in `pkg/viola/viola.go`:
   - Before saving, inject `[_viola]` section with recipients
   - Extract recipients from `Options.Keys.Recipients`
   - Place at top of TOML output

2. Modify `Load()` in `pkg/viola/viola.go`:
   - After parsing TOML, extract `[_viola]` section
   - Read recipients from `[_viola].recipients` array
   - Merge with `Options.Keys.Recipients` (CLI takes precedence)
   - Remove `[_viola]` section from returned tree (don't expose to user)

3. Update `encryptAction()` in `cmd/viola/main.go`:
   - By default, embed recipients into `[_viola]` section
   - Add `--no-embed-recipients` flag to disable (for backward compat)

4. Update `readAction()` in `cmd/viola/main.go`:
   - Use embedded recipients if no CLI recipients provided
   - Show info message about using embedded recipients (if verbose)

**Backward Compatibility**:
- Files without `[_viola]` section work as before
- CLI recipients always take precedence
- No breaking changes to API or CLI

#### Change 2: Add `viola set` Command

**Purpose**: Add or update encrypted fields without full decrypt-edit-encrypt cycle.

**Command Signature**:
```bash
viola set <file> <path> <value> [flags]

Flags:
  -i, --identity string       Age identity file for decryption
  --type string              Value type: string, int, bool, json (default "string")
  -o, --output string        Output file (default: overwrite input)
  --dry-run                  Show what would change without saving
  -v, --verbose              Show detailed operation info
```

**Examples**:
```bash
# Set top-level field
viola set config.toml private_api_key "secret"

# Set nested field (dot notation)
viola set config.toml database.private_password "newpass"

# Set with type
viola set config.toml private_port 5432 --type int
viola set config.toml private_enabled true --type bool
viola set config.toml private_config '{"timeout": 30}' --type json

# Dry run
viola set config.toml private_key "new" --dry-run
```

**Implementation**:

1. Create `setCommand()` in `cmd/viola/main.go`:
```go
func setCommand() *cli.Command {
    return &cli.Command{
        Name:  "set",
        Usage: "Set an encrypted field in a TOML file",
        Flags: []cli.Flag{
            &cli.StringSliceFlag{
                Name:    "identity",
                Aliases: []string{"i"},
                Usage:   "Path to age identity file for decryption",
            },
            &cli.StringFlag{
                Name:  "type",
                Usage: "Value type: string, int, bool, json",
                Value: "string",
            },
            &cli.StringFlag{
                Name:    "output",
                Aliases: []string{"o"},
                Usage:   "Output file path (default: overwrite input)",
            },
            &cli.BoolFlag{
                Name:  "dry-run",
                Usage: "Show what would change without saving",
            },
            &cli.BoolFlag{
                Name:    "verbose",
                Aliases: []string{"v"},
                Usage:   "Show detailed operation info",
            },
        },
        Action: setAction,
    }
}
```

2. Implement `setAction()`:
```go
func setAction(c *cli.Context) error {
    // Validate args
    if c.NArg() != 3 {
        return cli.NewExitError("Usage: viola set <file> <path> <value>", 1)
    }

    filename := c.Args().Get(0)
    path := c.Args().Get(1)
    value := c.Args().Get(2)

    // Read file
    data, err := readFile(filename)
    if err != nil {
        return cli.NewExitError(fmt.Sprintf("Error reading file: %v", err), 1)
    }

    // Build key sources (including embedded recipients)
    keySources, err := buildKeySourcesWithEmbedded(c, data)
    if err != nil {
        return cli.NewExitError(fmt.Sprintf("Error loading keys: %v", err), 1)
    }

    // Parse value according to type
    parsedValue, err := parseValue(value, c.String("type"))
    if err != nil {
        return cli.NewExitError(fmt.Sprintf("Error parsing value: %v", err), 1)
    }

    // Use Transform to update field
    opts := viola.Options{Keys: keySources}
    updatedData, _, err := viola.Transform(data, opts, func(tree any) error {
        return setNestedField(tree, path, parsedValue)
    })
    if err != nil {
        return cli.NewExitError(fmt.Sprintf("Error updating field: %v", err), 1)
    }

    // Handle dry-run
    if c.Bool("dry-run") {
        fmt.Printf("Would update %s to:\n%s\n", path, parsedValue)
        return nil
    }

    // Write output
    outputFile := c.String("output")
    if outputFile == "" {
        outputFile = filename
    }

    if err := os.WriteFile(outputFile, updatedData, 0644); err != nil {
        return cli.NewExitError(fmt.Sprintf("Error writing file: %v", err), 1)
    }

    if c.Bool("verbose") {
        fmt.Printf("✓ Updated %s in %s\n", path, outputFile)
    }

    return nil
}
```

3. Add helper functions:
```go
// buildKeySourcesWithEmbedded loads keys from CLI and embedded recipients
func buildKeySourcesWithEmbedded(c *cli.Context, fileData []byte) (enc.KeySources, error) {
    // First get CLI-provided keys
    ks, err := buildKeySources(c)
    if err != nil {
        return ks, err
    }

    // Then try to load embedded recipients
    embeddedRecipients, err := extractEmbeddedRecipients(fileData)
    if err == nil && len(embeddedRecipients) > 0 {
        // Merge (CLI takes precedence, so add embedded as fallback)
        if len(ks.Recipients) == 0 && ks.RecipientsFile == "" {
            ks.Recipients = embeddedRecipients
        }
    }

    return ks, nil
}

// extractEmbeddedRecipients parses TOML and extracts [_viola].recipients
func extractEmbeddedRecipients(data []byte) ([]string, error) {
    var tree map[string]any
    if err := toml.Unmarshal(data, &tree); err != nil {
        return nil, err
    }

    viola, ok := tree["_viola"].(map[string]any)
    if !ok {
        return nil, fmt.Errorf("no _viola section")
    }

    recipientsRaw, ok := viola["recipients"]
    if !ok {
        return nil, fmt.Errorf("no recipients in _viola section")
    }

    // Handle []any from TOML parser
    recipientsSlice, ok := recipientsRaw.([]any)
    if !ok {
        return nil, fmt.Errorf("recipients not an array")
    }

    recipients := make([]string, len(recipientsSlice))
    for i, r := range recipientsSlice {
        recipients[i], ok = r.(string)
        if !ok {
            return nil, fmt.Errorf("recipient %d not a string", i)
        }
    }

    return recipients, nil
}

// setNestedField sets a value at a dot-notation path
func setNestedField(tree any, path string, value any) error {
    parts := strings.Split(path, ".")
    treeMap, ok := tree.(map[string]any)
    if !ok {
        return fmt.Errorf("tree is not a map")
    }

    // Navigate to parent
    current := treeMap
    for i := 0; i < len(parts)-1; i++ {
        key := parts[i]
        next, exists := current[key]
        if !exists {
            // Create intermediate map
            current[key] = make(map[string]any)
            current = current[key].(map[string]any)
        } else {
            nextMap, ok := next.(map[string]any)
            if !ok {
                return fmt.Errorf("path %s is not a map", strings.Join(parts[:i+1], "."))
            }
            current = nextMap
        }
    }

    // Set final value
    finalKey := parts[len(parts)-1]
    current[finalKey] = value

    return nil
}

// parseValue converts string value to appropriate type
func parseValue(value string, valueType string) (any, error) {
    switch valueType {
    case "string":
        return value, nil
    case "int":
        return strconv.Atoi(value)
    case "bool":
        return strconv.ParseBool(value)
    case "json":
        var result any
        if err := json.Unmarshal([]byte(value), &result); err != nil {
            return nil, err
        }
        return result, nil
    default:
        return nil, fmt.Errorf("unknown type: %s", valueType)
    }
}
```

#### Change 3: Update `inspect` Command

Add ability to show embedded recipients:

```bash
viola inspect config.toml --recipients

# Output:
File: config.toml
Encrypted fields: 3

Embedded Recipients (from [_viola] section):
  - age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
  - age1another...

Field: private_password
  Recipients: (uses embedded recipients above)
```

### Architecture Impact

**Benefits**:
- Self-contained files (no external dependencies)
- Faster field updates (no decrypt-edit-encrypt)
- Better UX (fewer commands, less error-prone)
- Discoverable (recipients visible in file)

**No Breaking Changes**:
- Files without `[_viola]` work as before
- CLI flags still work (and take precedence)
- Library API remains compatible
- All existing commands unchanged

**New Capabilities**:
- Direct field updates via `viola set`
- Self-contained file operation
- Reduced dependency on external files
- Better portability

---

## Implementation Milestones

### Milestone 1: Embedded Recipients Support
**Goal**: Files can store and use embedded recipients for self-contained operation

**Success Criteria**:
- [ ] `Save()` embeds recipients in `[_viola]` section
- [ ] `Load()` reads and uses embedded recipients
- [ ] `encrypt` command auto-embeds recipients by default
- [ ] `read` command uses embedded recipients if no CLI recipients
- [ ] CLI recipients take precedence over embedded
- [ ] Backward compatibility maintained (files without `[_viola]` work)

**Validation**:
```bash
viola encrypt test.toml -r recipients.txt -o out.toml
grep -A2 "\[_viola\]" out.toml  # Should show recipients
viola read out.toml -i identity.key  # Should work without -r flag
```

---

### Milestone 2: Basic `viola set` Command
**Goal**: Users can add/update encrypted fields with a single command

**Success Criteria**:
- [ ] `viola set` command exists and is wired up
- [ ] Can set top-level fields (e.g., `private_api_key`)
- [ ] Fields with `private_` prefix are automatically encrypted
- [ ] String values work correctly
- [ ] File is updated in place
- [ ] Round-trip encryption/decryption verified

**Validation**:
```bash
viola set config.toml private_test "secret"
viola read config.toml -i identity.key | grep "private_test.*secret"
```

---

### Milestone 3: Nested Path Support
**Goal**: Support dot notation for nested fields

**Success Criteria**:
- [ ] Dot notation parsing works (e.g., `db.auth.private_key`)
- [ ] Creates intermediate maps if they don't exist
- [ ] Updates existing nested fields
- [ ] Error handling for invalid paths
- [ ] Works with multiple nesting levels

**Validation**:
```bash
viola set config.toml database.private_password "newpass"
viola set config.toml services.auth.private_token "bearer-xyz"
viola read config.toml -i identity.key  # Both fields present
```

---

### Milestone 4: Self-Contained Workflow Integration
**Goal**: `viola set` uses embedded recipients automatically

**Success Criteria**:
- [ ] `viola set` reads embedded recipients from file
- [ ] Works without specifying CLI recipients
- [ ] Falls back to CLI recipients if no embedded ones
- [ ] Error message if no recipients available
- [ ] Verbose mode shows source of recipients

**Validation**:
```bash
viola encrypt config.toml -r recipients.txt -o config.toml
viola set config.toml private_new "value"  # No -r flag needed!
```

---

### Milestone 5: Enhanced Features & UX
**Goal**: Support multiple value types and improve user experience

**Success Criteria**:
- [ ] `--type` flag supports: string, int, bool, json
- [ ] `--dry-run` flag shows changes without saving
- [ ] `--output` flag allows saving to different file
- [ ] Helpful error messages for common mistakes
- [ ] `inspect` command shows embedded recipients

**Validation**: Test all value types and flags

---

### Milestone 6: Documentation Complete
**Goal**: Users can learn and use the feature effectively

**Success Criteria**:
- [ ] README updated with `viola set` examples
- [ ] Self-contained workflow documented
- [ ] `[_viola]` metadata section explained
- [ ] Migration guide for existing users
- [ ] Command reference updated

**Validation**: Documentation reviewed and examples tested

---

## Testing Strategy

### Unit Tests

**New Tests in `pkg/viola/viola_test.go`**:
- Embedding recipients in `[_viola]` section during `Save()`
- Reading embedded recipients during `Load()`
- Merging CLI and embedded recipients (CLI precedence)
- Removing `[_viola]` from returned tree
- Backward compatibility with files without `[_viola]`

**New Test File `cmd/viola/set_test.go`**:
- Dot notation path parsing
- Setting top-level fields
- Setting nested fields
- Creating intermediate maps
- Value type conversion (string, int, bool, json)
- Error handling for invalid paths

### Integration Tests

**End-to-End Workflows**:
```bash
# Test 1: Embedded recipients workflow
viola encrypt test.toml -r recipients.txt -o out.toml
viola read out.toml -i identity.key  # No -r needed
viola inspect out.toml --recipients  # Shows embedded

# Test 2: viola set basic
viola set config.toml private_key "secret"
viola read config.toml -i identity.key | grep "private_key.*secret"

# Test 3: viola set with embedded recipients
viola encrypt config.toml -r recipients.txt -o config.toml
viola set config.toml private_new "value"  # Uses embedded recipients
viola read config.toml -i identity.key | grep "private_new.*value"

# Test 4: Nested paths
viola set config.toml db.auth.private_pass "pass"
viola read config.toml -i identity.key | grep -A2 "\[db.auth\]"

# Test 5: Value types
viola set config.toml private_port 5432 --type int
viola set config.toml private_enabled true --type bool
viola set config.toml private_json '{"key":"val"}' --type json
```

### Manual Testing

- Test on real configuration files
- Verify file portability (copy to new location)
- Test with multiple recipients
- Test error scenarios (no recipients, invalid paths, etc.)
- Verify `[_viola]` section placement in output

---

## Documentation Updates

### README.md Updates

#### Add Section: Quick Field Updates

```markdown
### Quick Field Updates with `viola set`

Add or update encrypted fields without the decrypt-edit-encrypt dance:

```bash
# Add a new encrypted field
viola set config.toml private_api_key "sk-abc123"

# Update an existing field
viola set config.toml database.private_password "new-password"

# Set nested fields with dot notation
viola set config.toml services.auth.private_token "bearer-xyz"

# Different value types
viola set config.toml private_port 5432 --type int
viola set config.toml private_enabled true --type bool
viola set config.toml private_config '{"timeout": 30}' --type json

# Preview changes without saving
viola set config.toml private_key "new" --dry-run
```

The `set` command:
- Works directly on encrypted files (no decryption step needed)
- Automatically encrypts fields with `private_` prefix
- Uses embedded recipients from the file (no `-r` flag needed)
- Creates intermediate maps for nested paths
- Safer than decrypt-edit-encrypt (no plaintext files)
```

#### Add Section: Self-Contained Configuration Files

```markdown
### Self-Contained Configuration Files

Viola can embed recipients directly in your TOML files, making them self-contained:

```bash
# Encrypt a file (recipients are auto-embedded)
viola encrypt config.toml -r recipients.txt -o encrypted.toml

# The file now contains a [_viola] metadata section:
# [_viola]
# recipients = ["age1ql3z...", "age1another..."]

# Decrypt without specifying recipients (they're in the file!)
viola read encrypted.toml -i ~/.age/keys.txt

# Add fields without specifying recipients
viola set encrypted.toml private_new_key "secret"

# Copy file anywhere and it just works
cp encrypted.toml ~/other-project/
cd ~/other-project
viola read encrypted.toml -i ~/.age/keys.txt  # No recipients needed!
```

**Benefits**:
- **Portable**: Share just the TOML file, no recipient files needed
- **Self-documenting**: See who can decrypt by looking at `[_viola]` section
- **Bootstrap-friendly**: Easy to use in new projects
- **No CLI flags**: Decryption and field updates work without `-r`

**The `[_viola]` Metadata Section**:

Viola stores metadata in a special `[_viola]` section at the top of the file:

```toml
[_viola]
recipients = [
    "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
    "age1another..."
]

# Your configuration below
[database]
host = "localhost"
private_password = "-----BEGIN AGE ENCRYPTED FILE-----..."
```

This section is:
- Automatically created by `viola encrypt`
- Automatically used by `viola read` and `viola set`
- Removed from the output when you decrypt
- Optional (files without it still work with CLI recipients)

**CLI Precedence**:

If you specify recipients via CLI flags, they take precedence over embedded recipients:

```bash
# Use embedded recipients
viola read config.toml -i identity.key

# Override with CLI recipients
viola read config.toml -i identity.key -r other-recipients.txt
```
```

#### Add Section: Common Workflows

```markdown
### Common Workflows

#### Bootstrap a New Project

```bash
# Create initial encrypted config
cat > config.toml <<EOF
database = "localhost"
private_password = "initial-pass"
EOF

viola encrypt config.toml --recipients-inline "age1ql3z..." -o config.toml

# Later: Add more secrets as needed
viola set config.toml private_api_key "sk-abc123"
viola set config.toml services.redis.private_password "redis-pass"

# Share with team (just the TOML file, self-contained!)
git add config.toml
git commit -m "Add encrypted config"
```

#### Migrate Existing Configs to Self-Contained

```bash
# If you have an old encrypted config without embedded recipients
viola read old-config.toml -i identity.key -r recipients.txt > temp.toml
viola encrypt temp.toml -r recipients.txt -o new-config.toml
rm temp.toml

# Now new-config.toml is self-contained!
viola read new-config.toml -i identity.key  # No -r needed
```

#### Quick Secret Rotation

```bash
# Rotate a single secret
viola set config.toml database.private_password "new-password"

# That's it! No decrypt-edit-encrypt needed.
```

#### Add Field in CI/CD

```bash
# In your CI pipeline, add deployment-specific secrets
viola set config.toml deployment.private_token "$DEPLOY_TOKEN"
viola set config.toml deployment.private_region "us-west-2"
```
```

#### Update Command Reference Table

Add to the commands table:

| Command | Description |
|---------|-------------|
| `set` | Add or update an encrypted field directly |

Detailed `set` command reference:

```markdown
#### `viola set`

Add or update an encrypted field in a TOML file.

**Usage**: `viola set <file> <path> <value> [flags]`

**Flags**:
| Flag | Alias | Type | Description |
|------|-------|------|-------------|
| `--identity` | `-i` | string | Age identity file for decryption |
| `--type` | | string | Value type: string, int, bool, json (default: string) |
| `--output` | `-o` | string | Output file (default: overwrite input) |
| `--dry-run` | | bool | Show what would change without saving |
| `--verbose` | `-v` | bool | Show detailed operation info |

**Examples**:
```bash
# Set a string field
viola set config.toml private_api_key "sk-abc123"

# Set nested field
viola set config.toml database.auth.private_password "secret"

# Set with type
viola set config.toml private_port 5432 --type int
viola set config.toml private_enabled true --type bool
viola set config.toml private_settings '{"timeout": 30}' --type json

# Preview changes
viola set config.toml private_key "new" --dry-run

# Save to different file
viola set config.toml private_key "new" -o updated.toml
```

**Notes**:
- Fields starting with `private_` are automatically encrypted
- Uses embedded recipients from `[_viola]` section if present
- Creates intermediate maps for nested paths (e.g., `a.b.c`)
- Falls back to CLI identity if no embedded recipients
```
```

---

## Risks and Mitigations

### Risk 1: Breaking Existing Files
**Risk**: Changes to `Load()`/`Save()` could break existing encrypted files
**Impact**: High - users can't decrypt their configs
**Mitigation**:
- Maintain full backward compatibility
- Files without `[_viola]` work exactly as before
- Extensive testing with existing test files
- Gradual rollout with beta testing

**Status**: Low risk - design maintains backward compatibility

### Risk 2: `[_viola]` Conflicts with User Config
**Risk**: User might already have a `[_viola]` section in their config
**Impact**: Medium - naming collision
**Mitigation**:
- Use underscore prefix (uncommon in TOML configs)
- Document the reserved section name clearly
- Add validation to warn if user tries to create `[_viola]`
- Allow disabling with `--no-embed-recipients` flag

**Status**: Low risk - underscore prefix is intentionally reserved

### Risk 3: Path Parsing Ambiguity
**Risk**: Dot notation might conflict with keys that contain dots
**Impact**: Low - edge case
**Mitigation**:
- Document that dots are path separators
- Recommend avoiding dots in key names
- Future: Support bracket notation (`[key.with.dot]`) if needed

**Status**: Very low risk - dots in keys are rare

### Risk 4: File Corruption During `set`
**Risk**: Error during save could corrupt the file
**Impact**: High - data loss
**Mitigation**:
- Write to temp file first, then rename (atomic operation)
- Keep backup on error
- Add `--dry-run` for preview
- Extensive error handling and testing

**Status**: Medium risk - mitigated by atomic writes

### Risk 5: Embedded Recipients Leakage
**Risk**: Users might misunderstand that recipients are public info
**Impact**: Low - recipients are public keys (not secrets)
**Mitigation**:
- Document that recipients are safe to share
- Explain age encryption model in docs
- No actual security risk (recipients are meant to be public)

**Status**: No security risk - educational issue only

---

## Dependencies

**Existing Dependencies** (no new ones needed):
- `github.com/BurntSushi/toml` - TOML parsing (already used)
- `filippo.io/age` - Age encryption (already used)
- `github.com/urfave/cli/v2` - CLI framework (already used)

**Internal Dependencies**:
- `pkg/viola.Transform()` - Existing function for decrypt-modify-encrypt
- `pkg/enc` - Existing encryption helpers
- `cmd/viola` helpers - Existing CLI utilities

---

## Timeline Estimate

| Milestone | Estimated Effort | Dependencies |
|-----------|-----------------|--------------|
| Embedded Recipients Support | 4-6 hours | None |
| Basic `viola set` Command | 3-4 hours | None (can develop in parallel) |
| Nested Path Support | 2-3 hours | Milestone 2 |
| Self-Contained Integration | 1-2 hours | Milestones 1 & 2 |
| Enhanced Features & UX | 2-3 hours | Milestone 2 |
| Documentation | 2-3 hours | All milestones |
| **Total** | **14-21 hours** | |

**Development Order**:
1. Milestones 1 & 2 in parallel (independent)
2. Milestone 3 (builds on 2)
3. Milestone 4 (integrates 1 & 2)
4. Milestone 5 (polish)
5. Milestone 6 (document)

---

## Open Questions

1. **Embedded Recipients by Default?**
   - **Question**: Should `encrypt` always embed recipients, or require opt-in flag?
   - **Recommendation**: Always embed by default (add `--no-embed-recipients` to disable)
   - **Rationale**: Better UX, self-contained is the desired behavior

2. **Remove `[_viola]` from Decrypted Output?**
   - **Question**: Should `viola read` show or hide the `[_viola]` section?
   - **Recommendation**: Hide it (filter out before returning tree)
   - **Rationale**: It's metadata, not user config

3. **Value Type Auto-Detection?**
   - **Question**: Should `viola set` try to auto-detect value types?
   - **Recommendation**: No, require `--type` flag for non-strings
   - **Rationale**: Explicit is better than implicit, avoids ambiguity

4. **Array/Object Support in `set`?**
   - **Question**: How to set array elements or complex objects?
   - **Recommendation**: Use `--type json` for now, dedicated array syntax later
   - **Rationale**: JSON covers 80% of use cases, array syntax is complex

5. **Multiple Sets in One Command?**
   - **Question**: Support `viola set config.toml --set key1=val1 --set key2=val2`?
   - **Recommendation**: Not in initial release (add if users request)
   - **Rationale**: Keep scope manageable, single field is MVP

---

## Future Enhancements

Beyond this PRD scope, worth considering:

1. **`viola unset` Command**: Remove fields
   ```bash
   viola unset config.toml private_old_field
   ```

2. **`viola get` Command**: Read specific field value
   ```bash
   viola get config.toml database.private_password
   # Output: secret123
   ```

3. **Batch Operations**: Multiple sets in one command
   ```bash
   viola set config.toml --set key1="val1" --set key2="val2"
   ```

4. **Interactive Mode**: Prompt for values
   ```bash
   viola set config.toml private_password --interactive
   # Prompts: Enter value for private_password: ********
   ```

5. **Array/Object Syntax**: Dedicated syntax for complex types
   ```bash
   viola set config.toml servers[0].private_key "key1"
   viola set config.toml "config.nested[key]" "value"
   ```

6. **Metadata Extensions**: Add more to `[_viola]` section
   ```toml
   [_viola]
   recipients = [...]
   version = "1.0"
   cipher = "age"
   created = "2025-10-04"
   ```

7. **Recipient Management Commands**:
   ```bash
   viola recipients add config.toml age1newkey...
   viola recipients list config.toml
   viola recipients remove config.toml age1oldkey...
   ```

---

## Appendix: Examples

### Example 1: Bootstrap New Project

```bash
# Step 1: Create initial config
cat > config.toml <<EOF
app_name = "myapp"
database = "postgres://localhost/mydb"
EOF

# Step 2: Encrypt with your age key
viola encrypt config.toml --recipients-inline "age1ql3z..." -o config.toml

# Step 3: Add secrets as needed
viola set config.toml private_db_password "secret123"
viola set config.toml private_api_key "sk-abc123"
viola set config.toml services.redis.private_password "redis-pass"

# Step 4: Commit (it's encrypted and self-contained!)
git add config.toml
git commit -m "Add encrypted config"
```

### Example 2: Share Self-Contained Config

```bash
# On your machine: Create and share
viola encrypt config.toml -r recipients.txt -o config.toml
# Recipients are auto-embedded in [_viola] section
scp config.toml teammate@server:/app/

# On teammate's machine: Just works!
cd /app
viola read config.toml -i ~/.age/keys.txt
# No need to transfer recipients.txt!
```

### Example 3: Quick Secret Rotation

```bash
# Old way (multiple steps, error-prone):
viola read config.toml -i identity.key > temp.toml
# Edit temp.toml in text editor
viola encrypt temp.toml -r recipients.txt -o config.toml
rm temp.toml  # Don't forget!

# New way (one command):
viola set config.toml database.private_password "new-password"
# Done!
```

### Example 4: CI/CD Integration

```bash
#!/bin/bash
# deploy.sh

# Clone repo (includes encrypted config.toml with embedded recipients)
git clone https://github.com/myorg/myapp.git
cd myapp

# Add deployment-specific secrets
viola set config.toml deployment.private_api_token "$API_TOKEN" -i /secrets/deploy.key
viola set config.toml deployment.private_region "$AWS_REGION" -i /secrets/deploy.key

# Deploy with decrypted config
viola read config.toml -i /secrets/deploy.key -o json > /app/config.json
./deploy.sh
```

### Example 5: Inspect Self-Contained File

```bash
$ viola inspect config.toml --recipients

File: config.toml
Encrypted fields: 5

Embedded Recipients (from [_viola] section):
  - age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p (Alice)
  - age1another... (Bob)

Encrypted Fields:
  - private_api_key
  - database.private_password
  - services.redis.private_password
  - services.auth.private_token
  - deployment.private_config
```

### Example 6: File Structure Comparison

**Before (current - not self-contained)**:
```bash
# Files needed:
config.toml           # Encrypted config
recipients.txt        # List of recipients
identity.key          # Your private key

# Usage:
viola read config.toml -i identity.key -r recipients.txt
viola encrypt config.toml -r recipients.txt -o config.toml
```

**After (self-contained)**:
```bash
# Files needed:
config.toml           # Encrypted config with embedded recipients
identity.key          # Your private key

# Usage:
viola read config.toml -i identity.key
viola set config.toml private_key "value"
```

The `config.toml` now looks like:
```toml
[_viola]
recipients = [
    "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p",
    "age1another..."
]

app_name = "myapp"
database = "postgres://localhost/mydb"
private_db_password = """-----BEGIN AGE ENCRYPTED FILE-----
YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBYWlhYWFhYWFhYWFhY...
-----END AGE ENCRYPTED FILE-----"""

[services.redis]
private_password = """-----BEGIN AGE ENCRYPTED FILE-----
YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSBZWVlZWVlZWVlZWVlZ...
-----END AGE ENCRYPTED FILE-----"""
```

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-10-04 | Use `[_viola]` section for metadata | Extensible, won't conflict with user config (underscore prefix) |
| 2025-10-04 | Embed recipients by default | Better UX, self-contained is the goal |
| 2025-10-04 | Remove `[_viola]` from decrypted output | It's metadata, not user config |
| 2025-10-04 | CLI recipients take precedence | Allows overrides, matches existing CLI patterns |
| 2025-10-04 | Use dot notation for nested paths | Common convention, familiar to users |
| 2025-10-04 | Require `--type` flag for non-strings | Explicit > implicit, avoids ambiguity |
| 2025-10-04 | Single `set` command (not add/update/remove) | Simpler UX, upsert pattern is common |

---

## Progress Log

| Date | Update |
|------|--------|
| 2025-10-04 | PRD created based on user requirements for self-contained configs and field setting |
