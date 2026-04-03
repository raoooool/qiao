# Design: `qiao init` Command

## Overview

Add a `qiao init` subcommand that guides first-time users through selecting a default provider and configuring its required credentials. When a user runs qiao without initialization and without an explicit `--provider` flag, print a hint and exit.

## Motivation

Currently qiao silently defaults to the `codex` provider when no config file exists. Users who want to use a different provider (e.g. tencent) must manually create the config file and know which keys to set. A guided init flow makes first-run setup explicit and discoverable.

## Design

### 1. Configuration Metadata (`core.ConfigField`)

Add a `ConfigField` struct to `internal/core/types.go`:

```go
type ConfigField struct {
    Key      string // config key name, e.g. "secret_id"
    Label    string // display label, e.g. "Secret ID"
    Required bool   // whether the field is required
    Secret   bool   // whether to hide input (for credentials)
}
```

Each provider package exposes a package-level variable (not function) returning its config fields:

- **codex**: `[]core.ConfigField{}` (empty — no required config)
- **claude**: `[]core.ConfigField{}` (empty — no required config)
- **tencent**: `[]core.ConfigField{{Key: "secret_id", Label: "Secret ID", Required: true, Secret: true}, {Key: "secret_key", Label: "Secret Key", Required: true, Secret: true}}`

### 2. Registry Extension

Update `registry.Register` to accept config metadata:

```go
type ProviderInfo struct {
    Factory      Factory
    ConfigFields []core.ConfigField
}

func (r *Registry) Register(name string, factory Factory, fields []core.ConfigField)
```

Add a query method:

```go
func (r *Registry) ConfigFields(name string) []core.ConfigField
```

This allows `qiao init` to ask the registry what a provider needs without instantiating it.

### 3. `qiao init` Subcommand

New file: `internal/cli/init.go`

**Dependencies struct:**

```go
type InitDependencies struct {
    Stdin         io.Reader
    Stdout        io.Writer
    Stderr        io.Writer
    ConfigPath    string
    ListProviders func() []string
    ConfigFields  func(string) []core.ConfigField
    ReadSecret    func() (string, error) // reads secret input; caller handles prompting. Production uses term.ReadPassword
}
```

**Flow:**

1. Check if config file exists at `ConfigPath`. If yes, print `Already initialized. Use "qiao config" to modify settings.` and return nil (success, not error).
2. Print provider list with numbers. The list comes from `ListProviders()` which returns alphabetically sorted names (`claude`, `codex`, `tencent`). The default choice is `codex` — display it as such:
   ```
   Select a default translation provider:
     [1] claude
     [2] codex (default)
     [3] tencent
   Enter number (default 2):
   ```
3. Read user input. Empty input defaults to the index of `codex`. Invalid input prints `Invalid choice, try again:` and loops.
4. Query `ConfigFields(selectedProvider)` for required fields.
5. For each required field, prompt: `<Label>:` and read input. Empty input for a required field re-prompts with `<Label> is required:`. For `Secret: true` fields, print the prompt first, then call `deps.ReadSecret()` to read with terminal echo disabled. If the user sends EOF (Ctrl+D) at any prompt, exit cleanly without writing a partial config.
6. Build a `config.Config` with `DefaultProvider` and `Providers` map populated.
7. Call `config.Save(ConfigPath)` to write the config file.
8. Print `Configuration saved to <ConfigPath>`.

### 4. Translation Pre-Check

In `internal/cli/translate.go`, within `RunE`, before resolving the provider:

- If `--provider` flag was explicitly set (detect via `cmd.Flags().Changed("provider")`): skip the check, proceed normally.
- Otherwise: check if the config file exists using `os.Stat(deps.ConfigPath)` — specifically checking for `os.IsNotExist`.
  - If it does not exist: print `Tip: Run "qiao init" to set up your default provider.` to stderr and return an error (exit without translating).
  - If it exists: proceed normally.

Note: `config.Load` already returns an empty `Config{}` when the file is missing, so the translate pre-check must operate at the file level, not the config value level, to match the "config file existence = initialized" decision.

The config file existence check requires access to the config path and a file-stat function (for testability). These will be added to `TranslateDependencies`:

```go
type TranslateDependencies struct {
    // ... existing fields ...
    ConfigPath    string                       // path to config file, for init check
    FileExists    func(string) bool            // checks config file existence; production uses os.Stat
}
```

### 5. Registration Changes in `app.go`

Update provider registrations to include config fields:

```go
r.registry.Register("claude", claudeprovider.New, claudeprovider.ConfigFields)
r.registry.Register("codex", codexprovider.New, codexprovider.ConfigFields)
r.registry.Register("tencent", tencentprovider.New, tencentprovider.ConfigFields)
```

Add a method to `Runtime`:

```go
func (r *Runtime) ProviderConfigFields(name string) []core.ConfigField {
    return r.registry.ConfigFields(name)
}
```

### 6. Secret Input

For `Secret: true` config fields, the production `ReadSecret` implementation uses `golang.org/x/term.ReadPassword` to disable terminal echo. This is the only new external dependency. In tests, `ReadSecret` is injected as a simple function that reads from the test's stdin.

## User Experience

### First run without init

```
$ qiao hello
Tip: Run "qiao init" to set up your default provider.
$ echo $?
1
```

### First run with explicit provider

```
$ qiao --provider codex hello
你好
```

### Init flow

```
$ qiao init
Select a default translation provider:
  [1] claude
  [2] codex (default)
  [3] tencent
Enter number (default 2): 3
Secret ID: ****
Secret Key: ****
Configuration saved to /home/user/.config/qiao/config.yaml
```

### Already initialized

```
$ qiao init
Already initialized. Use "qiao config" to modify settings.
```

## Files Changed

| File | Change |
|------|--------|
| `internal/core/types.go` | Add `ConfigField` struct |
| `internal/providers/codex/provider.go` | Add `ConfigFields` variable |
| `internal/providers/claude/provider.go` | Add `ConfigFields` variable |
| `internal/providers/tencent/provider.go` | Add `ConfigFields` variable |
| `internal/providers/registry/registry.go` | Update `Register` signature, add `ConfigFields` method, change internal storage to `ProviderInfo` |
| `internal/app/app.go` | Update `Register` calls, add `ProviderConfigFields` method |
| `internal/cli/init.go` | New file — `qiao init` subcommand |
| `internal/cli/root.go` | Wire `InitDependencies`, register init subcommand |
| `internal/cli/translate.go` | Add config file existence check |
| `go.mod` / `go.sum` | Add `golang.org/x/term` dependency |

## Out of Scope

- Configuring default source/target language in init (can be added later)
- Re-running init to change provider (use `qiao config` instead)
- Validating credentials against the provider API during init
