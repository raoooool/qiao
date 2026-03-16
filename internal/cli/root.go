package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"qiao/internal/app"
	"qiao/internal/core"
)

type TranslateDependencies struct {
	Stdin           io.Reader
	Stdout          io.Writer
	ResolveProvider func(string) (core.Translator, error)
	ListProviders   func() []string
	DefaultProvider string
	DefaultSource   string
	DefaultTarget   string
}

func NewRootCommand() *cobra.Command {
	return newRootCommand(defaultTranslateDependencies())
}

func newRootCommand(deps TranslateDependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "qiao [text]",
		Short:        "Translate text from the command line",
		Long:         "qiao is a provider-oriented translation CLI. Google Cloud Translation is the first planned provider.",
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
	}

	configureTranslateCommand(cmd, deps)
	configureProvidersCommand(cmd, deps)

	return cmd
}

func defaultTranslateDependencies() TranslateDependencies {
	runtime, err := app.Load("")
	if err != nil {
		return TranslateDependencies{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			ResolveProvider: func(string) (core.Translator, error) {
				return nil, err
			},
			ListProviders: func() []string {
				return nil
			},
			DefaultProvider: "google",
			DefaultSource:   "auto",
			DefaultTarget:   "zh",
		}
	}

	return TranslateDependencies{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
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
