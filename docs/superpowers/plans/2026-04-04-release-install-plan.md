# Release Install Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish `qiao` through GitHub Releases and provide a native `install.sh` path that downloads and installs release binaries safely.

**Architecture:** Update the module path to the GitHub repository import path, then use GoReleaser to build cross-platform archives and checksums on version tags. A GitHub Actions workflow will run tests and publish releases. An `install.sh` script will resolve the latest or requested version, download the correct archive and checksum from GitHub Releases, verify integrity, and install the `qiao` binary into a user-writable directory.

**Tech Stack:** Go, GitHub Actions, GoReleaser, POSIX shell, curl, tar, sha256 utilities

---

### Task 1: Fix Module Path

**Files:**
- Modify: `go.mod`
- Modify: `cmd/qiao/main.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `internal/cli/config.go`
- Modify: `internal/cli/init.go`
- Modify: `internal/cli/init_test.go`
- Modify: `internal/cli/root.go`
- Modify: `internal/cli/translate.go`
- Modify: `internal/cli/translate_test.go`
- Modify: `internal/providers/claude/provider.go`
- Modify: `internal/providers/claude/provider_test.go`
- Modify: `internal/providers/codex/provider.go`
- Modify: `internal/providers/codex/provider_test.go`
- Modify: `internal/providers/registry/registry.go`
- Modify: `internal/providers/registry/registry_test.go`
- Modify: `internal/providers/tencent/provider.go`
- Modify: `internal/providers/tencent/provider_test.go`

- [ ] Update the module path in `go.mod` to `github.com/raoooool/qiao`.
- [ ] Replace all production and test imports that start with `qiao/` with `github.com/raoooool/qiao/`.
- [ ] Run `go test ./...` and confirm the import path update did not break the build.

### Task 2: Add Release Configuration

**Files:**
- Create: `.goreleaser.yml`

- [ ] Add a GoReleaser config that builds `qiao` from `./cmd/qiao` for Linux, macOS, and Windows on `amd64` and `arm64`.
- [ ] Configure archive names so `install.sh` can deterministically map OS and architecture to release assets.
- [ ] Configure checksum generation for all release artifacts.

### Task 3: Add Release Workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] Add a tag-triggered GitHub Actions workflow for tags matching `v*`.
- [ ] Run `go test ./...` before releasing.
- [ ] Install GoReleaser and publish GitHub Release assets using the repository `GITHUB_TOKEN`.

### Task 4: Add Native Installer

**Files:**
- Create: `install.sh`

- [ ] Add a shell installer that detects OS and CPU architecture, defaults to the latest release, and also accepts `VERSION=vX.Y.Z`.
- [ ] Download the matching archive and checksum from `https://github.com/raoooool/qiao/releases/download/<tag>/...`.
- [ ] Verify the archive checksum before extraction.
- [ ] Install the `qiao` binary into `${INSTALL_DIR:-$HOME/.local/bin}` and print PATH guidance when needed.

### Task 5: Document Release and Install Flow

**Files:**
- Modify: `README.md`

- [ ] Add install instructions for `go install`, `curl | bash`, and manual GitHub Release download.
- [ ] Add maintainer release steps: tag creation and push.
- [ ] Keep the existing English and Chinese README structure readable after the additions.

### Task 6: Verify End-to-End Configuration

**Files:**
- Verify: `go.mod`
- Verify: `.goreleaser.yml`
- Verify: `.github/workflows/release.yml`
- Verify: `install.sh`
- Verify: `README.md`

- [ ] Run `go test ./...`.
- [ ] Run `goreleaser check` if available; otherwise validate the YAML visually and note the limitation.
- [ ] Run `VERSION=v0.0.0 ./install.sh` expecting a clean failure message when the release does not exist.
- [ ] Review the git diff to confirm only the intended release/install changes were made.
