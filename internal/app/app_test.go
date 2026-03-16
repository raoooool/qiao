package app

import (
	"os"
	"path/filepath"
	"testing"

	"qiao/internal/config"
)

func TestNewAppliesBuiltInDefaults(t *testing.T) {
	runtime := New(config.Config{})

	if got := runtime.DefaultProvider(); got != "google" {
		t.Fatalf("expected default provider google, got %q", got)
	}

	if got := runtime.DefaultSource(); got != "auto" {
		t.Fatalf("expected default source auto, got %q", got)
	}

	if got := runtime.DefaultTarget(); got != "zh" {
		t.Fatalf("expected default target zh, got %q", got)
	}
}

func TestResolveProviderUsesConfigAndRegistry(t *testing.T) {
	runtime := New(config.Config{
		Providers: map[string]map[string]string{
			"google": {
				"project_id": "demo-project",
				"location":   "global",
			},
		},
	})

	translator, err := runtime.ResolveProvider("google")
	if err != nil {
		t.Fatalf("resolve provider: %v", err)
	}

	if got := translator.Name(); got != "google" {
		t.Fatalf("expected google translator, got %q", got)
	}
}

func TestLoadReadsConfigFileAndRegistersProviders(t *testing.T) {
	path := writeConfigFile(t, `
default_provider: google
default_source: auto
default_target: zh
providers:
  google:
    project_id: demo-project
`)

	runtime, err := Load(path)
	if err != nil {
		t.Fatalf("load app runtime: %v", err)
	}

	if got := runtime.ListProviders(); len(got) != 1 || got[0] != "google" {
		t.Fatalf("expected [google], got %v", got)
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
