package cli

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/raoooool/qiao/internal/config"
)

type ConfigDependencies struct {
	Stdout     io.Writer
	Stderr     io.Writer
	ConfigPath string
}

func configureConfigCommand(root *cobra.Command, deps ConfigDependencies) {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long: `Manage configuration stored in ~/.config/qiao/config.yaml.

Top-level keys:
  default_provider    Translation provider used when --provider is omitted (codex, claude, tencent)
  default_source      Default source language (default: auto)
  default_target      Default target language (default: zh)

Provider keys use the format "providers.<name>.<field>":
  providers.codex.model       Model for codex provider
  providers.codex.binary      Path to codex binary (default: codex)
  providers.claude.model      Model for claude provider
  providers.claude.binary     Path to claude binary (default: claude)
  providers.tencent.secret_id     Tencent Cloud API SecretId
  providers.tencent.secret_key    Tencent Cloud API SecretKey
  providers.tencent.region        Tencent Cloud API region (default: ap-guangzhou)`,
	}

	configCmd.AddCommand(newConfigSetCommand(deps))
	configCmd.AddCommand(newConfigGetCommand(deps))
	configCmd.AddCommand(newConfigListCommand(deps))
	configCmd.AddCommand(newConfigDeleteCommand(deps))

	root.AddCommand(configCmd)
}

func newConfigSetCommand(deps ConfigDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			fileExisted := true
			if _, err := os.Stat(deps.ConfigPath); os.IsNotExist(err) {
				fileExisted = false
			}

			cfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				return err
			}

			if err := cfg.Set(key, value); err != nil {
				return err
			}

			if err := cfg.Save(deps.ConfigPath); err != nil {
				return err
			}

			if !fileExisted {
				fmt.Fprintf(deps.Stderr, "Created config file: %s\n", deps.ConfigPath)
			}

			return nil
		},
	}
}

func newConfigGetCommand(deps ConfigDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				return err
			}

			val, err := cfg.Get(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintln(deps.Stdout, val)
			return nil
		},
	}
}

func newConfigListCommand(deps ConfigDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all configuration values",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				return err
			}

			entries := cfg.List()
			keys := make([]string, 0, len(entries))
			for k := range entries {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				fmt.Fprintf(deps.Stdout, "%s=%s\n", k, entries[k])
			}

			return nil
		},
	}
}

func newConfigDeleteCommand(deps ConfigDependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <key>",
		Short: "Delete a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(deps.ConfigPath)
			if err != nil {
				return err
			}

			if err := cfg.Delete(args[0]); err != nil {
				return err
			}

			return cfg.Save(deps.ConfigPath)
		},
	}
}
