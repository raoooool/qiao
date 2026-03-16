# qiao Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a new Go CLI project named `qiao` that translates text with Google Cloud Translation Advanced v3 by default and supports future provider expansion through a registry-based abstraction.

**Architecture:** Create a Go module in the repository root with a thin CLI layer, a provider-neutral core translation contract, config loading for shared defaults and provider-specific settings, and a provider registry that initially wires only the Google implementation. Keep the first version small: single-text translation, stdin support, text/JSON output, and a `providers` subcommand.

**Tech Stack:** Go, `cobra` for CLI, `cloud.google.com/go/translate/apiv3` for Google Translation, `gopkg.in/yaml.v3` for config, standard `testing` package.

---

## Execution Note

This run is explicitly approved to execute in the current workspace and current branch without creating a git worktree.

Treat the usual `using-git-worktrees` requirement as intentionally waived for this execution only.

### Task 1: Bootstrap the project skeleton

**Files:**
- Create: `go.mod`
- Create: `cmd/qiao/main.go`
- Create: `internal/cli/root.go`
- Create: `README.md`

**Step 1: Create the project directories**

Run: `mkdir -p cmd/qiao internal/cli`
Expected: directories are created with no output

**Step 2: Initialize the Go module**

Run: `go mod init qiao`
Expected: `go.mod` is created

**Step 3: Add a minimal root command**

Implement a `cobra` root command in `internal/cli/root.go` and call it from `cmd/qiao/main.go`.

**Step 4: Add a placeholder README**

Document the project name, purpose, and planned Google provider requirement in `README.md`.

**Step 5: Verify the binary boots**

Run: `go run ./cmd/qiao --help`
Expected: help output shows the `qiao` command without translation functionality yet

**Step 6: Commit**

```bash
git add go.mod cmd/qiao/main.go internal/cli/root.go README.md
git commit -m "chore: bootstrap qiao cli project"
```

### Task 2: Define provider-neutral core types

**Files:**
- Create: `internal/core/types.go`
- Create: `internal/core/types_test.go`

**Step 1: Write the failing test**

Add tests in `internal/core/types_test.go` that validate zero-value-safe request/response helpers if helper methods are added, or at minimum validate JSON field names for the response payload type.

**Step 2: Run test to verify it fails**

Run: `go test ./internal/core`
Expected: FAIL because the package or types do not exist yet

**Step 3: Write minimal implementation**

Define:

- `TranslateRequest`
- `TranslateResponse`
- `Translator`

Include fields for text, source language, target language, provider, translation, detected source language, and optional metadata.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/core`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/core/types.go internal/core/types_test.go
git commit -m "feat: add provider-neutral translation contracts"
```

### Task 3: Add configuration loading

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the failing test**

Add tests for:

- loading defaults from YAML
- missing config file returning an empty config plus no fatal error
- provider-specific config lookup for `google`

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config`
Expected: FAIL because config loading is not implemented

**Step 3: Write minimal implementation**

Implement:

- config structs
- default config path resolution
- YAML parsing
- helper methods to fetch provider config by name

Keep the API small and deterministic.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/config`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add qiao config loading"
```

### Task 4: Add a provider registry

**Files:**
- Create: `internal/providers/registry/registry.go`
- Create: `internal/providers/registry/registry_test.go`

**Step 1: Write the failing test**

Add tests for:

- registering a provider factory
- resolving a known provider
- returning a clear error for an unknown provider
- listing providers in stable order

**Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/registry`
Expected: FAIL because the registry package does not exist

**Step 3: Write minimal implementation**

Implement a registry that maps provider names to factory functions. The factory should receive config and return a `core.Translator`.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/registry`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/providers/registry/registry.go internal/providers/registry/registry_test.go
git commit -m "feat: add provider registry"
```

### Task 5: Implement the Google provider

**Files:**
- Create: `internal/providers/google/provider.go`
- Create: `internal/providers/google/provider_test.go`

**Step 1: Write the failing test**

Write tests around request construction and config validation without making real network calls. Cover:

- missing `project_id`
- default `location` fallback to `global`
- successful conversion from `TranslateRequest` to Google API request inputs

**Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/google`
Expected: FAIL because the provider does not exist

**Step 3: Write minimal implementation**

Implement:

- Google provider config parsing
- provider constructor with validation
- translation method that calls Cloud Translation Advanced v3
- translation response mapping back into `core.TranslateResponse`

Hide the Google SDK client behind a small interface so the package stays testable.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/google`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/providers/google/provider.go internal/providers/google/provider_test.go
git commit -m "feat: add google translation provider"
```

### Task 6: Implement input resolution and output formatting

**Files:**
- Create: `internal/cli/translate.go`
- Create: `internal/cli/translate_test.go`

**Step 1: Write the failing test**

Cover:

- positional argument input
- stdin input when no positional text is present
- positional input winning over stdin
- empty input returning a user-facing error
- `--json` producing valid structured output

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestTranslate`
Expected: FAIL because translation command behavior is not implemented

**Step 3: Write minimal implementation**

Add the main translation command with flags:

- `--from`, `-f`
- `--to`, `-t`
- `--provider`, `-p`
- `--json`

Implement input resolution, provider selection, translation call, and plain-text or JSON output.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestTranslate`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/translate.go internal/cli/translate_test.go
git commit -m "feat: add translate command"
```

### Task 7: Add the `providers` command

**Files:**
- Modify: `internal/cli/root.go`
- Create: `internal/cli/providers.go`
- Create: `internal/cli/providers_test.go`

**Step 1: Write the failing test**

Add tests that verify:

- the command lists registered providers
- output stays stable and machine-readable enough for shell use

**Step 2: Run test to verify it fails**

Run: `go test ./internal/cli -run TestProviders`
Expected: FAIL because the subcommand does not exist

**Step 3: Write minimal implementation**

Add `qiao providers` and wire it to the registry list output.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/cli -run TestProviders`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cli/root.go internal/cli/providers.go internal/cli/providers_test.go
git commit -m "feat: add providers command"
```

### Task 8: Wire application defaults and registration

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`
- Modify: `internal/cli/root.go`

**Step 1: Write the failing test**

Add tests that verify:

- default provider resolves to `google`
- default source resolves to `auto`
- default target resolves to `zh`
- registry and config cooperate correctly for command execution

**Step 2: Run test to verify it fails**

Run: `go test ./internal/app`
Expected: FAIL because the app wiring layer does not exist

**Step 3: Write minimal implementation**

Add a small wiring layer that:

- loads config
- registers built-in providers
- exposes runtime defaults to the CLI

Keep this layer thin so future providers can be added with minimal touch points.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/app`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go internal/cli/root.go
git commit -m "feat: wire config defaults and provider registration"
```

### Task 9: Document installation and Google auth flow

**Files:**
- Modify: `README.md`

**Step 1: Write the failing documentation check**

Review the README and list missing items:

- install command
- usage examples
- config file example
- Google credential setup
- supported provider list

**Step 2: Update the README**

Document:

- `go install` or local build usage
- translation examples
- stdin examples
- `providers` command
- config example
- Google ADC and `GOOGLE_APPLICATION_CREDENTIALS`
- note that only Google is implemented in v1

**Step 3: Verify the README against the running CLI**

Run: `go run ./cmd/qiao --help`
Expected: command help matches the documented flags and subcommands

**Step 4: Commit**

```bash
git add README.md
git commit -m "docs: add qiao usage and configuration guide"
```

### Task 10: Run full verification

**Files:**
- No file changes required unless fixes are found

**Step 1: Run unit tests**

Run: `go test ./...`
Expected: PASS

**Step 2: Run smoke tests for CLI behavior**

Run: `go run ./cmd/qiao "How are you?"`
Expected: either a successful translation if Google credentials are configured, or a clear authentication/configuration error

**Step 3: Run stdin smoke test**

Run: `printf 'How are you?' | go run ./cmd/qiao`
Expected: either a successful translation if Google credentials are configured, or the same clear authentication/configuration error path

**Step 4: Fix any issues found**

If tests or smoke checks fail unexpectedly, add the minimal code and test changes required before continuing.

**Step 5: Commit**

```bash
git add README.md go.mod go.sum cmd internal
git commit -m "test: verify qiao v1 workflow"
```
