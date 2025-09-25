package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/charmbracelet/lipgloss"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/andreweick/viola/pkg/enc"
	"github.com/andreweick/viola/pkg/viola"
)

var (
	// Styles for beautiful CLI output
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			PaddingTop(1).
			PaddingBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5A9FD4"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)
)

func main() {
	app := &cli.App{
		Name:  "viola",
		Usage: "Encrypted TOML configuration file manager",
		Description: titleStyle.Render(`🎭 viola - Versatile Immutable Obscured Loader for Archives

Like Shakespeare's Viola, who conceals her identity to safely move between worlds,
this tool helps your configs take on a safe, portable form while keeping their
secrets hidden from prying eyes.`),
		Version: "0.1.0",
		Commands: []*cli.Command{
			readCommand(),
			encryptCommand(),
			inspectCommand(),
			verifyCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func readCommand() *cli.Command {
	return &cli.Command{
		Name:    "read",
		Aliases: []string{"decrypt", "show", "view"},
		Usage:   "Read and decrypt a TOML configuration file",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "identity",
				Aliases: []string{"i"},
				Usage:   "Path to age identity file",
				Value:   cli.NewStringSlice(),
			},
			&cli.StringFlag{
				Name:    "key",
				Aliases: []string{"k"},
				Usage:   "Inline age identity key (insecure, for testing)",
			},
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
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: toml, json, yaml, env, flat",
				Value:   "toml",
			},
			&cli.BoolFlag{
				Name:  "raw",
				Usage: "Show raw encrypted values without decrypting",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "Extract specific path (dot notation: server.private_key)",
			},
			&cli.BoolFlag{
				Name:  "private-only",
				Usage: "Show only encrypted fields",
			},
			&cli.BoolFlag{
				Name:  "public-only",
				Usage: "Show only non-encrypted fields",
			},
			&cli.BoolFlag{
				Name:  "show-qr",
				Usage: "Display QR codes alongside values",
			},
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable colored output",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Suppress non-essential output",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Show detailed decryption info",
			},
		},
		Action: readAction,
	}
}

func encryptCommand() *cli.Command {
	return &cli.Command{
		Name:    "encrypt",
		Aliases: []string{"enc", "generate"},
		Usage:   "Encrypt a TOML configuration file",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "recipients",
				Aliases: []string{"r"},
				Usage:   "Path to recipients file containing age public keys",
			},
			&cli.StringFlag{
				Name:  "recipients-inline",
				Usage: "Comma-separated age public keys for encryption",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file path (default: stdout)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite output file if it exists",
			},
			&cli.StringFlag{
				Name:  "private-prefix",
				Usage: "Prefix for fields to encrypt (default: 'private_')",
				Value: "private_",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "Show what would be encrypted without doing it",
			},
			&cli.BoolFlag{
				Name:  "stats",
				Usage: "Show encryption statistics",
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Suppress non-essential output",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Show detailed encryption info",
			},
		},
		Action: encryptAction,
	}
}

func inspectCommand() *cli.Command {
	return &cli.Command{
		Name:  "inspect",
		Usage: "Inspect encrypted file metadata without decrypting",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "fields",
				Usage: "List all encrypted field paths",
			},
			&cli.BoolFlag{
				Name:  "recipients",
				Usage: "Show recipients for each field",
			},
			&cli.BoolFlag{
				Name:  "stats",
				Usage: "Show encryption statistics",
			},
			&cli.StringFlag{
				Name:  "qr",
				Usage: "Display QR for specific encrypted field",
			},
			&cli.StringFlag{
				Name:  "check-recipient",
				Usage: "Check if recipient can decrypt",
			},
		},
		Action: inspectAction,
	}
}

func verifyCommand() *cli.Command {
	return &cli.Command{
		Name:  "verify",
		Usage: "Verify file integrity and decryptability",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "identity",
				Aliases: []string{"i"},
				Usage:   "Identity to verify against",
			},
			&cli.BoolFlag{
				Name:  "check-all",
				Usage: "Verify all encrypted fields are decryptable",
			},
			&cli.BoolFlag{
				Name:  "check-format",
				Usage: "Verify TOML format is valid",
			},
			&cli.BoolFlag{
				Name:  "check-armor",
				Usage: "Verify armor blocks are valid",
			},
		},
		Action: verifyAction,
	}
}

