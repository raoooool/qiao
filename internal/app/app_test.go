package app

import (
	"os"
	"path/filepath"
	"testing"

	"qiao/internal/config"
)

func TestNewAppliesBuiltInDefaults(t *testing.T) {
	runtime := New(config.Config{})

	if got := runtime.DefaultProvider(); got != "codex" {
		t.Fatalf("expected default provider codex, got %q", got)
	}

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
