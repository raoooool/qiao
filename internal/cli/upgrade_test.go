package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/raoooool/qiao/internal/update"
)

func TestUpgradeCommandReportsSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer

	cmd := newRootCommand(defaultTestTranslateDeps(), ConfigDependencies{}, InitDependencies{}, UpgradeDependencies{
		Stdout: &stdout,
		Stderr: &stderr,
		Upgrade: func(ctx context.Context, version string) (update.UpgradeResult, error) {
			if version != "" {
				t.Fatalf("expected empty version, got %q", version)
			}
			return update.UpgradeResult{Version: "v1.2.0", Updated: true}, nil
		},
	})
	cmd.SetArgs([]string{"upgrade"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := stdout.String(); got != "Upgraded qiao to v1.2.0\n" {
		t.Fatalf("unexpected stdout %q", got)
	}
}

func TestUpgradeCommandReportsAlreadyCurrent(t *testing.T) {
	var stdout bytes.Buffer

	cmd := newRootCommand(defaultTestTranslateDeps(), ConfigDependencies{}, InitDependencies{}, UpgradeDependencies{
		Stdout: &stdout,
		Stderr: &bytes.Buffer{},
		Upgrade: func(ctx context.Context, version string) (update.UpgradeResult, error) {
			return update.UpgradeResult{Version: "v1.2.0", Updated: false}, nil
		},
	})
	cmd.SetArgs([]string{"upgrade"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if got := stdout.String(); got != "qiao is already up to date (v1.2.0)\n" {
		t.Fatalf("unexpected stdout %q", got)
	}
}

func TestUpgradeCommandPassesExplicitVersion(t *testing.T) {
	cmd := newRootCommand(defaultTestTranslateDeps(), ConfigDependencies{}, InitDependencies{}, UpgradeDependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Upgrade: func(ctx context.Context, version string) (update.UpgradeResult, error) {
			if version != "v1.2.0" {
				t.Fatalf("expected explicit version, got %q", version)
			}
			return update.UpgradeResult{Version: version, Updated: true}, nil
		},
	})
	cmd.SetArgs([]string{"upgrade", "--version", "v1.2.0"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestUpgradeCommandReturnsErrors(t *testing.T) {
	cmd := newRootCommand(defaultTestTranslateDeps(), ConfigDependencies{}, InitDependencies{}, UpgradeDependencies{
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		Upgrade: func(ctx context.Context, version string) (update.UpgradeResult, error) {
			return update.UpgradeResult{}, errors.New("boom")
		},
	})
	cmd.SetArgs([]string{"upgrade"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error %v", err)
	}
}