func readAction(c *cli.Context) error {
	filename := c.Args().First()
	if filename == "" {
		return cli.NewExitError(errorStyle.Render("Error: No file specified"), 1)
	}

	if !c.Bool("quiet") {
		fmt.Print(headerStyle.Render(" READ COMMAND "))
		fmt.Println()
		fmt.Println()
	}

	// Read the TOML file
	data, err := readFile(filename)
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error reading file: %v", err)), 1)
	}

	// Build key sources from CLI flags
	keySources, err := buildKeySources(c)
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error setting up keys: %v", err)), 1)
	}


	// Configure viola options
	opts := viola.Options{
		Keys: keySources,
	}

	// Load and decrypt the configuration
	result, err := viola.Load(data, opts)
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error loading configuration: %v", err)), 1)
	}

	// Handle raw output (show encrypted values without decrypting)
	if c.Bool("raw") {
		// Parse TOML without decryption - just read the raw file
		rawResult, err := viola.Load(data, viola.Options{}) // No keys
		if err != nil {
			return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error parsing file: %v", err)), 1)
		}
		rawData, err := formatOutput(rawResult.Tree, "toml", c.Bool("no-color"))
		if err != nil {
			return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error formatting output: %v", err)), 1)
		}
		fmt.Print(string(rawData))
		return nil
	}

	// Filter fields if requested
	tree := result.Tree
	if c.Bool("private-only") || c.Bool("public-only") {
		tree = filterFields(tree, result.Fields, c.Bool("private-only"))
	}

	// Extract specific path if requested
	if pathStr := c.String("path"); pathStr != "" {
		path := strings.Split(pathStr, ".")
		value, found := extractPath(tree, path)
		if !found {
			return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Path not found: %s", pathStr)), 1)
		}
		tree = map[string]any{pathStr: value}
	}

	// Format output
	outputFormat := c.String("output")
	output, err := formatOutput(tree, outputFormat, c.Bool("no-color"))
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error formatting output: %v", err)), 1)
	}

	fmt.Print(string(output))

	// Show verbose information if requested
	if c.Bool("verbose") && !c.Bool("quiet") {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, infoStyle.Render(fmt.Sprintf("✓ Processed %d encrypted fields", countEncryptedFields(result.Fields))))
		fmt.Fprintf(os.Stderr, "\n")
	}

	return nil
}

