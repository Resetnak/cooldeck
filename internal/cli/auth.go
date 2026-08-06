package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/credentials"
)

func newAuthCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage API tokens in the OS keyring",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newAuthAddCommand(opts))
	cmd.AddCommand(newAuthDeleteCommand(opts))
	cmd.AddCommand(newAuthStatusCommand(opts))
	return cmd
}

func newAuthAddCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "add <instance>",
		Short: "Store an instance token in the OS keyring",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := loadInstance(opts.configPath, args[0])
			if err != nil {
				return err
			}
			if inst.TokenSource != config.TokenSourceKeyring {
				return fmt.Errorf("instance %q does not use token_source = %q", inst.ID, config.TokenSourceKeyring)
			}

			token, err := readToken(cmd.InOrStdin(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			if err := credentials.Store(credentials.KeyFor(inst), credentials.NewToken(token)); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "stored token for %s\n", inst.ID)
			return err
		},
	}
}

func newAuthDeleteCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <instance>",
		Short: "Delete an instance token from the OS keyring",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := loadInstance(opts.configPath, args[0])
			if err != nil {
				return err
			}
			if err := credentials.Remove(credentials.KeyFor(inst)); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "deleted token for %s\n", inst.ID)
			return err
		},
	}
}

func newAuthStatusCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "status [instance]",
		Short: "Show credential availability without revealing tokens",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(opts.configPath)
			if err != nil {
				return err
			}

			instances := cfg.OrderedInstances()
			if len(args) == 1 {
				inst, lookupErr := cfg.Instance(args[0])
				if lookupErr != nil {
					return lookupErr
				}
				instances = []config.Instance{inst}
			}

			return printCredentialStatus(cmd.Context(), cmd.OutOrStdout(), instances)
		},
	}
}

func loadInstance(path, id string) (config.Instance, error) {
	cfg, err := loadConfig(path)
	if err != nil {
		return config.Instance{}, err
	}
	return cfg.Instance(id)
}

func printCredentialStatus(ctx context.Context, out io.Writer, instances []config.Instance) error {
	missing := false
	for _, inst := range instances {
		status := credentials.Check(ctx, inst)
		state := "available"
		if !status.Available {
			state = "missing"
			missing = true
		}
		if _, err := fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", inst.ID, state, status.Origin, status.Detail); err != nil {
			return err
		}
	}
	if missing {
		return errors.New("one or more credentials are unavailable")
	}
	return nil
}

func readToken(in io.Reader, prompt io.Writer) (string, error) {
	if f, ok := in.(*os.File); ok && term.IsTerminal(f.Fd()) {
		if _, err := fmt.Fprint(prompt, "API token: "); err != nil {
			return "", err
		}
		value, err := term.ReadPassword(f.Fd())
		if _, writeErr := fmt.Fprintln(prompt); err == nil && writeErr != nil {
			return "", writeErr
		}
		if err != nil {
			return "", fmt.Errorf("read token: %w", err)
		}
		return validateToken(string(value))
	}

	value, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read token: %w", err)
	}
	return validateToken(value)
}

func validateToken(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("token is empty")
	}
	return value, nil
}
