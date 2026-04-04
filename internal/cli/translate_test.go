package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/raoooool/qiao/internal/core"
	"github.com/raoooool/qiao/internal/update"
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
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	}, ConfigDependencies{}, InitDependencies{})
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
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	}, ConfigDependencies{}, InitDependencies{})

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
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	}, ConfigDependencies{}, InitDependencies{})
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
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(&fakeTranslator{}),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	}, ConfigDependencies{}, InitDependencies{})

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
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	}, ConfigDependencies{}, InitDependencies{})
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

func TestTranslateVerboseOutput(t *testing.T) {
	translator := &fakeTranslator{
		response: &core.TranslateResponse{
			Provider:       "fake",
			SourceLanguage: "auto",
			TargetLanguage: "zh",
			Text:           "hello",
			Translation:    "你好",
			Metadata: map[string]any{
				"command": `fake "exec" "hello"`,
			},
		},
	}
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "fake",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"-v", "hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "[qiao]") {
		t.Fatalf("expected stderr to contain [qiao] prefix, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, `fake "exec" "hello"`) {
		t.Fatalf("expected stderr to contain command, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "s)") {
		t.Fatalf("expected stderr to contain elapsed time, got %q", stderrStr)
	}

	if stdout.String() != "你好\n" {
		t.Fatalf("expected stdout to contain translation only, got %q", stdout.String())
	}
}

func TestTranslateNoVerboseByDefault(t *testing.T) {
	translator := &fakeTranslator{
		response: &core.TranslateResponse{
			Provider:    "fake",
			Text:        "hello",
			Translation: "你好",
			Metadata: map[string]any{
				"command": `fake "exec" "hello"`,
			},
		},
	}
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "fake",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute command: %v", err)
	}

	if stderr.String() != "" {
		t.Fatalf("expected no stderr output without -v, got %q", stderr.String())
	}
}

func TestTranslateVerboseOnError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:  strings.NewReader(""),
		Stdout: &stdout,
		Stderr: &stderr,
		ResolveProvider: func(string) (core.Translator, error) {
			return nil, errors.New("provider failed")
		},
		DefaultProvider: "fake",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"-v", "hello"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "[qiao]") {
		t.Fatalf("expected stderr to contain [qiao] even on error, got %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "s)") {
		t.Fatalf("expected stderr to contain elapsed time, got %q", stderrStr)
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

func TestTranslateExitsWhenNotInitialized(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ResolveProvider: fixedProviderResolver(&fakeTranslator{}),
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		ConfigPath:      "/nonexistent/path/config.yaml",
		FileExists:      func(string) bool { return false },
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"hello"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when not initialized")
	}

	if !strings.Contains(stderr.String(), "qiao init") {
		t.Fatalf("expected init hint in stderr, got %q", stderr.String())
	}
}

func TestTranslateSkipsCheckWhenProviderFlagSet(t *testing.T) {
	translator := &fakeTranslator{}
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		ConfigPath:      "/nonexistent/path/config.yaml",
		FileExists:      func(string) bool { return false },
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"--provider", "codex", "hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success with explicit --provider, got: %v", err)
	}

	if !strings.Contains(stdout.String(), "translated:hello") {
		t.Fatalf("expected translation output, got %q", stdout.String())
	}
}

func TestTranslateProceedsWhenInitialized(t *testing.T) {
	translator := &fakeTranslator{}
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		ConfigPath:      "/some/path/config.yaml",
		FileExists:      func(string) bool { return true },
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected success when initialized, got: %v", err)
	}

	if !strings.Contains(stdout.String(), "translated:hello") {
		t.Fatalf("expected translation output, got %q", stdout.String())
	}
}

func TestTranslateFailsWhenProviderIsNotConfigured(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ResolveProvider: fixedProviderResolver(&fakeTranslator{}),
		DefaultProvider: "",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		ConfigPath:      "/some/path/config.yaml",
		FileExists:      func(string) bool { return true },
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"hello"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when provider is not configured")
	}

	if !errors.Is(err, errProviderResolutionNotConfigured) {
		t.Fatalf("expected provider not configured error, got %v", err)
	}
}

func TestTranslatePrintsUpdateNoticeAfterSuccess(t *testing.T) {
	translator := &fakeTranslator{}
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		Stderr:          &stderr,
		ResolveProvider: fixedProviderResolver(translator),
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		RunAsync: func(fn func()) {
			fn()
		},
		CheckForUpdate: func(w io.Writer) {
			result := update.CheckResult{HasUpdate: true, LatestVersion: "v1.2.0"}
			if result.HasUpdate {
				_, _ = io.WriteString(w, "New version available: v1.2.0. Run: qiao upgrade\n")
			}
		},
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := stdout.String(); got != "translated:hello\n" {
		t.Fatalf("unexpected stdout %q", got)
	}
	if got := stderr.String(); !strings.Contains(got, "New version available: v1.2.0") {
		t.Fatalf("expected update notice, got %q", got)
	}
}

func TestTranslateDoesNotCheckForUpdatesOnFailure(t *testing.T) {
	var checks int

	cmd := newRootCommand(TranslateDependencies{
		Stdin:  strings.NewReader(""),
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		ResolveProvider: func(string) (core.Translator, error) {
			return nil, errors.New("provider failed")
		},
		DefaultProvider: "codex",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
		RunAsync: func(fn func()) {
			fn()
		},
		CheckForUpdate: func(io.Writer) {
			checks++
		},
	}, ConfigDependencies{}, InitDependencies{})
	cmd.SetArgs([]string{"hello"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if checks != 0 {
		t.Fatalf("expected no update checks on failure, got %d", checks)
	}
}