func encryptAction(c *cli.Context) error {
	filename := c.Args().First()
	if filename == "" {
		return cli.NewExitError(errorStyle.Render("Error: No file specified"), 1)
	}

	if !c.Bool("quiet") {
		fmt.Print(headerStyle.Render(" ENCRYPT COMMAND "))
		fmt.Println()
		fmt.Println()
	}

	// Read the plain TOML file
	data, err := readFile(filename)
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error reading file: %v", err)), 1)
	}

	// Build recipients from CLI flags
	recipients, err := buildRecipients(c)
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error setting up recipients: %v", err)), 1)
	}

	// Configure viola options
	opts := viola.Options{
		Keys: enc.KeySources{
			Recipients: recipients,
		},
		PrivatePrefix: c.String("private-prefix"),
	}

	// Load the plain configuration (no decryption needed)
	result, err := viola.Load(data, viola.Options{}) // No keys for loading
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error parsing TOML: %v", err)), 1)
	}

	if c.Bool("dry-run") {
		// Show what would be encrypted
		encryptedFields := findFieldsToEncrypt(result.Tree, []string{}, c.String("private-prefix"))

		if !c.Bool("quiet") {
			if len(encryptedFields) == 0 {
				fmt.Println(infoStyle.Render("No fields found with the specified prefix"))
			} else {
				fmt.Println(headerStyle.Render(fmt.Sprintf("Would encrypt %d fields:", len(encryptedFields))))
				for _, field := range encryptedFields {
					fmt.Printf("  - %s\n", strings.Join(field, "."))
				}
			}
		}
		return nil
	}

	// Encrypt the configuration
	encryptedTOML, fields, err := viola.Save(result.Tree, opts)
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error encrypting configuration: %v", err)), 1)
	}

	// Handle output
	outputFile := c.String("output")
	if outputFile != "" {
		// Check if file exists and force flag
		if _, err := os.Stat(outputFile); err == nil && !c.Bool("force") {
			return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Output file exists: %s (use --force to overwrite)", outputFile)), 1)
		}

		// Write to file
		err = os.WriteFile(outputFile, encryptedTOML, 0644)
		if err != nil {
			return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error writing output file: %v", err)), 1)
		}

		if !c.Bool("quiet") {
			fmt.Printf("✓ Encrypted configuration written to: %s\n", outputFile)
		}
	} else {
		// Write to stdout
		fmt.Print(string(encryptedTOML))
	}

	// Show statistics if requested
	if c.Bool("stats") && !c.Bool("quiet") {
		encryptedCount := 0
		for _, field := range fields {
			if field.WasEncrypted {
				encryptedCount++
			}
		}

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, successStyle.Render(fmt.Sprintf("✓ Encrypted %d fields", encryptedCount)))
		fmt.Fprintf(os.Stderr, "\n")

		if c.Bool("verbose") {
			fmt.Fprintf(os.Stderr, "Encrypted fields:\n")
			for _, field := range fields {
				if field.WasEncrypted {
					fmt.Fprintf(os.Stderr, "  - %s\n", strings.Join(field.Path, "."))
				}
			}
		}
	}

	// Show verbose information if requested
	if c.Bool("verbose") && !c.Bool("quiet") && !c.Bool("stats") {
		encryptedCount := countEncryptedFields(fields)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, successStyle.Render(fmt.Sprintf("✓ Encrypted %d fields", encryptedCount)))
		fmt.Fprintf(os.Stderr, "\n")
	}

	return nil
}

func inspectAction(c *cli.Context) error {
	filename := c.Args().First()
	if filename == "" {
		return cli.NewExitError(errorStyle.Render("Error: No file specified"), 1)
	}

	fmt.Print(headerStyle.Render(" INSPECT COMMAND "))
	fmt.Println()
	fmt.Println()

	// Read the TOML file
	data, err := readFile(filename)
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error reading file: %v", err)), 1)
	}

	// Parse TOML without decryption to find encrypted fields
	result, err := viola.Load(data, viola.Options{}) // No keys - just parse
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error parsing TOML: %v", err)), 1)
	}

	// Find all encrypted fields
	encryptedFields := findEncryptedFields(result.Tree, []string{})

	if c.Bool("stats") {
		fmt.Printf("File: %s\n", filename)
		fmt.Printf("Total fields: %d\n", countAllFields(result.Tree))
		fmt.Printf("Encrypted fields: %d\n", len(encryptedFields))
		fmt.Printf("File size: %d bytes\n", len(data))
		fmt.Println()
	}

	if c.Bool("fields") {
		if len(encryptedFields) == 0 {
			fmt.Println(infoStyle.Render("No encrypted fields found"))
		} else {
			fmt.Println(headerStyle.Render("Encrypted Fields:"))
			for _, field := range encryptedFields {
				fmt.Printf("  %s\n", strings.Join(field.Path, "."))
			}
		}
		fmt.Println()
	}

	if c.Bool("recipients") {
		if len(encryptedFields) == 0 {
			fmt.Println(infoStyle.Render("No encrypted fields found"))
		} else {
			fmt.Println(headerStyle.Render("Recipients per Field:"))
			for _, field := range encryptedFields {
				fmt.Printf("  %s:\n", strings.Join(field.Path, "."))
				recipients := extractRecipientsFromArmor(field.Armored)
				if len(recipients) > 0 {
					for _, recipient := range recipients {
						fmt.Printf("    - %s\n", recipient)
					}
				} else {
					fmt.Printf("    (could not extract recipients)\n")
				}
			}
		}
		fmt.Println()
	}

	if qrField := c.String("qr"); qrField != "" {
		path := strings.Split(qrField, ".")
		for _, field := range encryptedFields {
			if len(field.Path) == len(path) {
				match := true
				for i, part := range path {
					if field.Path[i] != part {
						match = false
						break
					}
				}
				if match {
					fmt.Printf(headerStyle.Render("QR Code for %s:"), qrField)
					fmt.Println()
					fmt.Println(infoStyle.Render("QR code generation not yet implemented"))
					fmt.Printf("Armored data (%d chars):\n%s\n", len(field.Armored), field.Armored)
					return nil
				}
			}
		}
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Encrypted field not found: %s", qrField)), 1)
	}

	// Default output if no specific flags
	if !c.Bool("stats") && !c.Bool("fields") && !c.Bool("recipients") && c.String("qr") == "" {
		fmt.Printf("File: %s\n", filename)
		fmt.Printf("Encrypted fields: %d\n", len(encryptedFields))
		if len(encryptedFields) > 0 {
			fmt.Println("\nEncrypted field paths:")
			for _, field := range encryptedFields {
				fmt.Printf("  - %s\n", strings.Join(field.Path, "."))
			}
		}
	}

	return nil
}

