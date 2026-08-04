package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/resetnak/cooldeck/internal/config"
)

func newConfigCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and validate configuration",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the effective configuration path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := opts.configPath
			if path == "" {
				var err error
				path, err = config.DefaultPath()
				if err != nil {
					return err
				}
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), path)
			return err
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate the configuration file",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load(opts.configPath)
			if err != nil && !errors.Is(err, config.ErrUnknownKeys) {
				return err
			}
			if err != nil {
				if _, writeErr := fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err); writeErr != nil {
					return writeErr
				}
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"valid: %s (%d instances)\n",
				cfg.Path(),
				len(cfg.Instances),
			)
			return err
		},
	})
	return cmd
}
