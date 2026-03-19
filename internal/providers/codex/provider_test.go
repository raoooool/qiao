package codex

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"qiao/internal/config"
	"qiao/internal/core"
)

func fakeRunner(output string, err error) commandRunner {
	return func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return []byte(output), err
	}
}

func TestNewDefaultsBinary(t *testing.T) {
	translator, err := New(config.Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	if provider.binary != "codex" {
		t.Fatalf("expected default binary codex, got %q", provider.binary)
	}
}

func TestNewReadsConfig(t *testing.T) {
	translator, err := New(config.Config{
		Providers: map[string]map[string]string{
			"codex": {
				"model":  "o3",
				"binary": "/usr/local/bin/codex",
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	if provider.model != "o3" {
		t.Fatalf("expected model o3, got %q", provider.model)
	}
	if provider.binary != "/usr/local/bin/codex" {
		t.Fatalf("expected custom binary path, got %q", provider.binary)
	}
}

func TestTranslateSuccess(t *testing.T) {
	translator, err := New(config.Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	provider.runCmd = fakeRunner("你好吗？\n", nil)

	resp, err := provider.Translate(context.Background(), core.TranslateRequest{
		Text:           "How are you?",
		SourceLanguage: "en",
		TargetLanguage: "zh",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	if resp.Translation != "你好吗？" {
		t.Fatalf("expected translation, got %q", resp.Translation)
	}
	if resp.Provider != "codex" {
		t.Fatalf("expected provider codex, got %q", resp.Provider)
	}
	if resp.Text != "How are you?" {
		t.Fatalf("expected original text, got %q", resp.Text)
	}
}

func TestTranslateCommandError(t *testing.T) {
	translator, err := New(config.Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	provider.runCmd = fakeRunner("", fmt.Errorf("command failed"))

	_, err = provider.Translate(context.Background(), core.TranslateRequest{
		Text:           "Hello",
		TargetLanguage: "zh",
	})
	if err == nil {
		t.Fatal("expected error from failed command")
	}
}

func TestTranslateMetadataContainsCommand(t *testing.T) {
	translator, err := New(config.Config{})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	provider.runCmd = fakeRunner("translated\n", nil)

	resp, err := provider.Translate(context.Background(), core.TranslateRequest{
		Text:           "hello",
		SourceLanguage: "en",
		TargetLanguage: "zh",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	command, ok := resp.Metadata["command"].(string)
	if !ok || command == "" {
		t.Fatal("expected metadata to contain non-empty 'command' key")
	}
	if !strings.Contains(command, "codex") {
		t.Fatalf("expected command to contain 'codex', got %q", command)
	}
}

func TestBuildPromptAutoSource(t *testing.T) {
	prompt := buildPrompt(core.TranslateRequest{
		Text:           "Hello",
		SourceLanguage: "auto",
		TargetLanguage: "zh",
	})

	if got := prompt; got == "" {
		t.Fatal("expected non-empty prompt")
	}
	// Should not contain literal "auto" as language
	if contains(prompt, "from auto to") {
		t.Fatalf("prompt should not contain literal 'auto': %s", prompt)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
