package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestProvidersListsRegisteredProviders(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		ResolveProvider: fixedProviderResolver(&fakeTranslator{}),
		ListProviders: func() []string {
			return []string{"google", "openai"}
		},
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})
	cmd.SetArgs([]string{"providers"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute providers command: %v", err)
	}

	if got := stdout.String(); got != "google\nopenai\n" {
		t.Fatalf("unexpected providers output %q", got)
	}
}

func TestProvidersOutputIsStableForShellUse(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCommand(TranslateDependencies{
		Stdin:           strings.NewReader(""),
		Stdout:          &stdout,
		ResolveProvider: fixedProviderResolver(&fakeTranslator{}),
		ListProviders: func() []string {
			return []string{"deepl", "google"}
		},
		DefaultProvider: "google",
		DefaultSource:   "auto",
		DefaultTarget:   "zh",
	})
	cmd.SetArgs([]string{"providers"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute providers command: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	want := []string{"deepl", "google"}

	if len(lines) != len(want) {
		t.Fatalf("expected %d lines, got %d", len(want), len(lines))
	}

	for i := range want {
		if lines[i] != want[i] {
			t.Fatalf("expected output %v, got %v", want, lines)
		}
	}
}
