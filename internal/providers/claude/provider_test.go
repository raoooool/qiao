package claude

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/raoooool/qiao/internal/config"
	"github.com/raoooool/qiao/internal/core"
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
	if provider.binary != "claude" {
		t.Fatalf("expected default binary claude, got %q", provider.binary)
	}
}

func TestNewReadsConfig(t *testing.T) {
	translator, err := New(config.Config{
		Providers: map[string]map[string]string{
			"claude": {
				"model":  "sonnet",
				"binary": "/usr/local/bin/claude",
			},
		},
	})
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	provider := translator.(*Provider)
	if provider.model != "sonnet" {
		t.Fatalf("expected model sonnet, got %q", provider.model)
	}
	if provider.binary != "/usr/local/bin/claude" {
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
	if resp.Provider != "claude" {
		t.Fatalf("expected provider claude, got %q", resp.Provider)
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
	if !strings.Contains(command, "claude") {
		t.Fatalf("expected command to contain 'claude', got %q", command)
	}
}

func TestBuildPromptAutoSource(t *testing.T) {
	prompt := buildPrompt(core.TranslateRequest{
		Text:           "Hello",
		SourceLanguage: "auto",
		TargetLanguage: "zh",
	})

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if strings.Contains(prompt, "from auto to") {
		t.Fatalf("prompt should not contain literal 'auto': %s", prompt)
	}
}
