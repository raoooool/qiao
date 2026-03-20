# Design: `qiao config` Subcommand

## Summary

Add a `qiao config` subcommand for managing the configuration file (`~/.config/qiao/config.yaml`) directly from the CLI, supporting `get`, `set`, `list`, and `delete` operations with dot-notation key paths.

## Commands

```
qiao config set <key> <value>    # Write a config value
qiao config get <key>            # Read a config value
qiao config list                 # List all config as key=value
qiao config delete <key>         # Delete a config entry
```

## Key Format

Dot-separated paths supporting two levels:

- **Top-level keys:** `default_provider`, `default_source`, `default_target`
- **Provider keys:** `providers.<name>.<field>` (e.g., `providers.tencent.secret_id`)

Invalid paths return a descriptive error:
- **Wrong segment count** (e.g., `a.b.c.d` or `a.b`): `invalid key "<key>": use "field" for top-level or "providers.<name>.<field>" for provider config`
- **Bare `providers`** (1 segment): `"providers" is not a scalar key; use "providers.<name>.<field>"`
- **Unknown top-level key** (e.g., `foo`): `unknown key "<key>": valid keys are default_provider, default_source, default_target`
- **2 segments like `providers.tencent`**: same error as wrong segment count

Values are not validated on `set` — validation happens at translation time.

## Behavior

### `set`

- The CLI layer calls `config.DefaultPath()` to get the path, then checks if the file exists before calling `Save(path)`. If the file did not exist before, print `Created config file: <path>` to stderr after saving.
- For top-level keys, write directly to the corresponding struct field.
- For provider keys, create intermediate map entries as needed.
- Existing values are overwritten silently.

### `get`

- If found, print the value (no key prefix) to stdout, exit 0.
- If not found, print `key "<key>" not found` to stderr, exit 1.

### `list`

- Output all config as flattened `key=value` lines, sorted alphabetically.
- Only include fields with non-empty values (skip zero-value/empty-string fields).
- Empty config or missing file produces no output (no error).

### `delete`

- Remove the specified key from config.
- If deleting a provider field leaves the provider map empty, remove the provider entry too.
- If the key does not exist, print `key "<key>" not found` to stderr, exit 1.

## Implementation

### New file: `internal/cli/config.go`

Contains the `config` parent command and four subcommands (`get`, `set`, `list`, `delete`). Uses a `ConfigDependencies` struct for testability:

```go
type ConfigDependencies struct {
    Stdout io.Writer
    Stderr io.Writer
    ConfigPath string // resolved via config.DefaultPath(), injectable for tests
}
```

Each subcommand:

- Loads config via `config.Load(deps.ConfigPath)`
- Calls the corresponding `Config` method
- For `set` and `delete`: resolves path via `deps.ConfigPath`, saves via `config.Save(path)`

### Modified: `internal/config/config.go`

Add methods to `Config`:

- `Get(key string) (string, error)` — parse dot path, return value or error
- `Set(key, value string) error` — parse dot path, write value, create intermediate maps
- `Delete(key string) error` — parse dot path, remove value, clean up empty provider maps
- `List() map[string]string` — return flattened key=value map of all set fields
- `Save(path string) error` — marshal to YAML, create directories if needed, write file

Key path parsing is shared via a helper that splits on `.` and validates:
- 1 segment: must be a known top-level key (`default_provider`, `default_source`, `default_target`)
- 3 segments: must start with `providers`, second segment is provider name, third is field name
- Other segment counts: return error

### Modified: `internal/cli/root.go`

Register the `config` command in `NewRootCommand()`.

### New file: `internal/cli/config_test.go`

Tests for the four subcommands using `ConfigDependencies` with temp directories and `bytes.Buffer` for stdout/stderr.

### New file: `internal/config/config_test.go` (additions)

Tests for `Get`, `Set`, `Delete`, `List`, and `Save` methods on `Config`.

### Not modified: `internal/app/`

The `config` subcommand operates directly on `config.Config`, bypassing `app.Runtime` since no provider resolution is needed.

## Examples

```bash
# Set default provider
$ qiao config set default_provider tencent

# Set provider-specific config
$ qiao config set providers.tencent.secret_id AKID_EXAMPLE
$ qiao config set providers.tencent.secret_key SECRET_EXAMPLE

# Read a value
$ qiao config get default_provider
tencent

# List all config
$ qiao config list
default_provider=tencent
providers.tencent.secret_id=AKID_EXAMPLE
providers.tencent.secret_key=SECRET_EXAMPLE

# Delete a value
$ qiao config delete providers.tencent.region

# First-time use (file doesn't exist)
$ qiao config set default_provider codex
Created config file: /home/user/.config/qiao/config.yaml
```
