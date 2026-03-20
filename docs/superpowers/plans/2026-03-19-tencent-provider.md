# Tencent Translation Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Tencent Cloud Machine Translation API provider to qiao using only Go standard library.

**Architecture:** Single-file provider at `internal/providers/tencent/provider.go` implementing `core.Translator`. Uses `httpClient` function type for testability (same pattern as `commandRunner` in codex/claude). TC3-HMAC-SHA256 signing implemented inline with `crypto/hmac` and `crypto/sha256`.

**Tech Stack:** Go standard library only (`net/http`, `crypto/hmac`, `crypto/sha256`, `encoding/json`, `encoding/hex`)

**Spec:** `docs/superpowers/specs/2026-03-19-tencent-provider-design.md`

---

### Task 1: Language mapping

**Files:**

- Create: `internal/providers/tencent/provider.go`
- Test: `internal/providers/tencent/provider_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/providers/tencent/provider_test.go`:

```go
package tencent

import "testing"

func TestMapLanguageKnownNames(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"chinese", "zh"},
		{"english", "en"},
		{"japanese", "ja"},
		{"korean", "ko"},
		{"french", "fr"},
		{"spanish", "es"},
		{"german", "de"},
	}
	for _, tt := range tests {
		if got := mapLanguage(tt.input); got != tt.want {
			t.Errorf("mapLanguage(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapLanguagePassthrough(t *testing.T) {
	// Unrecognized values pass through as-is
	if got := mapLanguage("zh"); got != "zh" {
		t.Errorf("mapLanguage(%q) = %q, want %q", "zh", got, "zh")
	}
	if got := mapLanguage("auto"); got != "auto" {
		t.Errorf("mapLanguage(%q) = %q, want %q", "auto", got, "auto")
	}
	if got := mapLanguage("xx-custom"); got != "xx-custom" {
		t.Errorf("mapLanguage(%q) = %q, want %q", "xx-custom", got, "xx-custom")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/tencent/ -run TestMapLanguage -v`
Expected: FAIL — `mapLanguage` not defined

- [ ] **Step 3: Write minimal implementation**

Create `internal/providers/tencent/provider.go` with the package declaration, imports, and `mapLanguage`:

```go
package tencent

import "strings"

var languageMap = map[string]string{
	"chinese":    "zh",
	"english":    "en",
	"japanese":   "ja",
	"korean":     "ko",
	"french":     "fr",
	"spanish":    "es",
	"italian":    "it",
	"german":     "de",
	"turkish":    "tr",
	"russian":    "ru",
	"portuguese": "pt",
	"vietnamese": "vi",
	"indonesian": "id",
	"thai":       "th",
	"malay":      "ms",
	"arabic":     "ar",
	"hindi":      "hi",
}

func mapLanguage(input string) string {
	if code, ok := languageMap[strings.ToLower(input)]; ok {
		return code
	}
	return input
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/tencent/ -run TestMapLanguage -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/providers/tencent/provider.go internal/providers/tencent/provider_test.go
git commit -m "feat(tencent): add language mapping"
```

---

### Task 2: Factory function (New) with credential loading

**Files:**

- Modify: `internal/providers/tencent/provider.go`
- Modify: `internal/providers/tencent/provider_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `provider_test.go`:

```go
import (
	"qiao/internal/config"
)

func TestNewMissingCredentialsReturnsError(t *testing.T) {
	// Clear env vars to ensure they don't interfere
	t.Setenv("TENCENTCLOUD_SECRET_ID", "")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "")

	_, err := New(config.Config{})
	if err == nil {
		t.Fatal("expected error when credentials are missing")
	}
}

