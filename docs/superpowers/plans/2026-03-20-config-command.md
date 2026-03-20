# `qiao config` Subcommand Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `qiao config` subcommand with `get`, `set`, `list`, and `delete` operations for managing `~/.config/qiao/config.yaml` from the CLI.

**Architecture:** Config operations are methods on the existing `config.Config` struct. A new `internal/cli/config.go` file wires cobra subcommands using a `ConfigDependencies` struct for testability. The `app.Runtime` layer is not involved.

**Tech Stack:** Go, cobra, gopkg.in/yaml.v3

**Spec:** `docs/superpowers/specs/2026-03-20-config-command-design.md`

---

### Task 1: Key path parsing and validation

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for key path parsing**

Add to `internal/config/config_test.go`:

```go
func TestParseKeyPath_TopLevel(t *testing.T) {
	kind, parts, err := parseKeyPath("default_provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != keyKindTopLevel {
		t.Fatalf("expected keyKindTopLevel, got %d", kind)
	}
	if parts[0] != "default_provider" {
		t.Fatalf("expected default_provider, got %q", parts[0])
	}
}

func TestParseKeyPath_Provider(t *testing.T) {
	kind, parts, err := parseKeyPath("providers.tencent.secret_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != keyKindProvider {
		t.Fatalf("expected keyKindProvider, got %d", kind)
	}
	if parts[0] != "tencent" || parts[1] != "secret_id" {
		t.Fatalf("expected [tencent secret_id], got %v", parts)
	}
}

func TestParseKeyPath_BareProviders(t *testing.T) {
	_, _, err := parseKeyPath("providers")
	if err == nil {
		t.Fatal("expected error for bare providers key")
	}
}

func TestParseKeyPath_UnknownTopLevel(t *testing.T) {
	_, _, err := parseKeyPath("foo")
	if err == nil {
		t.Fatal("expected error for unknown top-level key")
	}
}

func TestParseKeyPath_WrongSegmentCount(t *testing.T) {
	_, _, err := parseKeyPath("a.b.c.d")
	if err == nil {
		t.Fatal("expected error for four segments")
	}

	_, _, err = parseKeyPath("providers.tencent")
	if err == nil {
		t.Fatal("expected error for two segments")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestParseKeyPath -v`
Expected: FAIL — `parseKeyPath` undefined

- [ ] **Step 3: Implement parseKeyPath**

Add to `internal/config/config.go`:

```go
type keyKind int

const (
	keyKindTopLevel keyKind = iota
	keyKindProvider
)

var validTopLevelKeys = map[string]bool{
	"default_provider": true,
	"default_source":   true,
	"default_target":   true,
}

func parseKeyPath(key string) (keyKind, []string, error) {
	parts := strings.Split(key, ".")

	switch len(parts) {
	case 1:
		if parts[0] == "providers" {
			return 0, nil, fmt.Errorf("%q is not a scalar key; use \"providers.<name>.<field>\"", key)
		}
		if !validTopLevelKeys[parts[0]] {
			return 0, nil, fmt.Errorf("unknown key %q: valid keys are default_provider, default_source, default_target", key)
		}
		return keyKindTopLevel, parts, nil
	case 3:
		if parts[0] != "providers" {
			return 0, nil, fmt.Errorf("invalid key %q: use \"field\" for top-level or \"providers.<name>.<field>\" for provider config", key)
		}
		return keyKindProvider, parts[1:], nil
	default:
		return 0, nil, fmt.Errorf("invalid key %q: use \"field\" for top-level or \"providers.<name>.<field>\" for provider config", key)
	}
}
```

Add imports `"fmt"` and `"strings"` to the import block.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestParseKeyPath -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add key path parsing with dot-notation validation"
```

---

### Task 2: Config.Get and Config.Set methods

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for Get**

Add to `internal/config/config_test.go`:

```go
func TestGet_TopLevel(t *testing.T) {
	cfg := Config{DefaultProvider: "tencent"}
	val, err := cfg.Get("default_provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "tencent" {
		t.Fatalf("expected tencent, got %q", val)
	}
}

