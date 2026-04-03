package registry

import (
	"context"
	"testing"

	"qiao/internal/config"
	"qiao/internal/core"
)

type testTranslator struct {
	name string
}

func (t testTranslator) Name() string {
	return t.name
}

func (t testTranslator) Translate(context.Context, core.TranslateRequest) (*core.TranslateResponse, error) {
	return &core.TranslateResponse{Provider: t.name}, nil
}

func TestRegisterAndResolveProvider(t *testing.T) {
	r := New()

	r.Register("google", func(config.Config) (core.Translator, error) {
		return testTranslator{name: "google"}, nil
	}, nil)

	translator, err := r.Resolve("google", config.Config{})
	if err != nil {
		t.Fatalf("resolve provider: %v", err)
	}

	if got := translator.Name(); got != "google" {
		t.Fatalf("expected translator google, got %q", got)
	}
}

func TestResolveUnknownProvider(t *testing.T) {
	r := New()

	_, err := r.Resolve("missing", config.Config{})
	if err == nil {
		t.Fatal("expected unknown provider error")
	}

	want := `unknown provider "missing"`
	if got := err.Error(); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestProvidersReturnsStableOrder(t *testing.T) {
	r := New()

	r.Register("openai", func(config.Config) (core.Translator, error) {
		return testTranslator{name: "openai"}, nil
	}, nil)
	r.Register("google", func(config.Config) (core.Translator, error) {
		return testTranslator{name: "google"}, nil
	}, nil)
	r.Register("deepl", func(config.Config) (core.Translator, error) {
		return testTranslator{name: "deepl"}, nil
	}, nil)

	got := r.Providers()
	want := []string{"deepl", "google", "openai"}

	if len(got) != len(want) {
		t.Fatalf("expected %d providers, got %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected providers %v, got %v", want, got)
		}
	}
}

func TestConfigFieldsReturnsRegisteredFields(t *testing.T) {
	r := New()

	fields := []core.ConfigField{
		{Key: "api_key", Label: "API Key", Required: true, Secret: true},
	}

	r.Register("test", func(config.Config) (core.Translator, error) {
		return testTranslator{name: "test"}, nil
	}, fields)

	got := r.ConfigFields("test")
	if len(got) != 1 {
		t.Fatalf("expected 1 config field, got %d", len(got))
	}
	if got[0].Key != "api_key" {
		t.Fatalf("expected key api_key, got %q", got[0].Key)
	}
}

func TestConfigFieldsReturnsNilForUnknownProvider(t *testing.T) {
	r := New()
	got := r.ConfigFields("missing")
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