func verifyAction(c *cli.Context) error {
	filename := c.Args().First()
	if filename == "" {
		return cli.NewExitError(errorStyle.Render("Error: No file specified"), 1)
	}

	fmt.Print(headerStyle.Render(" VERIFY COMMAND "))
	fmt.Println()
	fmt.Println()

	// Read the TOML file
	data, err := readFile(filename)
	if err != nil {
		return cli.NewExitError(errorStyle.Render(fmt.Sprintf("Error reading file: %v", err)), 1)
	}

	var hasErrors bool
	results := []string{}

	// Check TOML format
	if c.Bool("check-format") || c.Bool("check-all") {
		_, err := viola.Load(data, viola.Options{})
		if err != nil {
			results = append(results, errorStyle.Render("✗ TOML format invalid: "+err.Error()))
			hasErrors = true
		} else {
			results = append(results, successStyle.Render("✓ TOML format valid"))
		}
	}

	// Check armor blocks
	if c.Bool("check-armor") || c.Bool("check-all") {
		result, err := viola.Load(data, viola.Options{})
		if err != nil {
			results = append(results, errorStyle.Render("✗ Could not parse file to check armor"))
			hasErrors = true
		} else {
			encryptedFields := findEncryptedFields(result.Tree, []string{})
			armorValid := true
			for _, field := range encryptedFields {
				if !isValidArmor(field.Armored) {
					results = append(results, errorStyle.Render(fmt.Sprintf("✗ Invalid armor block in field: %s", strings.Join(field.Path, "."))))
					armorValid = false
					hasErrors = true
				}
			}
			if armorValid {
				if len(encryptedFields) > 0 {
					results = append(results, successStyle.Render(fmt.Sprintf("✓ All %d armor blocks are valid", len(encryptedFields))))
				} else {
					results = append(results, infoStyle.Render("ℹ No armor blocks found to verify"))
				}
			}
		}
	}

	// Check decryptability
	if c.Bool("check-all") || len(c.StringSlice("identity")) > 0 {
		keySources, err := buildKeySources(c)
		if err != nil {
			results = append(results, errorStyle.Render("✗ Error setting up keys: "+err.Error()))
			hasErrors = true
		} else {
			opts := viola.Options{Keys: keySources}
			result, err := viola.Load(data, opts)
			if err != nil {
				results = append(results, errorStyle.Render("✗ Decryption failed: "+err.Error()))
				hasErrors = true
			} else {
				encryptedFields := result.Fields
				decryptableFields := 0
				undecryptableFields := 0

				for _, field := range encryptedFields {
					if field.WasEncrypted {
						// Check if field was successfully decrypted by seeing if it's still armored
						value, found := extractPath(result.Tree, field.Path)
						if found {
							if strVal, ok := value.(string); ok && strings.Contains(strVal, "AGE ENCRYPTED FILE") {
								undecryptableFields++
							} else {
								decryptableFields++
							}
						}
					}
				}

				if undecryptableFields > 0 {
					results = append(results, errorStyle.Render(fmt.Sprintf("✗ %d fields could not be decrypted", undecryptableFields)))
					hasErrors = true
				}
				if decryptableFields > 0 {
					results = append(results, successStyle.Render(fmt.Sprintf("✓ %d fields successfully decrypted", decryptableFields)))
				}
				if decryptableFields == 0 && undecryptableFields == 0 {
					results = append(results, infoStyle.Render("ℹ No encrypted fields found"))
				}
			}
		}
	}

	// Print results
	fmt.Printf("File: %s\n\n", filename)
	for _, result := range results {
		fmt.Println(result)
	}

	if hasErrors {
		return cli.NewExitError("", 1)
	}

	return nil
}

