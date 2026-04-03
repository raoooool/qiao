# `qiao init` Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `qiao init` subcommand that guides first-time users through selecting a default provider and configuring its required credentials, and block translation when uninitialized.

**Architecture:** Extend the registry to carry per-provider config metadata (`core.ConfigField`). Add a new `init` CLI command that reads the metadata and interactively prompts the user. Add a pre-check in the translate command that requires initialization (config file existence) unless `--provider` is explicitly set.

**Tech Stack:** Go, cobra, golang.org/x/term (for secret input)

**Spec:** `docs/superpowers/specs/2026-03-20-init-command-design.md`

---

### Task 1: Add `ConfigField` type to core

**Files:**
- Modify: `internal/core/types.go:1-27`
- Test: `internal/core/types_test.go`

- [ ] **Step 1: Add `ConfigField` struct to `types.go`**

Append after the `Translator` interface (line 26):

```go
// ConfigField describes a provider configuration field for interactive setup.
type ConfigField struct {
	Key      string
	Label    string
	Required bool
	Secret   bool
}
```

- [ ] **Step 2: Run existing tests to verify no breakage**

Run: `go test ./internal/core/`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/core/types.go
git commit -m "feat(core): add ConfigField type for provider config metadata"
```

---

### Task 2: Add `ConfigFields` variable to each provider

**Files:**
- Modify: `internal/providers/codex/provider.go:12-13`
- Modify: `internal/providers/claude/provider.go:12-13`
- Modify: `internal/providers/tencent/provider.go:19-28`

- [ ] **Step 1: Add `ConfigFields` to codex provider**

Add after the imports in `internal/providers/codex/provider.go`, before the `commandRunner` type:

```go
// ConfigFields declares the configuration fields for the codex provider.
var ConfigFields []core.ConfigField
```

The import for `core` is already present.

- [ ] **Step 2: Add `ConfigFields` to claude provider**

Add after the imports in `internal/providers/claude/provider.go`, before the `commandRunner` type:

```go
// ConfigFields declares the configuration fields for the claude provider.
var ConfigFields []core.ConfigField
```

- [ ] **Step 3: Add `ConfigFields` to tencent provider**

Add after the imports in `internal/providers/tencent/provider.go`, before the `httpClient` type:

```go
// ConfigFields declares the configuration fields for the tencent provider.
var ConfigFields = []core.ConfigField{
	{Key: "secret_id", Label: "Secret ID", Required: true, Secret: true},
	{Key: "secret_key", Label: "Secret Key", Required: true, Secret: true},
}
```

- [ ] **Step 4: Run all provider tests**

Run: `go test ./internal/providers/...`
Expected: PASS (all existing tests still pass)

- [ ] **Step 5: Commit**

```bash
git add internal/providers/codex/provider.go internal/providers/claude/provider.go internal/providers/tencent/provider.go
git commit -m "feat(providers): add ConfigFields metadata to all providers"
```

---

### Task 3: Extend registry to store and query config fields

**Files:**
- Modify: `internal/providers/registry/registry.go:1-45`
- Modify: `internal/providers/registry/registry_test.go`

- [ ] **Step 1: Write failing test for `ConfigFields` query**

Append to `internal/providers/registry/registry_test.go`:

```go
func TestConfigFieldsReturnsRegisteredFields(t *testing.T) {
	r := New()

	fields := []core.ConfigField{
		{Key: "api_key", Label: "API Key", Required: true, Secret: true},
	}

	r.Register("test", func(config.Config) (core.Translator, error) {
		return testTranslator{name: "test"}, nil
	}, fields)

	got := r.ConfigFields("test")
	if len(got) != 1 {
		t.Fatalf("expected 1 config field, got %d", len(got))
	}
	if got[0].Key != "api_key" {
		t.Fatalf("expected key api_key, got %q", got[0].Key)
	}
}

