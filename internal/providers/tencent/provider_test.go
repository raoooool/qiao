package tencent

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/raoooool/qiao/internal/config"
	"github.com/raoooool/qiao/internal/core"
)

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

func TestNewMissingCredentialsReturnsError(t *testing.T) {
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
