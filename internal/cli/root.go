package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/raoooool/qiao/internal/app"
	"github.com/raoooool/qiao/internal/config"
	"github.com/raoooool/qiao/internal/core"
	"github.com/raoooool/qiao/internal/update"
)

var buildVersion = "dev"

func SetVersion(version string) {
	buildVersion = version
}

type TranslateDependencies struct {
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
	ResolveProvider func(string) (core.Translator, error)
	ListProviders   func() []string
	DefaultProvider string
	DefaultSource   string
	DefaultTarget   string
	ConfigPath      string
	FileExists      func(string) bool
	RunAsync        func(func())
	CheckForUpdate  func(io.Writer)
}

type UpgradeDependencies struct {
	Stdout  io.Writer
	Stderr  io.Writer
	Upgrade func(context.Context, string) (update.UpgradeResult, error)
}

func NewRootCommand() *cobra.Command {
	return newRootCommand(
		defaultTranslateDependencies(buildVersion),
		defaultConfigDependencies(),
		defaultInitDependencies(),
		defaultUpgradeDependencies(buildVersion),
	)
}

func newRootCommand(deps TranslateDependencies, cfgDeps ConfigDependencies, initDeps InitDependencies, upgradeDeps ...UpgradeDependencies) *cobra.Command {
	var resolvedUpgradeDeps UpgradeDependencies
	if len(upgradeDeps) > 0 {
		resolvedUpgradeDeps = upgradeDeps[0]
	}

	cmd := &cobra.Command{
		Use:   "qiao [text]",
		Short: "Translate text from the command line",
		Long: `qiao is a provider-oriented translation CLI.

Supported providers: codex, claude, tencent.

Configuration file: ~/.config/qiao/config.yaml
Use "qiao config" to manage configuration.`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}

	configureTranslateCommand(cmd, deps)
	configureProvidersCommand(cmd, deps)
	configureConfigCommand(cmd, cfgDeps)
	configureInitCommand(cmd, initDeps)
	configureUpgradeCommand(cmd, resolvedUpgradeDeps)

	return cmd
}

func defaultConfigDependencies() ConfigDependencies {
	configPath, _ := config.DefaultPath()
	return ConfigDependencies{
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		ConfigPath: configPath,
	}
}

func defaultTranslateDependencies(version string) TranslateDependencies {
	configPath, _ := config.DefaultPath()
	runtime, err := app.Load("")
	updateService := defaultUpdateService(version)
	if err != nil {
		return TranslateDependencies{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			ResolveProvider: func(string) (core.Translator, error) {
				return nil, err
			},
			ListProviders: func() []string {
				return nil
			},
			DefaultProvider: "",
			DefaultSource:   "auto",
			DefaultTarget:   "zh",
			ConfigPath:      configPath,
			FileExists: func(path string) bool {
				_, err := os.Stat(path)
				return err == nil
			},
			RunAsync: defaultRunAsync,
		}
	}

	return TranslateDependencies{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		ResolveProvider: func(name string) (core.Translator, error) {
			return runtime.ResolveProvider(name)
		},
		ListProviders: func() []string {
			return runtime.ListProviders()
		},
		DefaultProvider: runtime.DefaultProvider(),
		DefaultSource:   runtime.DefaultSource(),
		DefaultTarget:   runtime.DefaultTarget(),
		ConfigPath:      configPath,
		FileExists: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		RunAsync: defaultRunAsync,
		CheckForUpdate: func(stderr io.Writer) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			result, err := updateService.Check(ctx)
			if err != nil || !result.HasUpdate {
				return
			}
			fmt.Fprintf(stderr, "New version available: %s. Run: qiao upgrade\n", result.LatestVersion)
		},
	}
}

func defaultUpgradeDependencies(version string) UpgradeDependencies {
	service := defaultUpdateService(version)
	return UpgradeDependencies{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Upgrade: func(ctx context.Context, targetVersion string) (update.UpgradeResult, error) {
			return service.Upgrade(ctx, targetVersion)
		},
	}
}

func defaultInitDependencies() InitDependencies {
	configPath, _ := config.DefaultPath()
	runtime := app.New(config.Config{})

	return InitDependencies{
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		ConfigPath:    configPath,
		ListProviders: func() []string { return runtime.ListProviders() },
		ConfigFields:  func(name string) []core.ConfigField { return runtime.ProviderConfigFields(name) },
		ReadSecret:    defaultReadSecret,
	}
}

func defaultReadSecret() (string, error) {
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stdout) // newline after hidden input
	if err != nil {
		return "", err
	}
	return string(password), nil
}

func defaultUpdateService(version string) update.Service {
	cachePath, _ := update.DefaultCachePath()
	return update.Service{
		Version:   version,
		Repo:      "raoooool/qiao",
		CachePath: cachePath,
		Client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func defaultRunAsync(fn func()) {
	go fn()
}
