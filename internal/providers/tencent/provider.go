package tencent

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

func mapLanguage(input string) string {
	if code, ok := languageMap[strings.ToLower(input)]; ok {
		return code
	}
	return input
}
