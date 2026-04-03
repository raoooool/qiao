package app

import (
	"os"
	"path/filepath"
	"testing"

	"qiao/internal/config"
)

func TestNewDoesNotApplyBuiltInProviderFallback(t *testing.T) {
	runtime := New(config.Config{})

	if got := runtime.DefaultProvider(); got != "" {
		t.Fatalf("expected no provider fallback, got %q", got)
	}
}

func TestNewAppliesBuiltInLanguageDefaults(t *testing.T) {
	runtime := New(config.Config{})

	if got := runtime.DefaultSource(); got != "auto" {
		t.Fatalf("expected default source auto, got %q", got)
	}

	if got := runtime.DefaultTarget(); got != "zh" {
		t.Fatalf("expected default target zh, got %q", got)
	}
}

func TestResolveProviderUsesConfigAndRegistry(t *testing.T) {
	runtime := New(config.Config{})

	translator, err := runtime.ResolveProvider("claude")
	if err != nil {
		t.Fatalf("resolve provider: %v", err)
	}

	if got := translator.Name(); got != "claude" {
		t.Fatalf("expected claude translator, got %q", got)
	}
}

func TestLoadReadsConfigFileAndRegistersProviders(t *testing.T) {
	path := writeConfigFile(t, `
default_provider: claude
default_source: auto
default_target: zh
`)

	runtime, err := Load(path)
	if err != nil {
		t.Fatalf("load app runtime: %v", err)
	}

	if got := runtime.ListProviders(); len(got) != 3 || got[0] != "claude" || got[1] != "codex" || got[2] != "tencent" {
		t.Fatalf("expected [claude codex tencent], got %v", got)
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

func TestProviderConfigFieldsReturnsTencentFields(t *testing.T) {
	runtime := New(config.Config{})

	fields := runtime.ProviderConfigFields("tencent")
	if len(fields) != 2 {
		t.Fatalf("expected 2 config fields for tencent, got %d", len(fields))
	}
	if fields[0].Key != "secret_id" {
		t.Fatalf("expected first field key secret_id, got %q", fields[0].Key)
	}
	if fields[1].Key != "secret_key" {
		t.Fatalf("expected second field key secret_key, got %q", fields[1].Key)
	}
}

func TestProviderConfigFieldsReturnsEmptyForCodex(t *testing.T) {
	runtime := New(config.Config{})

	fields := runtime.ProviderConfigFields("codex")
	if len(fields) != 0 {
		t.Fatalf("expected 0 config fields for codex, got %d", len(fields))
	}
}
