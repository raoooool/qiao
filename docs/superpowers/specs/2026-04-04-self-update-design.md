# qiao Self-Update Design

## Goal

Add a native self-update capability to `qiao` with two user-facing behaviors:

1. `qiao upgrade` upgrades the current executable in place from GitHub Releases.
2. The main translation command checks for updates in the background at most once per day and prints a lightweight English notice when a newer version is available.

The update check must never block the normal translation flow.

## Scope

### In scope

- Add `qiao upgrade`
- Add `qiao upgrade --version vX.Y.Z`
- Replace the currently running `qiao` binary in place
- Detect latest version from GitHub Releases
- Cache update checks for 24 hours
- Trigger automatic update checks only for the main translation command
- Print a non-fatal English update notice after successful translation output
- Inject release version into the binary during tagged builds

### Out of scope

- Auto-updating on startup for `config`, `init`, or `providers` commands
- Automatic privilege escalation
- Homebrew, apt, or OS package manager integration
- Delta updates or patch-based upgrades
- Background daemon processes
- Release signing beyond the existing checksum validation

## User Experience

### Manual upgrade

Users can run:

```bash
qiao upgrade
```

Behavior:

- Resolves the latest GitHub Release version
- Compares it with the current binary version
- Downloads the matching archive and checksum for the current OS and architecture
- Validates the archive checksum
- Extracts the new binary
- Replaces the current executable path in place

If already up to date, print a short success-style message and exit.

If the current executable path is not writable, return an explicit error and do not attempt privilege escalation.

Users can also target a specific version:

```bash
qiao upgrade --version v0.2.0
```

### Automatic update notice

Only successful translation invocations may trigger an update check.

The check:

- runs asynchronously after the translation result is written
- does not affect the command exit code
- does not delay the translation output
- executes at most once every 24 hours

If a newer version is found, print a short English notice to stderr:

```text
New version available: v0.2.0. Run: qiao upgrade
```

If the current build version is `dev`, skip the automatic update check entirely.

## Architecture

Introduce a dedicated `internal/update` package so update logic does not leak into the CLI layer.

### Proposed package responsibilities

#### `internal/update`

Owns:

- current version handling
- release metadata fetching
- semantic version comparison
- update cache loading and saving
- release archive download and checksum verification
- archive extraction
- in-place executable replacement

#### `internal/cli/upgrade.go`

Owns:

- `qiao upgrade` Cobra command
- parsing `--version`
- calling the update service
- presenting user-facing upgrade messages and errors

#### `internal/cli/translate.go`

Owns:

- triggering asynchronous update checks only after successful translation command completion
- printing the lightweight update notice to stderr

#### `cmd/qiao/main.go`

Owns:

- exposing the build-time version variable, defaulting to `dev`

## Data Model

Store update metadata alongside the existing config directory.

### Update cache path

```text
~/.config/qiao/update.yaml
```

### Update cache shape

```yaml
last_checked_at: 2026-04-04T12:34:56Z
latest_version: v0.2.0
```

### Rules

- If `last_checked_at` is less than 24 hours ago, skip the network request
- If the file does not exist, treat it as no cache
- If the file is malformed, ignore it and proceed as uncached
- Cache write failures must not fail the main translation command

## Release Source

Use GitHub Releases from:

```text
https://github.com/raoooool/qiao/releases
```

Metadata source:

```text
https://api.github.com/repos/raoooool/qiao/releases/latest
```

Release asset naming must stay aligned with the current GoReleaser config:

- `qiao_linux_amd64.tar.gz`
- `qiao_linux_arm64.tar.gz`
- `qiao_darwin_amd64.tar.gz`
- `qiao_darwin_arm64.tar.gz`
- `qiao_windows_amd64.zip`
- `qiao_windows_arm64.zip`
- `qiao_checksums.txt`

## Upgrade Flow

### Manual upgrade flow

1. Determine current binary version
2. Determine target version
3. If current version is `dev`, allow explicit `--version` upgrade and latest-version upgrade, but do not try to compare `dev` semantically as a normal release version
4. Resolve OS and architecture
5. Download target archive and checksum file
6. Validate checksum
7. Extract new binary into a temporary directory
8. Determine current executable path using `os.Executable()`
9. Replace the current executable atomically where possible
10. Return success or an explicit error

### Replacement strategy

Preferred approach:

- write the extracted binary to a sibling temporary file in the destination directory
- chmod it appropriately
- rename it over the existing binary

This keeps the replacement local to the destination filesystem and avoids cross-device rename problems.

If the destination directory is not writable, fail with a clear permission error.

## Automatic Check Flow

Only run this for the main translation command after a successful translation response.

### Flow

1. Return translation output to the user
2. Start a goroutine for update checking
3. Load update cache
4. If cached within 24 hours, optionally reuse the cached `latest_version`
5. If stale or missing, fetch latest release metadata with a short timeout
6. Update cache
7. Compare latest release version with current version
8. If newer, print the update notice to stderr

### Timing constraints

- The network request must use a short timeout
- Errors are swallowed after optional internal handling
- No retry loops during normal command execution

## Version Handling

Add a build-time version variable in `cmd/qiao/main.go`, defaulting to:

```go
var version = "dev"
```

Plumb this through the CLI dependencies so update logic can compare the running version against the latest release.

GoReleaser should inject the real version during tagged builds via `ldflags`.

### Comparison rules

- Release versions are expected in `vX.Y.Z` form
- If current version is `dev`, skip automatic update checks
- If a manual upgrade target is specified, accept that version directly
- If a release tag cannot be parsed, return an explicit error rather than guessing

## Error Handling

### `qiao upgrade`

Must fail with a concrete error for:

- unsupported OS
- unsupported architecture
- release not found
- checksum entry missing
- checksum mismatch
- extracted binary missing
- destination not writable
- rename/replace failure

It must not leave behind partial files in the install directory except possibly a temporary file that is best-effort cleaned up.

### Automatic checks

Must never fail the main translation command for:

- cache read failures
- cache parse failures
- network errors
- GitHub API errors
- version parse errors
- cache write failures

These should remain non-fatal and silent to the end user.

## Testing Strategy

### `internal/update`

Use dependency injection for:

- HTTP client
- clock
- executable path lookup
- filesystem operations where practical

Test:

- latest version lookup
- version comparison
- skip when cache is fresh
- fetch when cache is stale
- skip automatic check for `dev`
- successful checksum validation
- checksum mismatch
- missing archive entry
- successful in-place replacement
- permission failure on replacement

### CLI tests

Test:

- `qiao upgrade` command wiring
- `qiao upgrade --version ...`
- successful translation triggers update check asynchronously
- non-translation commands do not trigger update checks
- update notice is printed only when a newer version exists

## Open Questions Resolved

- Notice language: English
- Automatic check scope: main translation command only
- Frequency: at most once per 24 hours
- Blocking behavior: never block the translation path
- Upgrade target path: replace the current executable in place

## Acceptance Criteria

- `qiao upgrade` upgrades the current executable from GitHub Releases
- `qiao upgrade --version vX.Y.Z` upgrades to the requested version
- Automatic checks happen only after successful translation commands
- Automatic checks do not block translation output
- Automatic checks happen at most once per day
- New-version notice is in English and printed to stderr
- `dev` builds skip automatic checks
- Tests cover the core success and failure paths
