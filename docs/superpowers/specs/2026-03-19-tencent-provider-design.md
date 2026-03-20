# Tencent Translation Provider Design

## Overview

Add a new translation provider (`tencent`) that calls the Tencent Cloud Machine Translation API (TextTranslate) via HTTP, using TC3-HMAC-SHA256 signature authentication. Implemented with Go standard library only — no external SDK dependencies.

API Reference: https://cloud.tencent.com/document/api/551/15619

## File Structure

```
internal/providers/tencent/
├── provider.go       # Provider implementation + signing + language mapping
└── provider_test.go  # Tests
```

No changes to CLI, config, or core types.

## Provider Struct

```go
type httpClient func(req *http.Request) (*http.Response, error)

type Provider struct {
    secretID  string
    secretKey string
    region    string
    client    httpClient
}
```

The `httpClient` function type follows the same testability pattern as `commandRunner` in the existing codex/claude providers — inject a mock in tests, use a dedicated `&http.Client{Timeout: 30 * time.Second}` in production (not `http.DefaultClient`, which has no timeout).

`Name() string` returns `"tencent"` (required by `core.Translator` interface).

## Configuration & Credentials

Factory function `New(cfg config.Config) (core.Translator, error)` reads credentials in priority order:

1. Environment variables: `TENCENTCLOUD_SECRET_ID`, `TENCENTCLOUD_SECRET_KEY`, `TENCENTCLOUD_REGION`
2. Config file (`~/.config/qiao/config.yaml`): `providers.tencent.secret_id`, `secret_key`, `region`
3. `region` defaults to `ap-guangzhou`
4. Missing `secret_id` or `secret_key` returns an error

Example config:

```yaml
providers:
  tencent:
    secret_id: "AKIDxxxxxxxx"
    secret_key: "xxxxxxxx"
    region: "ap-guangzhou"
```

## TC3-HMAC-SHA256 Signing

Internal function `func (p *Provider) sign(payload string, timestamp int64) string` implements the Tencent Cloud V3 signature:

1. **CanonicalRequest**: POST method, `/` URI, empty query string, `content-type: application/json` + `host: tmt.tencentcloudapi.com` headers, payload SHA256
2. **StringToSign**: algorithm (`TC3-HMAC-SHA256`), timestamp, credential scope (`date/tmt/tc3_request`), SHA256 of CanonicalRequest
3. **Signing key**: chain HMAC-SHA256: `"TC3" + SecretKey` → date → `"tmt"` → `"tc3_request"`
4. **Signature**: HMAC-SHA256 of StringToSign with the signing key
5. **Authorization header**: assembled from algorithm, credential, signed headers, and signature

All crypto from standard library: `crypto/hmac`, `crypto/sha256`.

## Translate Flow

1. Map source/target language codes via `mapLanguage()`
2. Build JSON request body: `{"SourceText": "...", "Source": "...", "Target": "...", "ProjectId": 0}`
3. Sign the payload and assemble HTTP request via `http.NewRequestWithContext(ctx, ...)` to `https://tmt.tencentcloudapi.com` — context from the `Translate` method is attached so cancellation and timeouts propagate
4. Set headers: `Authorization`, `Content-Type: application/json`, `Host`, `X-TC-Action: TextTranslate`, `X-TC-Version: 2018-03-21`, `X-TC-Timestamp`, `X-TC-Region`
5. Send via `httpClient`, `defer resp.Body.Close()`
6. Check `resp.StatusCode` — return error for non-200 responses before attempting JSON parsing
7. Parse response JSON; if `Response.Error` exists, return error with code and message
8. Return `core.TranslateResponse` with `Metadata["request_id"]` and `DetectedSourceLanguage` populated from the API's `Source` response field (useful when input source is `auto`)

## Language Mapping

`mapLanguage(input string) string` maps common English names to Tencent language codes:

| Input | Output |
|-------|--------|
| chinese | zh |
| english | en |
| japanese | ja |
| korean | ko |
| french | fr |
| spanish | es |
| italian | it |
| german | de |
| turkish | tr |
| russian | ru |
| portuguese | pt |
| vietnamese | vi |
| indonesian | id |
| thai | th |
| malay | ms |
| arabic | ar |
| hindi | hi |

Unrecognized values are passed through as-is. `auto` is passed through (supported by the API).

## Registration

In `internal/app/app.go`:

```go
import tencentprovider "qiao/internal/providers/tencent"

// In New()
r.registry.Register("tencent", tencentprovider.New)
```

## Testing

All tests use a mock `httpClient` — no real HTTP requests.

| Test Case | Description |
|-----------|-------------|
| Successful translation | Mock returns valid JSON, verify `TranslateResponse` fields |
| API error | Mock returns `Response.Error` JSON, verify error returned |
| Language mapping | Verify `"chinese"` → `"zh"`, unknown values pass through |
| Signing determinism | Fixed timestamp input, verify signature output is deterministic |
| Credential priority | Env vars override config file; missing credentials return error |
