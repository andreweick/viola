# PRD: Symmetric/Hybrid Encryption Support

**Status**: Draft
**Created**: 2025-10-04
**GitHub Issue**: [#1](https://github.com/andreweick/viola/issues/1)
**Owner**: TBD
**Priority**: High

---

## Problem Statement

Currently, viola's CLI `encrypt` command only supports asymmetric encryption using age recipients (public keys), despite the underlying library (`pkg/enc`) fully supporting password-based (scrypt) encryption. This creates several limitations:

1. **No Password Encryption**: Users cannot encrypt files using passwords, only age keys
2. **No Hybrid Encryption**: Cannot combine both age keys AND passwords as alternative decryption methods
3. **Limited Workflow Flexibility**: Development workflows that benefit from multiple decryption methods are not supported
4. **Incomplete CLI**: The `read` command supports password decryption, but `encrypt` does not support password encryption

### User Impact

**Development Workflow Pain Point**: Developers want to:
- Use their age identity key on their laptop for convenient, password-free decryption
- Have a password as a backup access method when working on different machines
- Support team collaboration where some members use keys, others use shared passwords
- Maintain recovery options if passwords are lost (decrypt with key, re-encrypt with new password)

Currently, this workflow is impossible because files can only be encrypted to age recipients.

---

## Solution Overview

Add password/passphrase encryption support to the `encrypt` command, mirroring the options already available in the `read` command. This enables:

1. **Password-Only Encryption**: Encrypt files using only a password (no recipients)
2. **Hybrid Encryption**: Encrypt files so they can be decrypted with EITHER age keys OR passwords
3. **Flexible Development Workflows**: Use keys for convenience, passwords for portability/sharing
4. **Recovery Capabilities**: If password is lost, decrypt with age key and re-encrypt with new password

### Key Principle

When a file is encrypted to multiple recipients (including scrypt/password recipients), it can be decrypted by **ANY** recipient, not **ALL** recipients. This is native age behavior.

---

## Success Criteria

### Must Have
- [ ] `encrypt` command accepts passphrase options (--passphrase, --passphrase-file, --passphrase-env)
- [ ] Can encrypt to password-only (no recipients)
- [ ] Can encrypt to hybrid mode (recipients + password)
- [ ] Files encrypted with password can be decrypted with existing `read` command
- [ ] Clear CLI error messages when no encryption method provided
- [ ] Documentation updated with examples

### Should Have
- [ ] Password strength validation/warnings
- [ ] Clear indication in `inspect` command when password is required
- [ ] Examples showing common workflows

### Could Have
- [ ] Password generation utility
- [ ] Interactive prompts when encryption method missing

---

## User Stories

### Story 1: Developer with Laptop Key + Backup Password
**As a** developer
**I want** to encrypt my config files to both my age key AND a password
**So that** I can decrypt quickly with my key on my laptop, but still access configs on other machines using the password

**Acceptance Criteria**:
- Encrypt once to both recipients and password
- Decrypt with either method independently
- Password not required when using age key

### Story 2: Team Sharing with Mixed Access Methods
**As a** team lead
**I want** to encrypt configs to multiple team members' age keys AND a shared password
**So that** each person can use their preferred decryption method

**Acceptance Criteria**:
- Multiple recipients + password in single file
- Each recipient can decrypt independently
- Password users don't need age keys

### Story 3: Password-Only Deployment
**As a** ops engineer
**I want** to encrypt configs using only a password
**So that** I can deploy to environments without managing age keys

**Acceptance Criteria**:
- No recipients required
- Simple password-based workflow
- Works with existing deployment tools

### Story 4: Key Rotation After Password Loss
**As a** security admin
**I want** to decrypt files with my age key when the password is lost
**So that** I can rotate to a new password without losing access

**Acceptance Criteria**:
- Decrypt with age key (ignoring password)
- Re-encrypt with new password
- Maintain backward compatibility

---

## Technical Design

### Current State

**Library Support** (`pkg/enc`):
- ✅ Full scrypt passphrase encryption/decryption support
- ✅ `KeySources.PassphraseProvider` function for password input
- ✅ Hybrid encryption (multiple recipients including scrypt)
- ✅ Test coverage for passphrase functionality

**CLI Support**:
- ✅ `read` command: Full passphrase support (--passphrase, --passphrase-file, --passphrase-env)
- ❌ `encrypt` command: NO passphrase support (recipients only)

### Proposed Changes

#### 1. Add Passphrase Flags to `encrypt` Command

```go
// Add to encrypt command flags (cmd/viola/main.go)
&cli.BoolFlag{
    Name:  "passphrase",
    Usage: "Prompt for passphrase interactively",
},
&cli.StringFlag{
    Name:  "passphrase-file",
    Usage: "Read passphrase from file (first line)",
},
&cli.StringFlag{
    Name:  "passphrase-env",
    Usage: "Read passphrase from environment variable",
},
```

#### 2. Wire Up Passphrase Provider in `encryptAction`

Mirror the pattern from `readAction` (lines 728-757 in main.go):
- Check which passphrase flag is set
- Configure `KeySources.PassphraseProvider` accordingly
- Use terminal password input for interactive mode

#### 3. Validation Logic

```go
// Ensure at least one encryption method is provided
hasRecipients := len(ks.Recipients) > 0 || ks.RecipientsFile != ""
hasPassphrase := ks.PassphraseProvider != nil

if !hasRecipients && !hasPassphrase {
    return cli.NewExitError("Must provide at least one of: recipients (-r) or passphrase", 1)
}
```

#### 4. Update `inspect` Command

Enhance recipient detection to clearly show when scrypt/password is used:
- Parse armor header to detect scrypt recipients
- Display "passphrase" or "password-protected" in recipient list
- Update line 1071-1073 with proper armor parsing

### Architecture Impact

**No breaking changes**:
- Existing files remain compatible
- Library API unchanged
- Only CLI flags added

**New capabilities**:
- Password-only encryption
- Hybrid encryption
- Workflow flexibility

---

## Implementation Milestones

### Milestone 1: Core Password Encryption Working
**Goal**: Users can encrypt and decrypt files using passwords

**Success Criteria**:
- [ ] `encrypt` command accepts --passphrase flag
- [ ] Interactive password prompt works (with terminal input masking)
- [ ] Can encrypt a file with password-only (no recipients)
- [ ] Can decrypt that file using `read --passphrase`
- [ ] Round-trip encryption/decryption verified

**Validation**: Manual test of password-only encryption workflow

---

### Milestone 2: Hybrid Encryption Working
**Goal**: Users can encrypt to both recipients AND passwords

**Success Criteria**:
- [ ] Can encrypt to recipients + password in same file
- [ ] Can decrypt with age key (ignoring password)
- [ ] Can decrypt with password (ignoring age key)
- [ ] Both methods work independently on same file

**Validation**: Manual test of hybrid encryption workflow with both decryption methods

---

### Milestone 3: Multiple Password Input Methods
**Goal**: Support all password input methods from `read` command

**Success Criteria**:
- [ ] --passphrase-file flag works
- [ ] --passphrase-env flag works
- [ ] All three methods (interactive, file, env) tested
- [ ] Error handling for missing/empty passwords

**Validation**: Test all three input methods with round-trip encryption/decryption

---

### Milestone 4: Enhanced UX and Validation
**Goal**: Provide clear feedback and prevent user errors

**Success Criteria**:
- [ ] Error message when no encryption method provided
- [ ] Warning for weak passwords (optional but recommended)
- [ ] `inspect` command shows "password-protected" in recipient list
- [ ] Help text updated with examples

**Validation**: Error scenarios tested, inspect command shows password status

---

### Milestone 5: Documentation Complete
**Goal**: Users can learn and use the feature effectively

**Success Criteria**:
- [ ] README.md updated with password encryption examples
- [ ] Hybrid encryption workflow documented
- [ ] Command reference updated with new flags
- [ ] Development workflow example added

**Validation**: Documentation reviewed and examples tested

---

## Testing Strategy

### Unit Tests
- Add tests for passphrase provider configuration in `encryptAction`
- Test validation logic (missing encryption methods)
- Test all three passphrase input methods

### Integration Tests
- Password-only encryption → decryption
- Hybrid encryption → decryption with key
- Hybrid encryption → decryption with password
- Error handling for invalid inputs

### Manual Testing
- Full development workflow (the user's use case)
- Cross-platform terminal password input
- File-based and env-based password input

---

## Documentation Updates

### README.md Updates

#### Add Password Encryption Section
```markdown
### Password Encryption

Encrypt files using passwords instead of or in addition to age keys:

```bash
# Password-only encryption
viola encrypt config.toml --passphrase -o encrypted.toml

# Hybrid: Both recipients AND password
viola encrypt config.toml -r recipients.txt --passphrase -o encrypted.toml

# Password from file
viola encrypt config.toml --passphrase-file pass.txt -o encrypted.toml

# Password from environment
export VIOLA_PASS="my-secret-password"
viola encrypt config.toml --passphrase-env VIOLA_PASS -o encrypted.toml
```

#### Add Development Workflow Example
```markdown
### Development Workflow: Key + Password

This workflow uses your age key for daily work and a password as backup:

```bash
# Encrypt to both your age key AND a password
viola encrypt config.toml \
  --recipients-inline "age1ql3z..." \
  --passphrase \
  -o encrypted-config.toml

# On your laptop: decrypt with age key (fast, no password)
viola read encrypted-config.toml -i ~/.age/keys.txt

# On other machines: decrypt with password
viola read encrypted-config.toml --passphrase

# If password lost: re-encrypt with new password
viola read encrypted-config.toml -i ~/.age/keys.txt -o toml > temp.toml
viola encrypt temp.toml --recipients-inline "age1ql3z..." --passphrase -o encrypted-config.toml
rm temp.toml
```

### Command Reference Updates

Update the `viola encrypt` section with new flags:

| Flag | Alias | Type | Description |
|------|-------|------|-------------|
| `--passphrase` | | bool | Prompt for passphrase interactively |
| `--passphrase-file` | | string | Read passphrase from file (first line) |
| `--passphrase-env` | | string | Read passphrase from environment variable |

---

## Risks and Mitigations

### Risk 1: Password Strength
**Risk**: Users choose weak passwords, compromising security
**Impact**: Medium - encrypted files could be brute-forced
**Mitigation**:
- Add password strength warnings (optional)
- Document password best practices
- Consider minimum length enforcement (e.g., 12 chars)

### Risk 2: Password Storage
**Risk**: Users might store passwords insecurely
**Impact**: Medium - defeats purpose of encryption
**Mitigation**:
- Document secure password management practices
- Recommend password managers
- Warn against committing password files to git

### Risk 3: Backward Compatibility
**Risk**: Changes break existing workflows
**Impact**: Low - only adding features, not changing existing behavior
**Mitigation**:
- No changes to existing flags or behavior
- Existing files remain compatible
- Comprehensive testing

### Risk 4: Cross-Platform Password Input
**Risk**: Terminal password input might not work on all platforms
**Impact**: Low - golang.org/x/term is well-tested
**Mitigation**:
- Use battle-tested terminal library
- Provide file/env alternatives
- Test on macOS, Linux, Windows

---

## Dependencies

- No new external dependencies (all libraries already present)
- Uses existing `filippo.io/age` scrypt support
- Uses existing `golang.org/x/term` for password input

---

## Timeline Estimate

| Milestone | Estimated Effort | Dependencies |
|-----------|-----------------|--------------|
| Core Password Encryption | 2-4 hours | None |
| Hybrid Encryption | 1-2 hours | Milestone 1 |
| Multiple Input Methods | 1-2 hours | Milestone 1 |
| Enhanced UX | 2-3 hours | Milestones 1-3 |
| Documentation | 1-2 hours | Milestones 1-4 |
| **Total** | **7-13 hours** | |

---

## Open Questions

1. **Password Strength Enforcement**: Should we enforce minimum password length/complexity?
   - Recommendation: Warn but don't block (let users decide)

2. **Password Confirmation**: Should interactive mode ask for password twice?
   - Recommendation: Yes, to prevent typos (common UX pattern)

3. **Default Behavior**: If no flags provided, what should happen?
   - Current: Error (no recipients)
   - After: Still error, but mention both options

4. **Inspect Enhancement**: How detailed should password detection be?
   - Minimum: Show "password-protected" in recipient list
   - Nice-to-have: Show scrypt work factor (from armor header)

---

## Future Enhancements

Beyond this PRD scope, but worth considering:

1. **Password Generation**: Built-in password generator utility
2. **Key Rotation Command**: Dedicated command for rotating keys/passwords
3. **Batch Re-encryption**: Re-encrypt multiple files at once
4. **Password Strength Meter**: Real-time feedback during password entry
5. **Hardware Security Module**: Support for HSM-stored keys

---

## Appendix: Examples

### Example 1: Password-Only Encryption
```bash
# Create a config file
cat > config.toml <<EOF
database = "localhost"
private_password = "secret123"
EOF

# Encrypt with password only
viola encrypt config.toml --passphrase -o encrypted.toml
# Prompts: Enter passphrase: ********
# Prompts: Confirm passphrase: ********

# Decrypt
viola read encrypted.toml --passphrase
# Prompts: Enter passphrase: ********
```

### Example 2: Hybrid Encryption
```bash
# Encrypt to both age key and password
viola encrypt config.toml \
  -r recipients.txt \
  --passphrase \
  -o encrypted.toml

# Decrypt with age key (no password needed)
viola read encrypted.toml -i identity.key

# OR decrypt with password (no age key needed)
viola read encrypted.toml --passphrase
```

### Example 3: Development Workflow
```bash
# Initial setup: encrypt to your key + password
viola encrypt dev-config.toml \
  --recipients-inline "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p" \
  --passphrase-env DEV_CONFIG_PASS \
  -o encrypted-dev-config.toml

# Daily work: use age key (stored in ~/.age/keys.txt)
viola read encrypted-dev-config.toml -i ~/.age/keys.txt

# On CI/CD server: use password from environment
export DEV_CONFIG_PASS="team-shared-password"
viola read encrypted-dev-config.toml --passphrase-env DEV_CONFIG_PASS

# Rotate password after team member leaves
viola read encrypted-dev-config.toml -i ~/.age/keys.txt -o toml > temp.toml
export NEW_PASS="new-team-password"
viola encrypt temp.toml \
  --recipients-inline "age1ql3z..." \
  --passphrase-env NEW_PASS \
  -o encrypted-dev-config.toml
rm temp.toml
```

### Example 4: Inspect Password-Protected File
```bash
# Check what encryption methods are used
viola inspect encrypted.toml --recipients

# Output:
# File: encrypted.toml
# Encrypted fields: 3
#
# Field: private_password
#   Recipients:
#     - age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
#     - password-protected (scrypt)
```

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2025-10-04 | Use same flag names as `read` command | Consistency across CLI commands |
| 2025-10-04 | Support hybrid encryption from start | Core use case requires it |
| 2025-10-04 | No minimum password enforcement | User freedom, warn instead |

---

## Progress Log

| Date | Update |
|------|--------|
| 2025-10-04 | PRD created |