func TestGet_TopLevelNotSet(t *testing.T) {
	cfg := Config{}
	_, err := cfg.Get("default_provider")
	if err == nil {
		t.Fatal("expected error for unset key")
	}
}

func TestGet_Provider(t *testing.T) {
	cfg := Config{
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123"},
		},
	}
	val, err := cfg.Get("providers.tencent.secret_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "AKID123" {
		t.Fatalf("expected AKID123, got %q", val)
	}
}

func TestGet_ProviderNotFound(t *testing.T) {
	cfg := Config{Providers: map[string]map[string]string{}}
	_, err := cfg.Get("providers.tencent.secret_id")
	if err == nil {
		t.Fatal("expected error for missing provider key")
	}
}

func TestGet_InvalidKey(t *testing.T) {
	cfg := Config{}
	_, err := cfg.Get("foo")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestGet_ -v`
Expected: FAIL — `cfg.Get` undefined

- [ ] **Step 3: Implement Get**

Add to `internal/config/config.go`:

```go
func (c Config) Get(key string) (string, error) {
	kind, parts, err := parseKeyPath(key)
	if err != nil {
		return "", err
	}

	switch kind {
	case keyKindTopLevel:
		val := c.getTopLevel(parts[0])
		if val == "" {
			return "", fmt.Errorf("key %q not found", key)
		}
		return val, nil
	case keyKindProvider:
		if c.Providers == nil {
			return "", fmt.Errorf("key %q not found", key)
		}
		providerMap, ok := c.Providers[parts[0]]
		if !ok {
			return "", fmt.Errorf("key %q not found", key)
		}
		val, ok := providerMap[parts[1]]
		if !ok || val == "" {
			return "", fmt.Errorf("key %q not found", key)
		}
		return val, nil
	}
	return "", fmt.Errorf("key %q not found", key)
}

func (c Config) getTopLevel(field string) string {
	switch field {
	case "default_provider":
		return c.DefaultProvider
	case "default_source":
		return c.DefaultSource
	case "default_target":
		return c.DefaultTarget
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestGet_ -v`
Expected: PASS

- [ ] **Step 5: Write failing tests for Set**

Add to `internal/config/config_test.go`:

```go
func TestSet_TopLevel(t *testing.T) {
	cfg := Config{}
	if err := cfg.Set("default_provider", "tencent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProvider != "tencent" {
		t.Fatalf("expected tencent, got %q", cfg.DefaultProvider)
	}
}

func TestSet_Provider(t *testing.T) {
	cfg := Config{}
	if err := cfg.Set("providers.tencent.secret_id", "AKID123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Providers["tencent"]["secret_id"] != "AKID123" {
		t.Fatalf("expected AKID123, got %q", cfg.Providers["tencent"]["secret_id"])
	}
}

func TestSet_ProviderCreatesIntermediateMaps(t *testing.T) {
	cfg := Config{}
	if err := cfg.Set("providers.claude.model", "opus"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Providers == nil {
		t.Fatal("expected providers map to be created")
	}
	if cfg.Providers["claude"]["model"] != "opus" {
		t.Fatalf("expected opus, got %q", cfg.Providers["claude"]["model"])
	}
}

func TestSet_OverwritesExisting(t *testing.T) {
	cfg := Config{DefaultProvider: "codex"}
	if err := cfg.Set("default_provider", "tencent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProvider != "tencent" {
		t.Fatalf("expected tencent, got %q", cfg.DefaultProvider)
	}
}

func TestSet_InvalidKey(t *testing.T) {
	cfg := Config{}
	if err := cfg.Set("foo", "bar"); err == nil {
		t.Fatal("expected error for invalid key")
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestSet_ -v`
Expected: FAIL — `cfg.Set` undefined

- [ ] **Step 7: Implement Set**

Add to `internal/config/config.go`:

```go
func (c *Config) Set(key, value string) error {
	kind, parts, err := parseKeyPath(key)
	if err != nil {
		return err
	}

	switch kind {
	case keyKindTopLevel:
		c.setTopLevel(parts[0], value)
	case keyKindProvider:
		if c.Providers == nil {
			c.Providers = map[string]map[string]string{}
		}
		if c.Providers[parts[0]] == nil {
			c.Providers[parts[0]] = map[string]string{}
		}
		c.Providers[parts[0]][parts[1]] = value
	}
	return nil
}

func (c *Config) setTopLevel(field, value string) {
	switch field {
	case "default_provider":
		c.DefaultProvider = value
	case "default_source":
		c.DefaultSource = value
	case "default_target":
		c.DefaultTarget = value
	}
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestSet_ -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add Get and Set methods with dot-path support"
```

---

### Task 3: Config.Delete and Config.List methods

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for Delete**

Add to `internal/config/config_test.go`:

```go
func TestDelete_TopLevel(t *testing.T) {
	cfg := Config{DefaultProvider: "tencent"}
	if err := cfg.Delete("default_provider"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProvider != "" {
		t.Fatalf("expected empty, got %q", cfg.DefaultProvider)
	}
}

func TestDelete_Provider(t *testing.T) {
	cfg := Config{
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123", "region": "ap-guangzhou"},
		},
	}
	if err := cfg.Delete("providers.tencent.secret_id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cfg.Providers["tencent"]["secret_id"]; ok {
		t.Fatal("expected secret_id to be deleted")
	}
	if cfg.Providers["tencent"]["region"] != "ap-guangzhou" {
		t.Fatal("expected region to remain")
	}
}

func TestDelete_ProviderCleansUpEmptyMap(t *testing.T) {
	cfg := Config{
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123"},
		},
	}
	if err := cfg.Delete("providers.tencent.secret_id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cfg.Providers["tencent"]; ok {
		t.Fatal("expected empty provider map to be removed")
	}
}

func TestDelete_NotFound(t *testing.T) {
	cfg := Config{}
	err := cfg.Delete("default_provider")
	if err == nil {
		t.Fatal("expected error for deleting unset key")
	}
}

func TestDelete_ProviderNotFound(t *testing.T) {
	cfg := Config{Providers: map[string]map[string]string{}}
	err := cfg.Delete("providers.tencent.secret_id")
	if err == nil {
		t.Fatal("expected error for deleting missing provider key")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestDelete_ -v`
Expected: FAIL — `cfg.Delete` undefined

- [ ] **Step 3: Implement Delete**

Add to `internal/config/config.go`:

```go
func (c *Config) Delete(key string) error {
	kind, parts, err := parseKeyPath(key)
	if err != nil {
		return err
	}

	switch kind {
	case keyKindTopLevel:
		val := c.getTopLevel(parts[0])
		if val == "" {
			return fmt.Errorf("key %q not found", key)
		}
		c.setTopLevel(parts[0], "")
	case keyKindProvider:
		if c.Providers == nil {
			return fmt.Errorf("key %q not found", key)
		}
		providerMap, ok := c.Providers[parts[0]]
		if !ok {
			return fmt.Errorf("key %q not found", key)
		}
		if _, ok := providerMap[parts[1]]; !ok {
			return fmt.Errorf("key %q not found", key)
		}
		delete(providerMap, parts[1])
		if len(providerMap) == 0 {
			delete(c.Providers, parts[0])
		}
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestDelete_ -v`
Expected: PASS

- [ ] **Step 5: Write failing tests for List**

Add to `internal/config/config_test.go`:

```go
func TestList_FullConfig(t *testing.T) {
	cfg := Config{
		DefaultProvider: "tencent",
		DefaultTarget:   "en",
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123"},
		},
	}
	got := cfg.List()
	want := map[string]string{
		"default_provider":           "tencent",
		"default_target":             "en",
		"providers.tencent.secret_id": "AKID123",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestList_SkipsEmptyValues(t *testing.T) {
	cfg := Config{DefaultProvider: "codex"}
	got := cfg.List()
	if _, ok := got["default_source"]; ok {
		t.Fatal("expected empty default_source to be skipped")
	}
	if got["default_provider"] != "codex" {
		t.Fatalf("expected codex, got %q", got["default_provider"])
	}
}

func TestList_EmptyConfig(t *testing.T) {
	cfg := Config{}
	got := cfg.List()
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}
```

- [ ] **Step 6: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestList_ -v`
Expected: FAIL — `cfg.List` undefined

- [ ] **Step 7: Implement List**

Add to `internal/config/config.go`:

```go
func (c Config) List() map[string]string {
	result := map[string]string{}

	if c.DefaultProvider != "" {
		result["default_provider"] = c.DefaultProvider
	}
	if c.DefaultSource != "" {
		result["default_source"] = c.DefaultSource
	}
	if c.DefaultTarget != "" {
		result["default_target"] = c.DefaultTarget
	}

	for provider, fields := range c.Providers {
		for field, value := range fields {
			if value != "" {
				result[fmt.Sprintf("providers.%s.%s", provider, field)] = value
			}
		}
	}

	return result
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestList_ -v`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add Delete and List methods"
```

---

### Task 4: Config.Save method

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests for Save**

Add to `internal/config/config_test.go`:

```go
func TestSave_WritesYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{
		DefaultProvider: "tencent",
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123"},
		},
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load after save: %v", err)
	}

	if loaded.DefaultProvider != "tencent" {
		t.Fatalf("expected tencent, got %q", loaded.DefaultProvider)
	}
	if loaded.Providers["tencent"]["secret_id"] != "AKID123" {
		t.Fatalf("expected AKID123, got %q", loaded.Providers["tencent"]["secret_id"])
	}
}

func TestSave_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.yaml")

	cfg := Config{DefaultProvider: "codex"}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestSave_OmitsEmptyProviders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{DefaultProvider: "codex"}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if strings.Contains(string(data), "providers") {
		t.Fatalf("expected no providers key in output, got:\n%s", data)
	}
}
```

Add `"strings"` to the test file imports if not already present.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestSave_ -v`
Expected: FAIL — `cfg.Save` undefined

- [ ] **Step 3: Implement Save**

First, update the `Providers` struct tag in `internal/config/config.go` to add `omitempty` so empty/nil providers are not written to YAML:

```go
Providers map[string]map[string]string `yaml:"providers,omitempty"`
```

Then add the `Save` method to `internal/config/config.go`:

```go
func (c Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestSave_ -v`
Expected: PASS

- [ ] **Step 5: Run all config tests**

Run: `go test ./internal/config/ -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add Save method for writing YAML config to disk"
```

---

### Task 5: CLI config subcommands

**Files:**
- Create: `internal/cli/config.go`
- Create: `internal/cli/config_test.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Write failing tests for config subcommands**

Create `internal/cli/config_test.go`:

```go
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

	cmd := newRootCommand(defaultTestTranslateDeps(), deps)
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
	cmd2 := newRootCommand(defaultTestTranslateDeps(), deps2)
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

	cmd := newRootCommand(defaultTestTranslateDeps(), deps)
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

	cmd := newRootCommand(defaultTestTranslateDeps(), deps)
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

	cmd := newRootCommand(defaultTestTranslateDeps(), deps)
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
	cmd2 := newRootCommand(defaultTestTranslateDeps(), deps2)
	cmd2.SetArgs([]string{"config", "get", "default_provider"})

	if err := cmd2.Execute(); err == nil {
		t.Fatal("expected error after deleting key")
	}
}

func TestConfigDelete_NotFound(t *testing.T) {
	deps, _ := newConfigTestDeps(t)

	cmd := newRootCommand(defaultTestTranslateDeps(), deps)
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
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestConfig -v`
Expected: FAIL — `ConfigDependencies` and `configureConfigCommand` undefined

- [ ] **Step 3: Implement config.go**

Create `internal/cli/config.go`:

```go
package cli

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"qiao/internal/config"
)

type ConfigDependencies struct {
	Stdout     io.Writer
	Stderr     io.Writer
	ConfigPath string
}

func configureConfigCommand(root *cobra.Command, deps ConfigDependencies) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	configCmd.AddCommand(newConfigSetCommand(deps))
	configCmd.AddCommand(newConfigGetCommand(deps))
	configCmd.AddCommand(newConfigListCommand(deps))
	configCmd.AddCommand(newConfigDeleteCommand(deps))

	root.AddCommand(configCmd)
}

func newConfigSetCommand(deps ConfigDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			fileExisted := true
			if _, err := os.Stat(deps.ConfigPath); os.IsNotExist(err) {
				fileExisted = false
			}

			cfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				return err
			}

			if err := cfg.Set(key, value); err != nil {
				return err
			}

			if err := cfg.Save(deps.ConfigPath); err != nil {
				return err
			}

			if !fileExisted {
				fmt.Fprintf(deps.Stderr, "Created config file: %s\n", deps.ConfigPath)
			}

			return nil
		},
	}
}

func newConfigGetCommand(deps ConfigDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				return err
			}

			val, err := cfg.Get(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintln(deps.Stdout, val)
			return nil
		},
	}
}

func newConfigListCommand(deps ConfigDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				return err
			}

			entries := cfg.List()
			keys := make([]string, 0, len(entries))
			for k := range entries {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Fprintf(deps.Stdout, "%s=%s\n", k, entries[k])
			}

			return nil
		},
	}
}

