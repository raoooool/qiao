package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"qiao/internal/app"
	"qiao/internal/config"
	"qiao/internal/core"
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
}

func NewRootCommand() *cobra.Command {
	return newRootCommand(defaultTranslateDependencies(), defaultConfigDependencies())
}

func newRootCommand(deps TranslateDependencies, cfgDeps ConfigDependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "qiao [text]",
		Short:        "Translate text from the command line",
		Long: `qiao is a provider-oriented translation CLI.

Supported providers: codex (default), claude, tencent.

Configuration file: ~/.config/qiao/config.yaml
Use "qiao config" to manage configuration.`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}

	configureTranslateCommand(cmd, deps)
	configureProvidersCommand(cmd, deps)
	configureConfigCommand(cmd, cfgDeps)

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
			DefaultProvider: "codex",
			DefaultSource:   "auto",
			DefaultTarget:   "zh",
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
	}
}