func TestNewReadsFromConfig(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "")

	translator, err := New(config.Config{
		Providers: map[string]map[string]string{
			"tencent": {
				"secret_id":  "cfg-id",
				"secret_key": "cfg-key",
				"region":     "ap-shanghai",
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	p := translator.(*Provider)
	if p.secretID != "cfg-id" {
		t.Errorf("secretID = %q, want %q", p.secretID, "cfg-id")
	}
	if p.secretKey != "cfg-key" {
		t.Errorf("secretKey = %q, want %q", p.secretKey, "cfg-key")
	}
	if p.region != "ap-shanghai" {
		t.Errorf("region = %q, want %q", p.region, "ap-shanghai")
	}
}

func TestNewEnvVarsOverrideConfig(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "env-id")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "env-key")
	t.Setenv("TENCENTCLOUD_REGION", "ap-beijing")

	translator, err := New(config.Config{
		Providers: map[string]map[string]string{
			"tencent": {
				"secret_id":  "cfg-id",
				"secret_key": "cfg-key",
				"region":     "ap-shanghai",
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	p := translator.(*Provider)
	if p.secretID != "env-id" {
		t.Errorf("secretID = %q, want %q", p.secretID, "env-id")
	}
	if p.secretKey != "env-key" {
		t.Errorf("secretKey = %q, want %q", p.secretKey, "env-key")
	}
	if p.region != "ap-beijing" {
		t.Errorf("region = %q, want %q", p.region, "ap-beijing")
	}
}

func TestNewDefaultRegion(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "id")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "key")
	t.Setenv("TENCENTCLOUD_REGION", "")

	translator, err := New(config.Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	p := translator.(*Provider)
	if p.region != "ap-guangzhou" {
		t.Errorf("region = %q, want %q", p.region, "ap-guangzhou")
	}
}

func TestName(t *testing.T) {
	t.Setenv("TENCENTCLOUD_SECRET_ID", "id")
	t.Setenv("TENCENTCLOUD_SECRET_KEY", "key")

	translator, err := New(config.Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	if got := translator.Name(); got != "tencent" {
		t.Errorf("Name() = %q, want %q", got, "tencent")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/providers/tencent/ -run "TestNew|TestName" -v`
Expected: FAIL — `New`, `Provider` not defined

- [ ] **Step 3: Write minimal implementation**

Add to `provider.go` (update the import block and add after `mapLanguage`):

```go
import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"qiao/internal/config"
	"qiao/internal/core"
)

type httpClient func(req *http.Request) (*http.Response, error)

type Provider struct {
	secretID  string
	secretKey string
	region    string
	client    httpClient
}

func New(cfg config.Config) (core.Translator, error) {
	providerConfig, _ := cfg.ProviderConfig("tencent")

	secretID := os.Getenv("TENCENTCLOUD_SECRET_ID")
	if secretID == "" {
		secretID = providerConfig["secret_id"]
	}

	secretKey := os.Getenv("TENCENTCLOUD_SECRET_KEY")
	if secretKey == "" {
		secretKey = providerConfig["secret_key"]
	}

	if secretID == "" || secretKey == "" {
		return nil, fmt.Errorf("tencent provider requires secret_id and secret_key (set TENCENTCLOUD_SECRET_ID/TENCENTCLOUD_SECRET_KEY env vars or configure in providers.tencent)")
	}

	region := os.Getenv("TENCENTCLOUD_REGION")
	if region == "" {
		region = providerConfig["region"]
	}
	if region == "" {
		region = "ap-guangzhou"
	}

	httpCli := &http.Client{Timeout: 30 * time.Second}

	return &Provider{
		secretID:  secretID,
		secretKey: secretKey,
		region:    region,
		client:    httpCli.Do,
	}, nil
}

func (p *Provider) Name() string {
	return "tencent"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/providers/tencent/ -run "TestNew|TestName|TestMapLanguage" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/providers/tencent/provider.go internal/providers/tencent/provider_test.go
git commit -m "feat(tencent): add factory function with credential loading"
```

---

### Task 3: TC3-HMAC-SHA256 signing

**Files:**

- Modify: `internal/providers/tencent/provider.go`
- Modify: `internal/providers/tencent/provider_test.go`

- [ ] **Step 1: Write the failing test**

Append to `provider_test.go`:

```go
func TestSignDeterministic(t *testing.T) {
	p := &Provider{
		secretID:  "xxx",
		secretKey: "xxx",
		region:    "ap-guangzhou",
	}

	payload := `{"SourceText":"hello","Source":"en","Target":"zh","ProjectId":0}`
	timestamp := int64(1551113065)

	sig1 := p.sign(payload, timestamp)
	sig2 := p.sign(payload, timestamp)

	if sig1 != sig2 {
		t.Errorf("sign is not deterministic: %q != %q", sig1, sig2)
	}
	if sig1 == "" {
		t.Error("sign returned empty string")
	}

	// Verify it contains the expected algorithm prefix
	if !strings.Contains(sig1, "TC3-HMAC-SHA256") {
		t.Errorf("sign missing algorithm prefix: %q", sig1)
	}
	if !strings.Contains(sig1, "xxx") {
		t.Errorf("sign missing secret ID: %q", sig1)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/providers/tencent/ -run TestSignDeterministic -v`
Expected: FAIL — `sign` method not defined

- [ ] **Step 3: Write minimal implementation**

Add to `provider.go` (update imports to include `crypto/hmac`, `crypto/sha256`, `encoding/hex`, `time`):

```go
import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"qiao/internal/config"
	"qiao/internal/core"
)

func (p *Provider) sign(payload string, timestamp int64) string {
	service := "tmt"
	host := "tmt.tencentcloudapi.com"
	algorithm := "TC3-HMAC-SHA256"

	t := time.Unix(timestamp, 0).UTC()
	date := t.Format("2006-01-02")

	// Step 1: CanonicalRequest
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := "content-type:application/json\nhost:" + host + "\n"
	signedHeaders := "content-type;host"
	hashedPayload := sha256Hex(payload)

	canonicalRequest := strings.Join([]string{
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedPayload,
	}, "\n")

	// Step 2: StringToSign
	credentialScope := date + "/" + service + "/tc3_request"
	stringToSign := strings.Join([]string{
		algorithm,
		fmt.Sprintf("%d", timestamp),
		credentialScope,
		sha256Hex(canonicalRequest),
	}, "\n")

	// Step 3: Signing key
	secretDate := hmacSHA256([]byte("TC3"+p.secretKey), []byte(date))
	secretService := hmacSHA256(secretDate, []byte(service))
	secretSigning := hmacSHA256(secretService, []byte("tc3_request"))

	// Step 4: Signature
	signature := hex.EncodeToString(hmacSHA256(secretSigning, []byte(stringToSign)))

	// Step 5: Authorization
	return fmt.Sprintf(
		"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, p.secretID, credentialScope, signedHeaders, signature,
	)
}

func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/providers/tencent/ -run TestSignDeterministic -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/providers/tencent/provider.go internal/providers/tencent/provider_test.go
git commit -m "feat(tencent): add TC3-HMAC-SHA256 signing"
```

---

### Task 4: Translate method

**Files:**

- Modify: `internal/providers/tencent/provider.go`
- Modify: `internal/providers/tencent/provider_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `provider_test.go`:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"qiao/internal/core"
)

func mockHTTPClient(statusCode int, body string) httpClient {
	return func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	}
}

func TestTranslateSuccess(t *testing.T) {
	p := &Provider{
		secretID:  "test-id",
		secretKey: "test-key",
		region:    "ap-guangzhou",
		client: mockHTTPClient(200, `{
			"Response": {
				"TargetText": "你好",
				"Source": "en",
				"Target": "zh",
				"RequestId": "req-123"
			}
		}`),
	}

	resp, err := p.Translate(context.Background(), core.TranslateRequest{
		Text:           "hello",
		SourceLanguage: "english",
		TargetLanguage: "chinese",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if resp.Translation != "你好" {
		t.Errorf("translation = %q, want %q", resp.Translation, "你好")
	}
	if resp.Provider != "tencent" {
		t.Errorf("provider = %q, want %q", resp.Provider, "tencent")
	}
	if resp.DetectedSourceLanguage != "en" {
		t.Errorf("detected source = %q, want %q", resp.DetectedSourceLanguage, "en")
	}
	if resp.Metadata["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want %q", resp.Metadata["request_id"], "req-123")
	}
}

func TestTranslateAPIError(t *testing.T) {
	p := &Provider{
		secretID:  "test-id",
		secretKey: "test-key",
		region:    "ap-guangzhou",
		client: mockHTTPClient(200, `{
			"Response": {
				"Error": {
					"Code": "UnsupportedOperation.UnsupportedLanguage",
					"Message": "language pair not supported"
				},
				"RequestId": "req-456"
			}
		}`),
	}

	_, err := p.Translate(context.Background(), core.TranslateRequest{
		Text:           "hello",
		SourceLanguage: "en",
		TargetLanguage: "xx",
	})
	if err == nil {
		t.Fatal("expected error for API error response")
	}
	if !strings.Contains(err.Error(), "UnsupportedOperation.UnsupportedLanguage") {
		t.Errorf("error should contain error code: %v", err)
	}
}

func TestTranslateHTTPError(t *testing.T) {
	p := &Provider{
		secretID:  "test-id",
		secretKey: "test-key",
		region:    "ap-guangzhou",
		client:    mockHTTPClient(403, "Forbidden"),
	}

	_, err := p.Translate(context.Background(), core.TranslateRequest{
		Text:           "hello",
		SourceLanguage: "en",
		TargetLanguage: "zh",
	})
	if err == nil {
		t.Fatal("expected error for HTTP 403")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestTranslateRequestBody(t *testing.T) {
	var capturedBody []byte
	p := &Provider{
		secretID:  "test-id",
		secretKey: "test-key",
		region:    "ap-guangzhou",
		client: func(req *http.Request) (*http.Response, error) {
			capturedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 200,
				Body: io.NopCloser(bytes.NewBufferString(`{
					"Response": {"TargetText": "ok", "Source": "en", "Target": "zh", "RequestId": "r1"}
				}`)),
			}, nil
		},
	}

	_, err := p.Translate(context.Background(), core.TranslateRequest{
		Text:           "hello",
		SourceLanguage: "english",
		TargetLanguage: "chinese",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(capturedBody, &body); err != nil {
		t.Fatalf("unmarshal request body: %v", err)
	}
	if body["Source"] != "en" {
		t.Errorf("Source = %v, want %q (language mapping should apply)", body["Source"], "en")
	}
	if body["Target"] != "zh" {
		t.Errorf("Target = %v, want %q (language mapping should apply)", body["Target"], "zh")
	}
	if body["SourceText"] != "hello" {
		t.Errorf("SourceText = %v, want %q", body["SourceText"], "hello")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/providers/tencent/ -run "TestTranslate" -v`
Expected: FAIL — `Translate` method not defined

- [ ] **Step 3: Write minimal implementation**

Add to `provider.go` (update imports to include `bytes`, `context`, `encoding/json`, `io`):

```go
import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"qiao/internal/config"
	"qiao/internal/core"
)

type apiRequest struct {
	SourceText string `json:"SourceText"`
	Source     string `json:"Source"`
	Target     string `json:"Target"`
	ProjectId  int    `json:"ProjectId"`
}

type apiResponse struct {
	Response struct {
		TargetText string `json:"TargetText"`
		Source     string `json:"Source"`
		Target     string `json:"Target"`
		RequestId  string `json:"RequestId"`
		Error      *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"Response"`
}

func (p *Provider) Translate(ctx context.Context, req core.TranslateRequest) (*core.TranslateResponse, error) {
	source := mapLanguage(req.SourceLanguage)
	target := mapLanguage(req.TargetLanguage)

	body := apiRequest{
		SourceText: req.Text,
		Source:     source,
		Target:     target,
		ProjectId:  0,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	timestamp := time.Now().Unix()
	authorization := p.sign(string(payload), timestamp)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://tmt.tencentcloudapi.com", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", authorization)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Host", "tmt.tencentcloudapi.com")
	httpReq.Header.Set("X-TC-Action", "TextTranslate")
	httpReq.Header.Set("X-TC-Version", "2018-03-21")
	httpReq.Header.Set("X-TC-Timestamp", fmt.Sprintf("%d", timestamp))
	httpReq.Header.Set("X-TC-Region", p.region)

	resp, err := p.client(httpReq)
	if err != nil {
		return nil, fmt.Errorf("tencent api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tencent api returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Response.Error != nil {
		return nil, fmt.Errorf("tencent api error [%s]: %s", apiResp.Response.Error.Code, apiResp.Response.Error.Message)
	}

	return &core.TranslateResponse{
		Provider:               p.Name(),
		SourceLanguage:         req.SourceLanguage,
		TargetLanguage:         req.TargetLanguage,
		Text:                   req.Text,
		Translation:            apiResp.Response.TargetText,
		DetectedSourceLanguage: apiResp.Response.Source,
		Metadata: map[string]any{
			"request_id": apiResp.Response.RequestId,
		},
	}, nil
}
```

- [ ] **Step 4: Run all tests to verify they pass**

Run: `go test ./internal/providers/tencent/ -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/providers/tencent/provider.go internal/providers/tencent/provider_test.go
git commit -m "feat(tencent): add Translate method with HTTP request and response handling"
```

---

### Task 5: Register provider and update app test

**Files:**

- Modify: `internal/app/app.go:6-8` (add import)
- Modify: `internal/app/app.go:37-38` (add registration)
- Modify: `internal/app/app_test.go:52` (update provider count assertion)

- [ ] **Step 1: Update the app test to expect 3 providers**

In `internal/app/app_test.go`, change line 52 from:

```go
if got := runtime.ListProviders(); len(got) != 2 || got[0] != "claude" || got[1] != "codex" {
    t.Fatalf("expected [claude codex], got %v", got)
}
```

to:

```go
if got := runtime.ListProviders(); len(got) != 3 || got[0] != "claude" || got[1] != "codex" || got[2] != "tencent" {
    t.Fatalf("expected [claude codex tencent], got %v", got)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/ -run TestLoadReadsConfigFileAndRegistersProviders -v`
Expected: FAIL — only 2 providers registered

- [ ] **Step 3: Register the tencent provider**

In `internal/app/app.go`, add the import:

```go
import (
	"qiao/internal/config"
	"qiao/internal/core"
	claudeprovider "qiao/internal/providers/claude"
	codexprovider "qiao/internal/providers/codex"
	tencentprovider "qiao/internal/providers/tencent"
	"qiao/internal/providers/registry"
)
```

And add the registration line after the existing ones in `New()`:

```go
r.registry.Register("tencent", tencentprovider.New)
```

- [ ] **Step 4: Run all tests to verify they pass**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go
git commit -m "feat(tencent): register provider in app runtime"
```

---

### Task 6: Final verification

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: Build the binary**

Run: `go build ./cmd/qiao`
Expected: Successful build, no errors

- [ ] **Step 3: Verify provider is listed**

Run: `go run ./cmd/qiao --help`
Expected: CLI works, tencent should appear when listing providers (if the CLI has such a feature)
