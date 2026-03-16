package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func configureProvidersCommand(root *cobra.Command, deps TranslateDependencies) {
	root.AddCommand(&cobra.Command{
		Use:   "providers",
		Short: "List available translation providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, provider := range deps.ListProviders() {
				if _, err := fmt.Fprintln(deps.Stdout, provider); err != nil {
					return err
				}
			}

			return nil
		},
	})
}
