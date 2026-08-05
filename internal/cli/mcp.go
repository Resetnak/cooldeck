package cli

import (
	"github.com/spf13/cobra"

	"github.com/resetnak/cooldeck/internal/mcpserver"
	"github.com/resetnak/cooldeck/internal/version"
)

func newMCPCommand(opts *options) *cobra.Command {
	var allowMutations bool

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Serve the Coolify instance to an agent over MCP (stdio)",
		Long: `Serve this Coolify instance to an MCP client over stdin/stdout.

The server is read-only by default: an agent can list applications, read runtime
and deployment logs and inspect the instance, but cannot change anything. Pass
--allow-mutations to also expose deploy, restart, start and stop. There is no
confirmation prompt behind those — whatever drives the server can run them
unattended.

Run it against --demo to try the tools offline with deterministic fake data.

Configure a client to launch it, for example:

  {"mcpServers": {"cooldeck": {"command": "cooldeck", "args": ["mcp"]}}}`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			_, service, instanceName, err := resolveService(ctx, opts)
			if err != nil {
				return err
			}
			// Stdout is the protocol transport from here on: anything else
			// written to it corrupts the session.
			return mcpserver.Run(ctx, mcpserver.Options{
				Service:        service,
				InstanceName:   instanceName,
				Version:        version.Version,
				AllowMutations: allowMutations,
			})
		},
	}

	cmd.Flags().BoolVar(&allowMutations, "allow-mutations", false,
		"also expose deploy, restart, start and stop")
	return cmd
}
