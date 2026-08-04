package cli

import (
	"github.com/resetnak/cooldeck/internal/cli/setup"
	"github.com/spf13/cobra"
)

func newSetupCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard for first-time configuration",
		Long:  `Launch an interactive terminal wizard that guides you through creating your CoolDeck configuration file. The wizard will ask for your Coolify instance URL, API token, and display preferences.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return setup.Run()
		},
	}
}
