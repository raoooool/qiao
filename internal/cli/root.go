package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/raoooool/qiao/internal/app"
	"github.com/raoooool/qiao/internal/config"
	"github.com/raoooool/qiao/internal/core"
)

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
}

func NewRootCommand() *cobra.Command {
	return newRootCommand(defaultTranslateDependencies(), defaultConfigDependencies(), defaultInitDependencies())
}

func newRootCommand(deps TranslateDependencies, cfgDeps ConfigDependencies, initDeps InitDependencies) *cobra.Command {
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

func defaultTranslateDependencies() TranslateDependencies {
	configPath, _ := config.DefaultPath()
	runtime, err := app.Load("")
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
