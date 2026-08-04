package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/resetnak/cooldeck/internal/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build information",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version.Current().String())
			return err
		},
	}
}
