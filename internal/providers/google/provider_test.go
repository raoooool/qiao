package google

import (
	"context"
	"testing"

	"cloud.google.com/go/translate/apiv3/translatepb"
	gax "github.com/googleapis/gax-go/v2"

	"qiao/internal/config"
	"qiao/internal/core"
)

type fakeClient struct {
	req *translatepb.TranslateTextRequest
}

func (f *fakeClient) TranslateText(_ context.Context, req *translatepb.TranslateTextRequest, _ ...gax.CallOption) (*translatepb.TranslateTextResponse, error) {
	f.req = req

	return &translatepb.TranslateTextResponse{
		Translations: []*translatepb.Translation{
			{
				TranslatedText:       "你好吗？",
				DetectedLanguageCode: "en",
				Model:                "general/nmt",
			},
		},
	}, nil
}

func (f *fakeClient) Close() error {
	return nil
}

func TestNewRequiresProjectID(t *testing.T) {
	_, err := New(config.Config{
		Providers: map[string]map[string]string{
			"google": {},
		},
	})
	if err == nil {
		t.Fatal("expected missing project_id error")
	}

	want := `google provider requires "project_id"`
	if got := err.Error(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNewDefaultsLocationToGlobal(t *testing.T) {
	translator, err := New(config.Config{
		Providers: map[string]map[string]string{
			"google": {
				"project_id": "demo-project",
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	if provider.location != "global" {
		t.Fatalf("expected location global, got %q", provider.location)
	}
}

func TestTranslateBuildsGoogleRequest(t *testing.T) {
	translator, err := New(config.Config{
		Providers: map[string]map[string]string{
			"google": {
				"project_id": "demo-project",
				"location":   "us-central1",
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	client := &fakeClient{}
	provider.client = client

	resp, err := provider.Translate(context.Background(), core.TranslateRequest{
		Text:           "How are you?",
		SourceLanguage: "en",
		TargetLanguage: "zh",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	if client.req == nil {
		t.Fatal("expected translate request to be sent")
	}

	if got := client.req.GetParent(); got != "projects/demo-project/locations/us-central1" {
		t.Fatalf("expected parent path, got %q", got)
	}

	if got := client.req.GetSourceLanguageCode(); got != "en" {
		t.Fatalf("expected source language en, got %q", got)
	}

	if got := client.req.GetTargetLanguageCode(); got != "zh" {
		t.Fatalf("expected target language zh, got %q", got)
	}

	if len(client.req.GetContents()) != 1 || client.req.GetContents()[0] != "How are you?" {
		t.Fatalf("unexpected contents: %#v", client.req.GetContents())
	}

	if got := client.req.GetMimeType(); got != "text/plain" {
		t.Fatalf("expected mime type text/plain, got %q", got)
	}

	if resp.Translation != "你好吗？" {
		t.Fatalf("expected translated text, got %q", resp.Translation)
	}

	if resp.DetectedSourceLanguage != "en" {
		t.Fatalf("expected detected language en, got %q", resp.DetectedSourceLanguage)
	}
}