func TestConfigFieldsReturnsNilForUnknownProvider(t *testing.T) {
	r := New()
	got := r.ConfigFields("missing")
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/registry/ -run TestConfigFields -v`
Expected: compilation error — `Register` has wrong signature

- [ ] **Step 3: Update registry internals**

Replace the entire `internal/providers/registry/registry.go` with:

```go
package registry

import (
	"fmt"
	"slices"

	"qiao/internal/config"
	"qiao/internal/core"
)

type Factory func(config.Config) (core.Translator, error)

type providerInfo struct {
	factory      Factory
	configFields []core.ConfigField
}

type Registry struct {
	providers map[string]providerInfo
}

func New() *Registry {
	return &Registry{
		providers: map[string]providerInfo{},
	}
}

func (r *Registry) Register(name string, factory Factory, fields []core.ConfigField) {
	r.providers[name] = providerInfo{
		factory:      factory,
		configFields: fields,
	}
}

func (r *Registry) Resolve(name string, cfg config.Config) (core.Translator, error) {
	info, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}

	return info.factory(cfg)
}

func (r *Registry) Providers() []string {
	providers := make([]string, 0, len(r.providers))
	for name := range r.providers {
		providers = append(providers, name)
	}

	slices.Sort(providers)

	return providers
}

func (r *Registry) ConfigFields(name string) []core.ConfigField {
	info, ok := r.providers[name]
	if !ok {
		return nil
	}

	return info.configFields
}
```

- [ ] **Step 4: Fix existing registry tests to pass `nil` as third arg to `Register`**

Update each `r.Register(...)` call in the existing tests to add `, nil` as the third argument. For example:

```go
r.Register("google", func(config.Config) (core.Translator, error) {
    return testTranslator{name: "google"}, nil
}, nil)
```

Apply this to all three existing test functions: `TestRegisterAndResolveProvider`, `TestResolveUnknownProvider`, and `TestProvidersReturnsStableOrder`.

- [ ] **Step 5: Run all registry tests**

Run: `go test ./internal/providers/registry/ -v`
Expected: PASS (all tests including the two new ones)

- [ ] **Step 6: Commit**

```bash
git add internal/providers/registry/registry.go internal/providers/registry/registry_test.go
git commit -m "feat(registry): extend Register to accept config field metadata"
```

---

### Task 4: Update `app.go` registration calls and add `ProviderConfigFields`

**Files:**
- Modify: `internal/app/app.go:32-43` (Register calls) and append new method
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Write failing test for `ProviderConfigFields`**

Append to `internal/app/app_test.go`:

```go
func TestProviderConfigFieldsReturnsTencentFields(t *testing.T) {
	runtime := New(config.Config{})

	fields := runtime.ProviderConfigFields("tencent")
	if len(fields) != 2 {
		t.Fatalf("expected 2 config fields for tencent, got %d", len(fields))
	}
	if fields[0].Key != "secret_id" {
		t.Fatalf("expected first field key secret_id, got %q", fields[0].Key)
	}
	if fields[1].Key != "secret_key" {
		t.Fatalf("expected second field key secret_key, got %q", fields[1].Key)
	}
}

