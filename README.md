# qiao

`qiao` is a Go command-line translation tool with a provider-oriented architecture. It ships with three translation providers: Codex CLI, Claude Code, and Tencent Cloud Machine Translation API.

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

Show executed command and elapsed time:

```bash
qiao -v "How are you?"
```

List available providers:

```bash
qiao providers
```

Initialize provider configuration:

```bash
qiao init
```

### CLI Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--from` | `-f` | Source language | `auto` |
| `--to` | `-t` | Target language | `zh` |
| `--provider` | `-p` | Translation provider | *(configured by `qiao init`)* |
| `--json` | | Output structured JSON | `false` |
| `--verbose` | `-v` | Show executed command and elapsed time | `false` |

## Configuration

`qiao` reads configuration from:

```text
~/.config/qiao/config.yaml
```

### Managing Config via CLI

```bash
qiao config set <key> <value>    # Set a value
qiao config get <key>            # Get a value
qiao config list                 # List all values
qiao config delete <key>         # Delete a value
```

### Config Keys

#### Top-level keys

| Key | Description | Default |
|-----|-------------|---------|
| `default_provider` | Translation provider used when `--provider` is omitted | Example: `codex` |
| `default_source` | Default source language | `auto` |
| `default_target` | Default target language | `zh` |

#### Provider-specific keys

Provider config uses the format `providers.<name>.<field>`.

**codex** (`providers.codex.*`)

| Key | Description | Default |
|-----|-------------|---------|
| `providers.codex.model` | Model to use for translation | *(codex default)* |
| `providers.codex.binary` | Path to the codex CLI binary | `codex` |

**claude** (`providers.claude.*`)

| Key | Description | Default |
|-----|-------------|---------|
| `providers.claude.model` | Model to use for translation | *(claude default)* |
| `providers.claude.binary` | Path to the claude CLI binary | `claude` |

**tencent** (`providers.tencent.*`)

| Key | Description | Default |
|-----|-------------|---------|
| `providers.tencent.secret_id` | Tencent Cloud API SecretId | `$TENCENTCLOUD_SECRET_ID` |
| `providers.tencent.secret_key` | Tencent Cloud API SecretKey | `$TENCENTCLOUD_SECRET_KEY` |
| `providers.tencent.region` | Tencent Cloud API region | `ap-guangzhou` (or `$TENCENTCLOUD_REGION`) |

### Example config file

```yaml
default_provider: codex
default_source: auto
default_target: zh

providers:
  codex:
    model: o3
  claude:
    model: sonnet
  tencent:
    secret_id: AKIDxxxxxxxx
    secret_key: xxxxxxxx
    region: ap-guangzhou
```

CLI flags override config file values for provider, source language, and target language.

## Supported Providers

| Provider | Description | Requirements |
|----------|-------------|-------------|
| `codex` | Uses [Codex CLI](https://github.com/openai/codex) via `codex exec` | `codex` binary in PATH |
| `claude` | Uses [Claude Code](https://claude.ai/code) via `claude -p` | `claude` binary in PATH |
| `tencent` | Uses [Tencent Cloud Machine Translation API](https://cloud.tencent.com/product/tmt) | API credentials (env vars or config) |

## Environment Variables

| Variable | Description | Used by |
|----------|-------------|---------|
| `TENCENTCLOUD_SECRET_ID` | Tencent Cloud API SecretId | tencent provider |
| `TENCENTCLOUD_SECRET_KEY` | Tencent Cloud API SecretKey | tencent provider |
| `TENCENTCLOUD_REGION` | Tencent Cloud API region | tencent provider |