// Helper functions

// readFile reads a file and returns its contents
func readFile(filename string) ([]byte, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %w", filename, err)
	}

	return data, nil
}

// buildKeySources creates KeySources from CLI flags
func buildKeySources(c *cli.Context) (enc.KeySources, error) {
	ks := enc.KeySources{}

	// Add identity files
	identityFiles := c.StringSlice("identity")

	if len(identityFiles) > 0 {
		for _, file := range identityFiles {
			if _, err := os.Stat(file); err != nil {
				return ks, fmt.Errorf("identity file not accessible: %s", file)
			}
		}
		if len(identityFiles) == 1 {
			ks.IdentitiesFile = identityFiles[0]
		} else {
			// Multiple files - read them all into IdentitiesData
			for _, file := range identityFiles {
				data, err := os.ReadFile(file)
				if err != nil {
					return ks, fmt.Errorf("cannot read identity file %s: %w", file, err)
				}
				ks.IdentitiesData = append(ks.IdentitiesData, string(data))
			}
		}
	}

	// Add inline key
	key := c.String("key")
	if key != "" {
		ks.IdentitiesData = append(ks.IdentitiesData, key)
	}

	// Set up passphrase provider
	if c.Bool("passphrase") {
		ks.PassphraseProvider = func() (string, error) {
			fmt.Print("Enter passphrase: ")
			password, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			return string(password), err
		}
	} else if passphraseFile := c.String("passphrase-file"); passphraseFile != "" {
		ks.PassphraseProvider = func() (string, error) {
			data, err := os.ReadFile(passphraseFile)
			if err != nil {
				return "", err
			}
			// Use first line only
			lines := strings.Split(string(data), "\n")
			if len(lines) > 0 {
				return strings.TrimSpace(lines[0]), nil
			}
			return "", fmt.Errorf("empty passphrase file")
		}
	} else if passphraseEnv := c.String("passphrase-env"); passphraseEnv != "" {
		ks.PassphraseProvider = func() (string, error) {
			passphrase := os.Getenv(passphraseEnv)
			if passphrase == "" {
				return "", fmt.Errorf("passphrase environment variable %s is empty", passphraseEnv)
			}
			return passphrase, nil
		}
	}

	return ks, nil
}

// buildRecipients creates a list of recipients from CLI flags
func buildRecipients(c *cli.Context) ([]string, error) {
	var recipients []string

	// Add recipients from file
	recipientFiles := c.StringSlice("recipients")

	if len(recipientFiles) > 0 {
		for _, file := range recipientFiles {
			if _, err := os.Stat(file); err != nil {
				return nil, fmt.Errorf("recipients file not accessible: %s", file)
			}

			data, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("cannot read recipients file %s: %w", file, err)
			}

			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					recipients = append(recipients, line)
				}
			}
		}
	}

	// Add inline recipients
	inlineRecipients := c.String("recipients-inline")

	if inlineRecipients != "" {
		parts := strings.Split(inlineRecipients, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				recipients = append(recipients, part)
			}
		}
	}

	if len(recipients) == 0 {
		return nil, fmt.Errorf("no recipients specified (use --recipients or --recipients-inline)")
	}

	return recipients, nil
}

