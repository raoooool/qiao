package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raoooool/qiao/internal/config"
	"github.com/raoooool/qiao/internal/core"
)

type InitDependencies struct {
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	ConfigPath    string
	ListProviders func() []string
	ConfigFields  func(string) []core.ConfigField
	ReadSecret    func() (string, error)
}

func configureInitCommand(root *cobra.Command, deps InitDependencies) {
	root.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Set up qiao for first use",
		Long:  "Interactive setup wizard that configures the translation provider and any required credentials.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(deps)
		},
	})
}

func runInit(deps InitDependencies) error {
	// Check if already initialized
	if _, err := os.Stat(deps.ConfigPath); err == nil {
		fmt.Fprintln(deps.Stdout, "Already initialized. Use \"qiao config\" to modify settings.")
		return nil
	}

	scanner := bufio.NewScanner(deps.Stdin)

	// List providers
	providers := deps.ListProviders()

	fmt.Fprintln(deps.Stdout, "Select a translation provider:")
	for i, p := range providers {
		fmt.Fprintf(deps.Stdout, "  [%d] %s\n", i+1, p)
	}
	fmt.Fprint(deps.Stdout, "Enter number: ")

	// Read provider selection
	selectedProvider := ""
	for {
		if !scanner.Scan() {
			return nil // EOF — exit cleanly
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			fmt.Fprint(deps.Stdout, "Invalid choice, try again: ")
			continue
		}
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(providers) {
			fmt.Fprint(deps.Stdout, "Invalid choice, try again: ")
			continue
		}
		selectedProvider = providers[num-1]
		break
	}

	// Collect required config fields
	fields := deps.ConfigFields(selectedProvider)
	providerConfig := map[string]string{}

	for _, field := range fields {
		if !field.Required {
			continue
		}

		var value string
		if field.Secret {
			fmt.Fprintf(deps.Stdout, "%s: ", field.Label)
			for {
				val, err := deps.ReadSecret()
				if err != nil {
					return nil // EOF or error — exit cleanly
				}
				value = strings.TrimSpace(val)
				if value != "" {
					break
				}
				fmt.Fprintf(deps.Stdout, "%s is required: ", field.Label)
			}
		} else {
			fmt.Fprintf(deps.Stdout, "%s: ", field.Label)
			for {
				if !scanner.Scan() {
					return nil // EOF — exit cleanly
				}
				value = strings.TrimSpace(scanner.Text())
				if value != "" {
					break
				}
				fmt.Fprintf(deps.Stdout, "%s is required: ", field.Label)
			}
		}
		providerConfig[field.Key] = value
	}

	// Build and save config
	cfg := config.Config{
		DefaultProvider: selectedProvider,
	}
	if len(providerConfig) > 0 {
		cfg.Providers = map[string]map[string]string{
			selectedProvider: providerConfig,
		}
	}

	if err := cfg.Save(deps.ConfigPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(deps.Stdout, "Configuration saved to %s\n", deps.ConfigPath)
	return nil
}
