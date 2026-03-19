# qiao

`qiao` is a Go command-line translation tool with a provider-oriented architecture. It ships with Codex CLI and Claude Code as LLM-based translation providers.

## Install

Build and run locally:

```bash
go run ./cmd/qiao --help
```

Install the binary into your Go bin directory:

```bash
go install ./cmd/qiao
```

## Usage

Translate positional text:

```bash
qiao "How are you?"
```

Translate stdin:

```bash
printf 'How are you?' | qiao
```

Override languages or provider:

```bash
qiao -f en -t zh "How are you?"
qiao -p claude "How are you?"
```

Return structured output:

```bash
qiao --json "How are you?"
```

List available providers:

```bash
qiao providers
```

## Configuration

`qiao` reads configuration from:

```text
~/.config/qiao/config.yaml
```

Example:

```yaml
default_provider: codex
default_source: auto
default_target: zh

providers:
  codex:
    model: o3
  claude:
    model: sonnet
```

CLI flags override config defaults for provider, source language, and target language.

## Supported Providers

- `codex` (default) — uses [Codex CLI](https://github.com/openai/codex) via `codex exec`
- `claude` — uses [Claude Code](https://claude.ai/code) via `claude -p`

Both providers support an optional `model` config field and a `binary` field to override the CLI path.