func TestProviderConfigFieldsReturnsEmptyForCodex(t *testing.T) {
	runtime := New(config.Config{})

	fields := runtime.ProviderConfigFields("codex")
	if len(fields) != 0 {
		t.Fatalf("expected 0 config fields for codex, got %d", len(fields))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -run TestProviderConfigFields -v`
Expected: compilation error — `Register` call has wrong number of args

- [ ] **Step 3: Update `app.go`**

Update the three `Register` calls in `New()` to pass the provider `ConfigFields`:

```go
r.registry.Register("claude", claudeprovider.New, claudeprovider.ConfigFields)
r.registry.Register("codex", codexprovider.New, codexprovider.ConfigFields)
r.registry.Register("tencent", tencentprovider.New, tencentprovider.ConfigFields)
```

Add the new method after `ListProviders()`:

```go
func (r *Runtime) ProviderConfigFields(name string) []core.ConfigField {
	return r.registry.ConfigFields(name)
}
```

Add `"qiao/internal/core"` to the imports.

- [ ] **Step 4: Run all app tests**

Run: `go test ./internal/app/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go
git commit -m "feat(app): pass config field metadata to registry and expose ProviderConfigFields"
```

---

### Task 5: Add `golang.org/x/term` dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add dependency**

Run: `go get golang.org/x/term`

- [ ] **Step 2: Tidy modules**

Run: `go mod tidy`

- [ ] **Step 3: Verify build**

Run: `go build ./cmd/qiao`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add golang.org/x/term dependency for secret input"
```

---

### Task 6: Implement `qiao init` command

**Files:**
- Create: `internal/cli/init.go`
- Create: `internal/cli/init_test.go`
- Modify: `internal/cli/root.go:14-23` (add `InitDependencies` fields to wiring)
- Modify: `internal/cli/root.go:25-48` (update `NewRootCommand` and `newRootCommand`)

- [ ] **Step 1: Write failing tests for `qiao init`**

Create `internal/cli/init_test.go`:

```go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qiao/internal/core"
)

func newInitTestDeps(t *testing.T) (InitDependencies, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	return InitDependencies{
		Stdin:  strings.NewReader(""),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
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

func TestInitSelectsDefaultProvider(t *testing.T) {
	deps, path := newInitTestDeps(t)
	deps.Stdin = strings.NewReader("\n") // empty input = default (codex)

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
```

Add `"io"` to the imports in `init_test.go`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run TestInit -v`
Expected: compilation error — `InitDependencies` not defined

- [ ] **Step 3: Create `internal/cli/init.go`**

```go
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"qiao/internal/config"
	"qiao/internal/core"
)

type InitDependencies struct {
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	ConfigPath    string
	ListProviders func() []string
	ConfigFields  func(string) []core.ConfigField
	ReadSecret    func() (string, error)
}

func configureInitCommand(root *cobra.Command, deps InitDependencies) {
	root.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Set up qiao for first use",
		Long:  "Interactive setup wizard that configures the default translation provider and any required credentials.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(deps)
		},
	})
}

