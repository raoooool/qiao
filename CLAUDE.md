# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

qiao is a Go CLI translation tool using a provider-oriented architecture. Ships with two LLM-based providers: Codex CLI (default) and Claude Code. Built with cobra for CLI and YAML for config (`~/.config/qiao/config.yaml`).

## Commands

```bash
go run ./cmd/qiao --help      # Run locally
go test ./...                 # Run all tests
go test ./internal/config/    # Run tests for a single package
go build ./cmd/qiao           # Build binary
```

## Architecture

The codebase follows a layered design:

- **`cmd/qiao/main.go`** — Entrypoint, delegates to `cli.NewRootCommand()`
- **`internal/cli/`** — Cobra command definitions. Uses a `TranslateDependencies` struct for dependency injection, making commands testable without real providers.
- **`internal/app/`** — `Runtime` wires config + registry together and provides defaults (provider: codex, source: auto, target: zh)
- **`internal/config/`** — YAML config loading from `~/.config/qiao/config.yaml`
- **`internal/core/`** — `Translator` interface and request/response types. All providers implement `core.Translator`.
- **`internal/providers/registry/`** — Factory-based provider registry. Providers are registered as `Factory func(config.Config) (core.Translator, error)`.
- **`internal/providers/codex/`** — Codex CLI provider. Shells out to `codex exec` for translation.
- **`internal/providers/claude/`** — Claude Code provider. Shells out to `claude -p` for translation.

Both CLI-based providers use a `commandRunner` function type for testability (mock the command execution, not the CLI).

## Adding a New Provider

1. Create `internal/providers/<name>/provider.go` implementing `core.Translator`
2. Accept `config.Config` in the factory constructor
3. Register it in `app.New()` via `r.registry.Register("<name>", providerpkg.New)`
