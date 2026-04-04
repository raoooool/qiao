package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func configureUpgradeCommand(root *cobra.Command, deps UpgradeDependencies) {
	var version string

	stdout := deps.Stdout
	if stdout == nil {
		stdout = io.Discard
	}

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade qiao from GitHub Releases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.Upgrade == nil {
				return errors.New("upgrade is not configured")
			}

			result, err := deps.Upgrade(cmd.Context(), version)
			if err != nil {
				return err
			}

			if result.Updated {
				_, err = fmt.Fprintf(stdout, "Upgraded qiao to %s\n", result.Version)
				return err
			}

			_, err = fmt.Fprintf(stdout, "qiao is already up to date (%s)\n", result.Version)
			return err
		},
	}

	cmd.Flags().StringVar(&version, "version", "", "upgrade to a specific version")
	root.AddCommand(cmd)
}
