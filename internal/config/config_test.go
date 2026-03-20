package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestParseKeyPath_TopLevel(t *testing.T) {
	kind, parts, err := parseKeyPath("default_provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != keyKindTopLevel {
		t.Fatalf("expected keyKindTopLevel, got %d", kind)
	}
	if parts[0] != "default_provider" {
		t.Fatalf("expected default_provider, got %q", parts[0])
	}
}

func TestParseKeyPath_Provider(t *testing.T) {
	kind, parts, err := parseKeyPath("providers.tencent.secret_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != keyKindProvider {
		t.Fatalf("expected keyKindProvider, got %d", kind)
	}
	if parts[0] != "tencent" || parts[1] != "secret_id" {
		t.Fatalf("expected [tencent secret_id], got %v", parts)
	}
}

func TestParseKeyPath_BareProviders(t *testing.T) {
	_, _, err := parseKeyPath("providers")
	if err == nil {
		t.Fatal("expected error for bare providers key")
	}
}

func TestParseKeyPath_UnknownTopLevel(t *testing.T) {
	_, _, err := parseKeyPath("foo")
	if err == nil {
		t.Fatal("expected error for unknown top-level key")
	}
}

func TestParseKeyPath_WrongSegmentCount(t *testing.T) {
	_, _, err := parseKeyPath("a.b.c.d")
	if err == nil {
		t.Fatal("expected error for four segments")
	}

	_, _, err = parseKeyPath("providers.tencent")
	if err == nil {
		t.Fatal("expected error for two segments")
	}
}

func TestGet_TopLevel(t *testing.T) {
	cfg := Config{DefaultProvider: "tencent"}
	val, err := cfg.Get("default_provider")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "tencent" {
		t.Fatalf("expected tencent, got %q", val)
	}
}

func TestGet_TopLevelNotSet(t *testing.T) {
	cfg := Config{}
	_, err := cfg.Get("default_provider")
	if err == nil {
		t.Fatal("expected error for unset key")
	}
}

func TestGet_Provider(t *testing.T) {
	cfg := Config{
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123"},
		},
	}
	val, err := cfg.Get("providers.tencent.secret_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "AKID123" {
		t.Fatalf("expected AKID123, got %q", val)
	}
}

func TestGet_ProviderNotFound(t *testing.T) {
	cfg := Config{Providers: map[string]map[string]string{}}
	_, err := cfg.Get("providers.tencent.secret_id")
	if err == nil {
		t.Fatal("expected error for missing provider key")
	}
}

func TestGet_InvalidKey(t *testing.T) {
	cfg := Config{}
	_, err := cfg.Get("foo")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestSet_TopLevel(t *testing.T) {
	cfg := Config{}
	if err := cfg.Set("default_provider", "tencent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProvider != "tencent" {
		t.Fatalf("expected tencent, got %q", cfg.DefaultProvider)
	}
}

func TestSet_Provider(t *testing.T) {
	cfg := Config{}
	if err := cfg.Set("providers.tencent.secret_id", "AKID123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Providers["tencent"]["secret_id"] != "AKID123" {
		t.Fatalf("expected AKID123, got %q", cfg.Providers["tencent"]["secret_id"])
	}
}

func TestSet_ProviderCreatesIntermediateMaps(t *testing.T) {
	cfg := Config{}
	if err := cfg.Set("providers.claude.model", "opus"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Providers == nil {
		t.Fatal("expected providers map to be created")
	}
	if cfg.Providers["claude"]["model"] != "opus" {
		t.Fatalf("expected opus, got %q", cfg.Providers["claude"]["model"])
	}
}

func TestSet_OverwritesExisting(t *testing.T) {
	cfg := Config{DefaultProvider: "codex"}
	if err := cfg.Set("default_provider", "tencent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProvider != "tencent" {
		t.Fatalf("expected tencent, got %q", cfg.DefaultProvider)
	}
}

func TestSet_InvalidKey(t *testing.T) {
	cfg := Config{}
	if err := cfg.Set("foo", "bar"); err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestDelete_TopLevel(t *testing.T) {
	cfg := Config{DefaultProvider: "tencent"}
	if err := cfg.Delete("default_provider"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultProvider != "" {
		t.Fatalf("expected empty, got %q", cfg.DefaultProvider)
	}
}

func TestDelete_Provider(t *testing.T) {
	cfg := Config{
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123", "region": "ap-guangzhou"},
		},
	}
	if err := cfg.Delete("providers.tencent.secret_id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cfg.Providers["tencent"]["secret_id"]; ok {
		t.Fatal("expected secret_id to be deleted")
	}
	if cfg.Providers["tencent"]["region"] != "ap-guangzhou" {
		t.Fatal("expected region to remain")
	}
}

func TestDelete_ProviderCleansUpEmptyMap(t *testing.T) {
	cfg := Config{
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123"},
		},
	}
	if err := cfg.Delete("providers.tencent.secret_id"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cfg.Providers["tencent"]; ok {
		t.Fatal("expected empty provider map to be removed")
	}
}

func TestDelete_NotFound(t *testing.T) {
	cfg := Config{}
	err := cfg.Delete("default_provider")
	if err == nil {
		t.Fatal("expected error for deleting unset key")
	}
}

func TestDelete_ProviderNotFound(t *testing.T) {
	cfg := Config{Providers: map[string]map[string]string{}}
	err := cfg.Delete("providers.tencent.secret_id")
	if err == nil {
		t.Fatal("expected error for deleting missing provider key")
	}
}

func TestList_FullConfig(t *testing.T) {
	cfg := Config{
		DefaultProvider: "tencent",
		DefaultTarget:   "en",
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123"},
		},
	}
	got := cfg.List()
	want := map[string]string{
		"default_provider":            "tencent",
		"default_target":              "en",
		"providers.tencent.secret_id": "AKID123",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestList_SkipsEmptyValues(t *testing.T) {
	cfg := Config{DefaultProvider: "codex"}
	got := cfg.List()
	if _, ok := got["default_source"]; ok {
		t.Fatal("expected empty default_source to be skipped")
	}
	if got["default_provider"] != "codex" {
		t.Fatalf("expected codex, got %q", got["default_provider"])
	}
}

func TestList_EmptyConfig(t *testing.T) {
	cfg := Config{}
	got := cfg.List()
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestSave_WritesYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{
		DefaultProvider: "tencent",
		Providers: map[string]map[string]string{
			"tencent": {"secret_id": "AKID123"},
		},
	}

	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load after save: %v", err)
	}

	if loaded.DefaultProvider != "tencent" {
		t.Fatalf("expected tencent, got %q", loaded.DefaultProvider)
	}
	if loaded.Providers["tencent"]["secret_id"] != "AKID123" {
		t.Fatalf("expected AKID123, got %q", loaded.Providers["tencent"]["secret_id"])
	}
}

func TestSave_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.yaml")

	cfg := Config{DefaultProvider: "codex"}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestSave_OmitsEmptyProviders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Config{DefaultProvider: "codex"}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if strings.Contains(string(data), "providers") {
		t.Fatalf("expected no providers key in output, got:\n%s", data)
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