// formatOutput formats data according to the specified format
func formatOutput(data any, format string, noColor bool) ([]byte, error) {
	switch format {
	case "json":
		if noColor {
			return json.Marshal(data)
		}
		return json.MarshalIndent(data, "", "  ")

	case "yaml":
		return yaml.Marshal(data)

	case "env":
		return formatAsEnv(data, ""), nil

	case "flat":
		return formatAsFlat(data, ""), nil

	case "toml":
		fallthrough
	default:
		return formatAsTOML(data)
	}
}

// formatAsTOML formats data as TOML
func formatAsTOML(data any) ([]byte, error) {
	var buf strings.Builder
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

// formatAsEnv formats data as environment variables
func formatAsEnv(data any, prefix string) []byte {
	var result []string
	flattenForEnv(data, prefix, &result)
	return []byte(strings.Join(result, "\n"))
}

// flattenForEnv recursively flattens data for environment variable format
func flattenForEnv(data any, prefix string, result *[]string) {
	switch v := data.(type) {
	case map[string]any:
		for key, value := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "_" + key
			}
			flattenForEnv(value, newPrefix, result)
		}
	case []any:
		for i, value := range v {
			newPrefix := fmt.Sprintf("%s_%d", prefix, i)
			flattenForEnv(value, newPrefix, result)
		}
	default:
		*result = append(*result, fmt.Sprintf("%s=%v", strings.ToUpper(prefix), v))
	}
}

// formatAsFlat formats data as flat key=value pairs
func formatAsFlat(data any, prefix string) []byte {
	var result []string
	flattenForFlat(data, prefix, &result)
	return []byte(strings.Join(result, "\n"))
}

// flattenForFlat recursively flattens data for flat format
func flattenForFlat(data any, prefix string, result *[]string) {
	switch v := data.(type) {
	case map[string]any:
		for key, value := range v {
			newPrefix := key
			if prefix != "" {
				newPrefix = prefix + "." + key
			}
			flattenForFlat(value, newPrefix, result)
		}
	case []any:
		for i, value := range v {
			newPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			flattenForFlat(value, newPrefix, result)
		}
	default:
		*result = append(*result, fmt.Sprintf("%s=%v", prefix, v))
	}
}

// filterFields filters the tree to show only private or public fields
func filterFields(tree map[string]any, fields []viola.FieldMeta, privateOnly bool) map[string]any {
	if privateOnly {
		// Show only encrypted fields
		result := make(map[string]any)
		for _, field := range fields {
			if field.WasEncrypted && len(field.Path) > 0 {
				setNestedValue(result, field.Path, getNestedValue(tree, field.Path))
			}
		}
		return result
	} else {
		// Show only non-encrypted fields (publicOnly)
		result := make(map[string]any)
		encryptedPaths := make(map[string]bool)

		// Mark encrypted paths
		for _, field := range fields {
			if field.WasEncrypted {
				encryptedPaths[strings.Join(field.Path, ".")] = true
			}
		}

		// Copy non-encrypted fields
		copyNonEncrypted(tree, result, "", encryptedPaths)
		return result
	}
}

// copyNonEncrypted recursively copies non-encrypted fields
func copyNonEncrypted(src, dest map[string]any, prefix string, encryptedPaths map[string]bool) {
	for key, value := range src {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		if !encryptedPaths[path] {
			if subMap, ok := value.(map[string]any); ok {
				dest[key] = make(map[string]any)
				copyNonEncrypted(subMap, dest[key].(map[string]any), path, encryptedPaths)
			} else {
				dest[key] = value
			}
		}
	}
}

// extractPath extracts a value from a nested map using a path
func extractPath(tree map[string]any, path []string) (any, bool) {
	current := tree
	for i, key := range path {
		if i == len(path)-1 {
			value, exists := current[key]
			return value, exists
		}
		next, exists := current[key]
		if !exists {
			return nil, false
		}
		if nextMap, ok := next.(map[string]any); ok {
			current = nextMap
		} else {
			return nil, false
		}
	}
	return nil, false
}