func newConfigDeleteCommand(deps ConfigDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				return err
			}

			if err := cfg.Delete(args[0]); err != nil {
				return err
			}

			return cfg.Save(deps.ConfigPath)
		},
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/cli/ -run TestConfig -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/config.go internal/cli/config_test.go
git commit -m "feat(cli): add config subcommand with get/set/list/delete"
```

---

### Task 6: Register config command in root and run full test suite

**Files:**
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Wire config command into root**

In `internal/cli/root.go`, modify `newRootCommand` to accept `ConfigDependencies` as a second parameter. This avoids double-registration in tests.

Update the public entry point:

```go
func NewRootCommand() *cobra.Command {
	return newRootCommand(defaultTranslateDependencies(), defaultConfigDependencies())
}
```

Update the internal constructor:

```go
func newRootCommand(deps TranslateDependencies, cfgDeps ConfigDependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "qiao [text]",
		Short:        "Translate text from the command line",
		Long:         "qiao is a provider-oriented translation CLI. Supports Codex and Claude Code as providers.",
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}

	configureTranslateCommand(cmd, deps)
	configureProvidersCommand(cmd, deps)
	configureConfigCommand(cmd, cfgDeps)

	return cmd
}
```

Add `defaultConfigDependencies` function:

```go
func defaultConfigDependencies() ConfigDependencies {
	configPath, _ := config.DefaultPath()
	return ConfigDependencies{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		ConfigPath: configPath,
	}
}
```

Add `"qiao/internal/config"` to the import block.

**Update all existing test call sites** in `translate_test.go` and `providers_test.go` — every call to `newRootCommand(TranslateDependencies{...})` becomes `newRootCommand(TranslateDependencies{...}, ConfigDependencies{})`. An empty `ConfigDependencies` is fine since those tests don't exercise config commands. (Note: `config_test.go` call sites were already updated in Task 5 to use the two-argument form.)

- [ ] **Step 2: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

- [ ] **Step 3: Manual smoke test**

Run:
```bash
go run ./cmd/qiao config set default_provider tencent
go run ./cmd/qiao config get default_provider
go run ./cmd/qiao config list
go run ./cmd/qiao config delete default_provider
go run ./cmd/qiao config --help
```

Expected:
- `set` prints creation message if first time
- `get` prints `tencent`
- `list` prints `default_provider=tencent`
- `delete` succeeds silently
- `--help` shows all four subcommands

- [ ] **Step 4: Commit**

```bash
git add internal/cli/root.go
git commit -m "feat(cli): register config subcommand in root command"
```
