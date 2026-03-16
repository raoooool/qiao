package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultPathUsesHomeConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}

	want := filepath.Join(home, ".config", "qiao", "config.yaml")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestLoadParsesDefaultsFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	content := []byte(`
default_provider: google
default_source: auto
default_target: zh
providers:
  google:
    project_id: demo-project
    location: global
`)

	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.DefaultProvider != "google" {
		t.Fatalf("expected default provider google, got %q", cfg.DefaultProvider)
	}

	if cfg.DefaultSource != "auto" {
		t.Fatalf("expected default source auto, got %q", cfg.DefaultSource)
	}

	if cfg.DefaultTarget != "zh" {
		t.Fatalf("expected default target zh, got %q", cfg.DefaultTarget)
	}
}

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load missing config: %v", err)
	}

	if !reflect.DeepEqual(cfg, Config{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestProviderConfigLookup(t *testing.T) {
	cfg := Config{
		Providers: map[string]map[string]string{
			"google": {
				"project_id": "demo-project",
				"location":   "global",
			},
		},
	}

	got, ok := cfg.ProviderConfig("google")
	if !ok {
		t.Fatal("expected google provider config")
	}

	want := map[string]string{
		"project_id": "demo-project",
		"location":   "global",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}