// getNestedValue gets a value from nested path
func getNestedValue(data map[string]any, path []string) any {
	value, _ := extractPath(data, path)
	return value
}

// setNestedValue sets a value at a nested path
func setNestedValue(data map[string]any, path []string, value any) {
	current := data
	for i, key := range path {
		if i == len(path)-1 {
			current[key] = value
			return
		}
		if _, exists := current[key]; !exists {
			current[key] = make(map[string]any)
		}
		if nextMap, ok := current[key].(map[string]any); ok {
			current = nextMap
		} else {
			// Overwrite with map if not already a map
			current[key] = make(map[string]any)
			current = current[key].(map[string]any)
		}
	}
}

// countEncryptedFields counts how many fields were encrypted
func countEncryptedFields(fields []viola.FieldMeta) int {
	count := 0
	for _, field := range fields {
		if field.WasEncrypted {
			count++
		}
	}
	return count
}

// findEncryptedFields finds all encrypted fields in a tree
func findEncryptedFields(tree any, path []string) []struct {
	Path    []string
	Armored string
} {
	var fields []struct {
		Path    []string
		Armored string
	}

	switch v := tree.(type) {
	case map[string]any:
		for key, value := range v {
			newPath := append(path, key)
			if strValue, ok := value.(string); ok && isArmoredData(strValue) {
				fields = append(fields, struct {
					Path    []string
					Armored string
				}{
					Path:    newPath,
					Armored: strValue,
				})
			} else {
				fields = append(fields, findEncryptedFields(value, newPath)...)
			}
		}
	case []any:
		for i, value := range v {
			newPath := append(path, fmt.Sprintf("[%d]", i))
			fields = append(fields, findEncryptedFields(value, newPath)...)
		}
	}

	return fields
}

// isArmoredData checks if a string looks like ASCII-armored age data
func isArmoredData(s string) bool {
	return strings.Contains(s, "-----BEGIN AGE ENCRYPTED FILE-----") &&
		strings.Contains(s, "-----END AGE ENCRYPTED FILE-----")
}

// countAllFields counts all fields in a tree
func countAllFields(tree any) int {
	count := 0
	switch v := tree.(type) {
	case map[string]any:
		for _, value := range v {
			count++
			count += countAllFields(value)
		}
	case []any:
		for _, value := range v {
			count += countAllFields(value)
		}
	}
	return count
}

// extractRecipientsFromArmor extracts recipient info from armor block (simplified)
func extractRecipientsFromArmor(armored string) []string {
	// This is a simplified implementation
	// In a real implementation, you'd parse the armor header
	if strings.Contains(armored, "scrypt") {
		return []string{"passphrase"}
	}
	return []string{"X25519 recipient"}
}

// isValidArmor checks if an armor block has valid structure
func isValidArmor(armored string) bool {
	return strings.Contains(armored, "-----BEGIN AGE ENCRYPTED FILE-----") &&
		strings.Contains(armored, "-----END AGE ENCRYPTED FILE-----") &&
		strings.Index(armored, "-----BEGIN AGE ENCRYPTED FILE-----") <
		strings.Index(armored, "-----END AGE ENCRYPTED FILE-----")
}

// findFieldsToEncrypt finds all fields that would be encrypted based on prefix
func findFieldsToEncrypt(tree any, path []string, prefix string) [][]string {
	var fields [][]string

	switch v := tree.(type) {
	case map[string]any:
		for key, value := range v {
			newPath := append(path, key)
			if strings.HasPrefix(key, prefix) {
				// This field would be encrypted
				fields = append(fields, newPath)
			} else {
				// Recursively check nested structures
				fields = append(fields, findFieldsToEncrypt(value, newPath, prefix)...)
			}
		}
	case []any:
		for i, value := range v {
			newPath := append(path, fmt.Sprintf("[%d]", i))
			fields = append(fields, findFieldsToEncrypt(value, newPath, prefix)...)
		}
	}

	return fields
}
