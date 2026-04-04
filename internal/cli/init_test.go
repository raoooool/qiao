package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/raoooool/qiao/internal/core"
)

func newInitTestDeps(t *testing.T) (InitDependencies, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	return InitDependencies{
		Stdin:      strings.NewReader(""),
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		ConfigPath: path,
		ListProviders: func() []string {
			return []string{"claude", "codex", "tencent"}
		},
		ConfigFields: func(name string) []core.ConfigField {
			if name == "tencent" {
				return []core.ConfigField{
					{Key: "secret_id", Label: "Secret ID", Required: true, Secret: true},
					{Key: "secret_key", Label: "Secret Key", Required: true, Secret: true},
				}
			}
			return nil
		},
		ReadSecret: func() (string, error) {
			return "test-secret-value", nil
		},
	}, path
}

func defaultTestConfigDeps(t *testing.T) ConfigDependencies {
	t.Helper()
	return ConfigDependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		ConfigPath: filepath.Join(t.TempDir(), "config.yaml"),
	}
}

func defaultTestInitDeps(t *testing.T) InitDependencies {
	t.Helper()
	deps, _ := newInitTestDeps(t)
	return deps
}

func TestInitRequiresExplicitProviderSelection(t *testing.T) {
	deps, path := newInitTestDeps(t)
	deps.Stdin = strings.NewReader("\n2\n")

	cmd := newRootCommand(defaultTestTranslateDeps(), defaultTestConfigDeps(t), deps)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "default_provider: codex") {
		t.Fatalf("expected default_provider: codex in config, got %q", string(data))
	}

	stdout := deps.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "Configuration saved") {
		t.Fatalf("expected saved message, got %q", stdout)
	}
	if !strings.Contains(stdout, "Invalid choice, try again: ") {
		t.Fatalf("expected retry prompt for empty selection, got %q", stdout)
	}
}

func TestInitOutputDoesNotDescribeProviderAsDefault(t *testing.T) {
	deps, _ := newInitTestDeps(t)
	deps.Stdin = strings.NewReader("\n")

	cmd := newRootCommand(defaultTestTranslateDeps(), defaultTestConfigDeps(t), deps)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	stdout := deps.Stdout.(*bytes.Buffer).String()
	if strings.Contains(stdout, "default translation provider") {
		t.Fatalf("expected init prompt to avoid default wording, got %q", stdout)
	}
	if strings.Contains(stdout, "(default)") {
		t.Fatalf("expected init prompt to avoid default marker, got %q", stdout)
	}
	if strings.Contains(stdout, "Enter number (default") {
		t.Fatalf("expected init prompt to avoid default input hint, got %q", stdout)
	}
	if strings.Contains(stdout, "Enter number [") {
		t.Fatalf("expected init prompt to avoid suggested provider index, got %q", stdout)
	}
}

func TestInitSelectsTencentWithCredentials(t *testing.T) {
	deps, path := newInitTestDeps(t)
	deps.Stdin = strings.NewReader("3\n") // select tencent
	secretInputs := []string{"my-secret-id", "my-secret-key"}
	callIndex := 0
	deps.ReadSecret = func() (string, error) {
		val := secretInputs[callIndex]
		callIndex++
		return val, nil
	}

	cmd := newRootCommand(defaultTestTranslateDeps(), defaultTestConfigDeps(t), deps)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "default_provider: tencent") {
		t.Fatalf("expected default_provider: tencent, got %q", content)
	}
	if !strings.Contains(content, "secret_id: my-secret-id") {
		t.Fatalf("expected secret_id in config, got %q", content)
	}
	if !strings.Contains(content, "secret_key: my-secret-key") {
		t.Fatalf("expected secret_key in config, got %q", content)
	}
}

func TestInitAlreadyInitialized(t *testing.T) {
	deps, path := newInitTestDeps(t)

	// Create config file first
	if err := os.WriteFile(path, []byte("default_provider: codex\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := newRootCommand(defaultTestTranslateDeps(), defaultTestConfigDeps(t), deps)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	stdout := deps.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(stdout, "Already initialized") {
		t.Fatalf("expected already initialized message, got %q", stdout)
	}
}

func TestInitInvalidChoiceRetries(t *testing.T) {
	deps, path := newInitTestDeps(t)
	deps.Stdin = strings.NewReader("9\n1\n") // invalid then claude

	cmd := newRootCommand(defaultTestTranslateDeps(), defaultTestConfigDeps(t), deps)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "default_provider: claude") {
		t.Fatalf("expected default_provider: claude, got %q", string(data))
	}
}

func TestInitEOFDuringProviderSelection(t *testing.T) {
	deps, path := newInitTestDeps(t)
	deps.Stdin = strings.NewReader("") // EOF immediately, no newline

	cmd := newRootCommand(defaultTestTranslateDeps(), defaultTestConfigDeps(t), deps)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// Config file should NOT have been created
	if _, err := os.Stat(path); err == nil {
		t.Fatal("expected no config file after EOF")
	}
}

func TestInitEOFDuringCredentialInput(t *testing.T) {
	deps, path := newInitTestDeps(t)
	deps.Stdin = strings.NewReader("3\n") // select tencent
	deps.ReadSecret = func() (string, error) {
		return "", io.EOF // simulate Ctrl+D
	}

	cmd := newRootCommand(defaultTestTranslateDeps(), defaultTestConfigDeps(t), deps)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// Config file should NOT have been created
	if _, err := os.Stat(path); err == nil {
		t.Fatal("expected no config file after EOF during credentials")
	}
}
