package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"qiao/internal/core"
)

type fakeTranslator struct {
	response *core.TranslateResponse
	requests []core.TranslateRequest
}

func (f *fakeTranslator) Name() string {
	return "fake"
}

func (f *fakeTranslator) Translate(_ context.Context, req core.TranslateRequest) (*core.TranslateResponse, error) {
	f.requests = append(f.requests, req)
	if f.response != nil {
		return f.response, nil
	}

	return &core.TranslateResponse{
		Provider:       req.Provider,
		SourceLanguage: req.SourceLanguage,
		TargetLanguage: req.TargetLanguage,
		Text:           req.Text,
		Translation:    "translated:" + req.Text,
	}, nil
}

func TestTranslateUsesPositionalInput(t *testing.T) {
	translator := &fakeTranslator{}
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader("ignored stdin"),
		Stdout:          &stdout,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})
	cmd.SetArgs([]string{"How are you?"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	if len(translator.requests) != 1 {
		t.Fatalf("expected one translate request, got %d", len(translator.requests))
	}

	if got := translator.requests[0].Text; got != "How are you?" {
		t.Fatalf("expected positional input, got %q", got)
	}

	if got := stdout.String(); got != "translated:How are you?\n" {
		t.Fatalf("unexpected stdout %q", got)
	}
}

func TestTranslateUsesStdinWhenPositionalInputMissing(t *testing.T) {
	translator := &fakeTranslator{}
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader("How are you?"),
		Stdout:          &stdout,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	if len(translator.requests) != 1 {
		t.Fatalf("expected one translate request, got %d", len(translator.requests))
	}

	if got := translator.requests[0].Text; got != "How are you?" {
		t.Fatalf("expected stdin input, got %q", got)
	}
}

func TestTranslatePositionalInputWinsOverStdin(t *testing.T) {
	translator := &fakeTranslator{}
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader("stdin text"),
		Stdout:          &stdout,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})
	cmd.SetArgs([]string{"positional text"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	if got := translator.requests[0].Text; got != "positional text" {
		t.Fatalf("expected positional input to win, got %q", got)
	}
}

func TestTranslateRequiresInput(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader("   "),
		Stdout:          &stdout,
		ResolveProvider: fixedProviderResolver(&fakeTranslator{}),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing input error")
	}

	want := "missing text input"
	if got := err.Error(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestTranslateJSONOutput(t *testing.T) {
	translator := &fakeTranslator{
		response: &core.TranslateResponse{
			Provider:       "google",
			SourceLanguage: "en",
			TargetLanguage: "zh",
			Text:           "How are you?",
			Translation:    "你好吗？",
		},
	}
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})
	cmd.SetArgs([]string{"--json", "How are you?"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode JSON output: %v", err)
	}

	if got["provider"] != "google" {
		t.Fatalf("expected provider google, got %#v", got["provider"])
	}

	if got["translation"] != "你好吗？" {
		t.Fatalf("expected translation in JSON output, got %#v", got["translation"])
	}
}

func fixedProviderResolver(translator core.Translator) func(string) (core.Translator, error) {
	return func(string) (core.Translator, error) {
		if translator == nil {
			return nil, errors.New("translator is not configured")
		}
		return translator, nil
	}
}
