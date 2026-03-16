# qiao Design

## Goal

Build a Go command-line translation tool named `qiao` that translates text from English to Chinese by default through Google Cloud Translation Advanced v3, while keeping the architecture open for future providers such as OpenAI, DeepL, or other AI-backed translation services.

## Product Positioning

`qiao` is a unified translation CLI, not a thin wrapper around a single vendor API.

The first release ships with a Google provider, but the command surface and internal architecture are designed so new providers can be added without changing the user-facing workflow.

## CLI Shape

Primary usage:

```bash
qiao "How are you?"
echo "How are you?" | qiao
```

Extended usage:

```bash
qiao -f en -t zh "How are you?"
qiao -p google "How are you?"
qiao --json "How are you?"
qiao providers
```

Behavior:

- Default source language: `auto`
- Default target language: `zh`
- Default provider: `google`
- Positional text input and stdin input are both supported
- Positional text wins over stdin when both are present
- Default output is plain translated text to `stdout`
- `--json` returns structured output for scripts and automation
- Errors are written to `stderr` with non-zero exit codes

## Architecture

The project should separate CLI parsing, configuration, provider selection, and provider implementations.

Recommended project layout:

```text
cmd/qiao/
internal/cli/
internal/config/
internal/core/
internal/providers/google/
internal/providers/registry/
README.md
go.mod
```

Core abstraction:

```go
type Translator interface {
    Name() string
    Translate(ctx context.Context, req TranslateRequest) (*TranslateResponse, error)
}
```

`TranslateRequest` and `TranslateResponse` should stay provider-neutral so AI-backed providers can reuse the same CLI and output contracts later.

## Configuration

Use a single user config file with provider-specific sections:

```text
~/.config/qiao/config.yaml
```

Suggested structure:

```yaml
default_provider: google
default_source: auto
default_target: zh

providers:
  google:
    project_id: your-gcp-project-id
    location: global
    credentials_file: /path/to/service-account.json

  openai:
    api_key_env: OPENAI_API_KEY
    model: gpt-4.1-mini

  deepl:
    api_key_env: DEEPL_API_KEY
```

Priority order:

1. CLI flags
2. Environment variables
3. Config file defaults

Each provider owns its own credentials and extra settings. Shared defaults stay at the top level.

## Provider Strategy

Google Cloud Translation Advanced v3 is the first provider.

Authentication should use Application Default Credentials or an explicit credentials file. The tool should not be designed around API keys because future providers will need different credential models anyway.

A registry layer should map provider names such as `google` and `openai` to concrete implementations.

## Output Model

Default output:

```text
你好吗？
```

JSON output:

```json
{
  "provider": "google",
  "source_language": "en",
  "target_language": "zh",
  "text": "How are you?",
  "translation": "你好吗？"
}
```

The response model should leave room for optional metadata, detected source language, and provider/model details without forcing them into the plain-text path.

## Error Handling

Expected errors should be explicit and actionable:

- Missing text input
- Unsupported provider
- Missing provider configuration
- Google authentication failure
- Invalid source or target language value

Normal output must remain clean so `qiao` works well in shell pipelines.

## Scope For V1

Included:

- `qiao "text"`
- stdin input
- `-f/--from`
- `-t/--to`
- `-p/--provider`
- `--json`
- `qiao providers`
- Google provider implementation
- Provider abstraction for future extension

Deferred:

- Batch file translation
- Interactive TUI
- Translation history or caching
- Multi-provider fallback logic
- Config editing commands such as `qiao config init`

## Notes

This design is saved in the current repository root at `/Users/panjunwen/Documents/Workspaces/qiao`. The implementation plan should treat this directory itself as the project root.
