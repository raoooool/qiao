# qiao

`qiao` is a Go command-line translation tool with a provider-oriented architecture. Version 1 ships with Google Cloud Translation Advanced v3 and keeps the CLI surface open for additional providers later.

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
qiao -p google "How are you?"
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
default_provider: google
default_source: auto
default_target: zh

providers:
  google:
    project_id: your-gcp-project-id
    location: global
    credentials_file: /path/to/service-account.json
```

CLI flags override config defaults for provider, source language, and target language.

## Google Authentication

The Google provider requires a `project_id` in the config file. Authentication uses Google Application Default Credentials.

Supported approaches:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

Or set `providers.google.credentials_file` in `~/.config/qiao/config.yaml`.

If credentials are missing or invalid, `qiao` returns a provider error on `stderr` and exits non-zero.

## Supported Providers

Currently implemented:

- `google`

Planned but not implemented in v1:

- `openai`
- `deepl`
