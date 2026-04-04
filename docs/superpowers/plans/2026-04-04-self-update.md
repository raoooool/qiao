# Self-Update Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `qiao upgrade` plus non-blocking once-per-day update notices for the main translation command.

**Architecture:** Introduce an `internal/update` package that owns version lookup, release metadata, cache storage, checksum validation, archive extraction, and executable replacement. The CLI layer wires this package into a new `upgrade` subcommand and triggers asynchronous post-translation update checks using injected dependencies.

**Tech Stack:** Go, Cobra, YAML, GitHub Releases API, GoReleaser build metadata

---

### Task 1: Add Version Plumbing

**Files:**
- Modify: `cmd/qiao/main.go`
- Modify: `internal/cli/root.go`

- [ ] Add a build-time version variable in `cmd/qiao/main.go`, defaulting to `dev`.
- [ ] Extend CLI dependencies so commands can read the running version.
- [ ] Ensure the root command can pass that version to update-related logic.

### Task 2: Build the Update Package

**Files:**
- Create: `internal/update/update.go`
- Create: `internal/update/update_test.go`

- [ ] Write failing tests for version comparison, cache freshness, latest release lookup, and upgrade failure/success paths.
- [ ] Implement update cache loading/saving for `~/.config/qiao/update.yaml`.
- [ ] Implement latest release lookup against GitHub Releases.
- [ ] Implement archive download, checksum verification, extraction, and in-place executable replacement.
- [ ] Re-run the update package tests until green.

### Task 3: Add the Upgrade Command

**Files:**
- Create: `internal/cli/upgrade.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/translate_test.go`

- [ ] Add failing CLI tests covering `qiao upgrade`, `qiao upgrade --version`, and error propagation.
- [ ] Implement the Cobra subcommand and inject the update service dependencies.
- [ ] Verify the new upgrade command tests pass.

### Task 4: Add Automatic Update Notices

**Files:**
- Modify: `internal/cli/translate.go`
- Modify: `internal/cli/translate_test.go`

- [ ] Add failing tests proving only successful translation commands trigger async checks.
- [ ] Implement non-blocking post-translation update checks for the main translation command only.
- [ ] Print the English notice to stderr only when a newer version is available.
- [ ] Re-run the translate tests until green.

### Task 5: Wire Release Metadata and Docs

**Files:**
- Modify: `.goreleaser.yml`
- Modify: `README.md`

- [ ] Add release `ldflags` so tagged builds inject the real version string.
- [ ] Document `qiao upgrade` and the automatic update notice behavior in English and Chinese README sections.

### Task 6: Verify End-to-End

**Files:**
- Verify: `cmd/qiao/main.go`
- Verify: `internal/update/update.go`
- Verify: `internal/cli/upgrade.go`
- Verify: `internal/cli/translate.go`
- Verify: `.goreleaser.yml`
- Verify: `README.md`

- [ ] Run `go test ./...`.
- [ ] Run focused tests for update and CLI packages if needed while iterating.
- [ ] Review the diff to confirm the feature stays within the approved design.
