package cli

import (
	"github.com/resetnak/cooldeck/internal/cli/themepicker"
	"github.com/spf13/cobra"
)

func newThemeCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "theme",
		Short: "Interactive theme selector wizard with live preview",
		Long:  `Launch an interactive terminal wizard to preview and set your CoolDeck color theme in real-time.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return themepicker.Run(opts.configPath)
		},
	}
}
