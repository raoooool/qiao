package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newConfigTestDeps(t *testing.T) (ConfigDependencies, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	return ConfigDependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		ConfigPath: path,
	}, path
}

func TestConfigSet_TopLevel(t *testing.T) {
	deps, path := newConfigTestDeps(t)

	cmd := newRootCommand(defaultTestTranslateDeps(), deps, InitDependencies{})
	cmd.SetArgs([]string{"config", "set", "default_provider", "tencent"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// Verify file was created and value was set
	stderr := deps.Stderr.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "Created config file") {
		t.Fatalf("expected creation message, got %q", stderr)
	}

	// Verify value can be read back
	deps2 := ConfigDependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		ConfigPath: path,
	}
	cmd2 := newRootCommand(defaultTestTranslateDeps(), deps2, InitDependencies{})
	cmd2.SetArgs([]string{"config", "get", "default_provider"})

	if err := cmd2.Execute(); err != nil {
		t.Fatalf("execute get: %v", err)
	}
	if got := strings.TrimSpace(deps2.Stdout.(*bytes.Buffer).String()); got != "tencent" {
		t.Fatalf("expected tencent, got %q", got)
	}
}

func TestConfigGet_NotFound(t *testing.T) {
	deps, _ := newConfigTestDeps(t)

	cmd := newRootCommand(defaultTestTranslateDeps(), deps, InitDependencies{})
	cmd.SetArgs([]string{"config", "get", "default_provider"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing key")
	}
}

func TestConfigList_ShowsAllConfig(t *testing.T) {
	deps, path := newConfigTestDeps(t)

	// Write a config file first
	content := []byte("default_provider: tencent\ndefault_target: en\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := newRootCommand(defaultTestTranslateDeps(), deps, InitDependencies{})
	cmd.SetArgs([]string{"config", "list"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := deps.Stdout.(*bytes.Buffer).String()
	if !strings.Contains(out, "default_provider=tencent") {
		t.Fatalf("expected default_provider=tencent in output, got %q", out)
	}
	if !strings.Contains(out, "default_target=en") {
		t.Fatalf("expected default_target=en in output, got %q", out)
	}
}

func TestConfigDelete_RemovesKey(t *testing.T) {
	deps, path := newConfigTestDeps(t)

	content := []byte("default_provider: tencent\ndefault_target: en\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := newRootCommand(defaultTestTranslateDeps(), deps, InitDependencies{})
	cmd.SetArgs([]string{"config", "delete", "default_provider"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	// Verify it's gone
	deps2 := ConfigDependencies{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		ConfigPath: path,
	}
	cmd2 := newRootCommand(defaultTestTranslateDeps(), deps2, InitDependencies{})
	cmd2.SetArgs([]string{"config", "get", "default_provider"})

	if err := cmd2.Execute(); err == nil {
		t.Fatal("expected error after deleting key")
	}
}

func TestConfigDelete_NotFound(t *testing.T) {
	deps, _ := newConfigTestDeps(t)

	cmd := newRootCommand(defaultTestTranslateDeps(), deps, InitDependencies{})
	cmd.SetArgs([]string{"config", "delete", "default_provider"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for deleting non-existent key")
	}
}

func defaultTestTranslateDeps() TranslateDependencies {
	return TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(&fakeTranslator{}),
		ListProviders:   func() []string { return []string{"codex"} },
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		FileExists:      func(string) bool { return true },
	}
}