func runInit(deps InitDependencies) error {
	// Check if already initialized
	if _, err := os.Stat(deps.ConfigPath); err == nil {
		fmt.Fprintln(deps.Stdout, "Already initialized. Use \"qiao config\" to modify settings.")
		return nil
	}

	scanner := bufio.NewScanner(deps.Stdin)

	// List providers
	providers := deps.ListProviders()
	defaultIndex := -1
	for i, p := range providers {
		if p == "codex" {
			defaultIndex = i
		}
	}
	if defaultIndex == -1 {
		defaultIndex = 0
	}

	fmt.Fprintln(deps.Stdout, "Select a default translation provider:")
	for i, p := range providers {
		if i == defaultIndex {
			fmt.Fprintf(deps.Stdout, "  [%d] %s (default)\n", i+1, p)
		} else {
			fmt.Fprintf(deps.Stdout, "  [%d] %s\n", i+1, p)
		}
	}
	fmt.Fprintf(deps.Stdout, "Enter number (default %d): ", defaultIndex+1)

	// Read provider selection
	selectedProvider := ""
	for {
		if !scanner.Scan() {
			return nil // EOF — exit cleanly
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			selectedProvider = providers[defaultIndex]
			break
		}
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(providers) {
			fmt.Fprint(deps.Stdout, "Invalid choice, try again: ")
			continue
		}
		selectedProvider = providers[num-1]
		break
	}

	// Collect required config fields
	fields := deps.ConfigFields(selectedProvider)
	providerConfig := map[string]string{}

	for _, field := range fields {
		if !field.Required {
			continue
		}

		var value string
		if field.Secret {
			fmt.Fprintf(deps.Stdout, "%s: ", field.Label)
			for {
				val, err := deps.ReadSecret()
				if err != nil {
					return nil // EOF or error — exit cleanly
				}
				value = strings.TrimSpace(val)
				if value != "" {
					break
				}
				fmt.Fprintf(deps.Stdout, "%s is required: ", field.Label)
			}
		} else {
			fmt.Fprintf(deps.Stdout, "%s: ", field.Label)
			for {
				if !scanner.Scan() {
					return nil // EOF — exit cleanly
				}
				value = strings.TrimSpace(scanner.Text())
				if value != "" {
					break
				}
				fmt.Fprintf(deps.Stdout, "%s is required: ", field.Label)
			}
		}
		providerConfig[field.Key] = value
	}

	// Build and save config
	cfg := config.Config{
		DefaultProvider: selectedProvider,
	}
	if len(providerConfig) > 0 {
		cfg.Providers = map[string]map[string]string{
			selectedProvider: providerConfig,
		}
	}

	if err := cfg.Save(deps.ConfigPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(deps.Stdout, "Configuration saved to %s\n", deps.ConfigPath)
	return nil
}
```

- [ ] **Step 4: Update `root.go` to wire `InitDependencies`**

Update `internal/cli/root.go`. Changes:

1. Add `"qiao/internal/core"` to imports.
2. Change `NewRootCommand` to also create init deps:

```go
func NewRootCommand() *cobra.Command {
	return newRootCommand(defaultTranslateDependencies(), defaultConfigDependencies(), defaultInitDependencies())
}
```

3. Change `newRootCommand` signature to accept `InitDependencies`:

```go
func newRootCommand(deps TranslateDependencies, cfgDeps ConfigDependencies, initDeps InitDependencies) *cobra.Command {
```

4. Add `configureInitCommand(cmd, initDeps)` after the existing `configure*` calls.

5. Add `defaultInitDependencies`. Note: use `app.New(config.Config{})` instead of `app.Load("")` to avoid a redundant config load — the init command only needs static registry metadata (provider names and config fields), not the user's config:

```go
func defaultInitDependencies() InitDependencies {
	configPath, _ := config.DefaultPath()
	runtime := app.New(config.Config{})

	return InitDependencies{
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		ConfigPath:    configPath,
		ListProviders: func() []string { return runtime.ListProviders() },
		ConfigFields:  func(name string) []core.ConfigField { return runtime.ProviderConfigFields(name) },
		ReadSecret:    defaultReadSecret,
	}
}
```

6. Add `defaultReadSecret` (using `golang.org/x/term`). Note: `ReadSecret` does NOT print its own prompt — the caller (`runInit`) handles all prompting to avoid double-prompt bugs on retry:

```go
func defaultReadSecret() (string, error) {
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stdout) // newline after hidden input
	if err != nil {
		return "", err
	}
	return string(password), nil
}
```

7. Add `"golang.org/x/term"` to imports.

- [ ] **Step 5: Add `defaultTestConfigDeps` helper and update existing test call sites**

In `internal/cli/config_test.go`, the existing `defaultTestTranslateDeps()` helper is already defined in `translate_test.go`. We need a helper for `InitDependencies` in tests and to update all `newRootCommand` call sites.

Add to `internal/cli/init_test.go`:

```go
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
```

Update **all** existing `newRootCommand(...)` call sites across test files to pass the third `InitDependencies` argument:

In `internal/cli/translate_test.go`: every `newRootCommand(TranslateDependencies{...}, ConfigDependencies{})` becomes `newRootCommand(TranslateDependencies{...}, ConfigDependencies{}, InitDependencies{})`.

In `internal/cli/config_test.go`: every `newRootCommand(defaultTestTranslateDeps(), deps)` becomes `newRootCommand(defaultTestTranslateDeps(), deps, InitDependencies{})`.

In `internal/cli/providers_test.go`: update similarly.

- [ ] **Step 6: Run all CLI tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/cli/init.go internal/cli/init_test.go internal/cli/root.go internal/cli/translate_test.go internal/cli/config_test.go internal/cli/providers_test.go
git commit -m "feat(cli): add qiao init command with interactive provider setup"
```

---

### Task 7: Add translation pre-check for uninitialized state

**Files:**
- Modify: `internal/cli/root.go:14-23` (add fields to `TranslateDependencies`)
- Modify: `internal/cli/translate.go:27-35` (add pre-check)
- Modify: `internal/cli/translate_test.go`

- [ ] **Step 1: Write failing tests for the pre-check**

Append to `internal/cli/translate_test.go`:

```go
func TestTranslateExitsWhenNotInitialized(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ResolveProvider: fixedProviderResolver(&fakeTranslator{}),
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		ConfigPath:      "/nonexistent/path/config.yaml",
		FileExists:      func(string) bool { return false },
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"hello"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when not initialized")
	}

	if !strings.Contains(stderr.String(), "qiao init") {
		t.Fatalf("expected init hint in stderr, got %q", stderr.String())
	}
}

func TestTranslateSkipsCheckWhenProviderFlagSet(t *testing.T) {
	translator := &fakeTranslator{}
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		ConfigPath:      "/nonexistent/path/config.yaml",
		FileExists:      func(string) bool { return false },
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"--provider", "codex", "hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success with explicit --provider, got: %v", err)
	}

	if !strings.Contains(stdout.String(), "translated:hello") {
		t.Fatalf("expected translation output, got %q", stdout.String())
	}
}

func TestTranslateProceedsWhenInitialized(t *testing.T) {
	translator := &fakeTranslator{}
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		ConfigPath:      "/some/path/config.yaml",
		FileExists:      func(string) bool { return true },
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success when initialized, got: %v", err)
	}

	if !strings.Contains(stdout.String(), "translated:hello") {
		t.Fatalf("expected translation output, got %q", stdout.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/cli/ -run "TestTranslate(ExitsWhen|SkipsCheck|ProceedsWhen)" -v`
Expected: compilation error — `ConfigPath` and `FileExists` not in `TranslateDependencies`

- [ ] **Step 3: Add `ConfigPath` and `FileExists` to `TranslateDependencies`**

In `internal/cli/root.go`, add two fields to `TranslateDependencies`:

```go
type TranslateDependencies struct {
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
	ResolveProvider func(string) (core.Translator, error)
	ListProviders   func() []string
	DefaultProvider string
	DefaultSource   string
	DefaultTarget   string
	ConfigPath      string
	FileExists      func(string) bool
}
```

Update `defaultTranslateDependencies()` to populate these fields:

```go
configPath, _ := config.DefaultPath()
```

Add to both return paths:

```go
ConfigPath: configPath,
FileExists: func(path string) bool {
    _, err := os.Stat(path)
    return err == nil
},
```

- [ ] **Step 4: Add pre-check to `translate.go`**

In `internal/cli/translate.go`, inside `RunE`, add **after** the `isTerminal` / help check (line 28-30) and **before** `resolveInput`. This ensures the check runs before stdin is consumed, so piped input is not lost when the check fails:

```go
		// Pre-check: require init unless --provider is explicitly set
		if !cmd.Flags().Changed("provider") && deps.FileExists != nil && !deps.FileExists(deps.ConfigPath) {
			fmt.Fprintln(deps.Stderr, "Tip: Run \"qiao init\" to set up your default provider.")
			return errors.New("not initialized")
		}
```

The full order of the `RunE` body becomes:
1. `isTerminal` / help check
2. **Init pre-check** (new)
3. `resolveInput`
4. Provider resolution
5. Translation

- [ ] **Step 5: Update `defaultTestTranslateDeps` to include new fields**

In `internal/cli/config_test.go`, update `defaultTestTranslateDeps()`:

```go
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
```

Also update **every** inline `TranslateDependencies{...}` literal to include `FileExists: func(string) bool { return true }` (so existing tests behave as "initialized"). The complete list of call sites:

- `internal/cli/translate_test.go` — 8 tests: `TestTranslateUsesPositionalInput`, `TestTranslateUsesStdinWhenPositionalInputMissing`, `TestTranslatePositionalInputWinsOverStdin`, `TestTranslateRequiresInput`, `TestTranslateJSONOutput`, `TestTranslateVerboseOutput`, `TestTranslateNoVerboseByDefault`, `TestTranslateVerboseOnError`
- `internal/cli/providers_test.go` — 2 tests: `TestProvidersListsRegisteredProviders`, `TestProvidersOutputIsStableForShellUse`

- [ ] **Step 6: Run all CLI tests**

Run: `go test ./internal/cli/ -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/cli/root.go internal/cli/translate.go internal/cli/translate_test.go internal/cli/config_test.go internal/cli/providers_test.go
git commit -m "feat(cli): block translation when uninitialized unless --provider is set"
```

---

### Task 8: Full integration test

**Files:**
- No new files — run existing test suites

- [ ] **Step 1: Run all tests**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: Build binary and manual smoke test**

Run: `go build -o /tmp/qiao ./cmd/qiao`

Test commands (manual):
```bash
/tmp/qiao hello          # should print init hint and exit
/tmp/qiao init           # should prompt for provider selection
/tmp/qiao hello          # should now translate (after init)
/tmp/qiao init           # should say "Already initialized"
```

- [ ] **Step 3: Commit (if any fixes needed)**

Only if adjustments were made during integration testing.
