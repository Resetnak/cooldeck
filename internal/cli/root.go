// Package cli defines cooldeck's command-line interface.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/coolify"
	"github.com/resetnak/cooldeck/internal/credentials"
	"github.com/resetnak/cooldeck/internal/logging"
	"github.com/resetnak/cooldeck/internal/tui"
	"github.com/resetnak/cooldeck/internal/version"
)

type options struct {
	configPath string
	instance   string
	debug      bool
	demo       bool
	noMouse    bool
	theme      string
}

// Execute runs the CLI with explicit streams so non-interactive commands are
// straightforward to test.
func Execute(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) error {
	opts := options{}
	cmd := newRootCommand(&opts)
	cmd.SetArgs(args)
	cmd.SetIn(in)
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	return cmd.ExecuteContext(ctx)
}

func newRootCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:           version.AppName,
		Short:         version.Tagline,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runTUI(cmd.Context(), opts)
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.configPath, "config", "", "configuration file")
	flags.StringVar(&opts.instance, "instance", "", "Coolify instance ID")
	flags.BoolVar(&opts.debug, "debug", false, "enable debug logging")
	flags.BoolVar(&opts.demo, "demo", false, "run with deterministic local demo data")
	flags.BoolVar(&opts.noMouse, "no-mouse", false, "disable mouse support")
	flags.StringVar(&opts.theme, "theme", "", "theme override: auto, dark, light, dracula, catppuccin, nord, gruvbox, tokyo-night")

	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newConfigCommand(opts))
	cmd.AddCommand(newAuthCommand(opts))
	cmd.AddCommand(newSetupCommand())
	cmd.AddCommand(newThemeCommand(opts))
	return cmd
}

func runTUI(ctx context.Context, opts *options) error {
	cfg, service, instanceName, err := resolveService(ctx, opts)
	if err != nil {
		return err
	}
	if err := applyThemeOverride(&cfg, opts.theme); err != nil {
		return err
	}
	cfg.UI.Mouse = cfg.UI.Mouse && !opts.noMouse

	logPath, pathErr := config.LogPath()
	logger, logErr := logging.Setup(logging.Options{Path: logPath, Debug: opts.debug})
	if pathErr != nil || logErr != nil || logger == nil {
		// Fall back to an in-memory / discard logger so a bad log path cannot
		// take down the TUI. Setup always returns a non-nil logger today, but
		// the nil check keeps the call site honest for static analysis.
		logger, _ = logging.Setup(logging.Options{Debug: opts.debug})
	}
	if logger != nil {
		defer logger.Close()
	}

	tuiOpts := tui.Options{
		Config:       cfg,
		Service:      service,
		InstanceName: instanceName,
		Demo:         opts.demo,
		Theme:        cfg.Theme,
		Mouse:        cfg.UI.Mouse,
		ConfigPath:   cfg.Path(),
		LogPath:      logPath,
	}
	if logger != nil {
		tuiOpts.Logger = logger.Logger
		tuiOpts.RecentErrors = func() []string {
			recs := logger.Recent()
			out := make([]string, 0, len(recs))
			for _, r := range recs {
				out = append(out, r.Level.String()+": "+r.Message)
			}
			return out
		}
	}
	if !opts.demo {
		configPath := cfg.Path()
		if configPath == "" {
			configPath = opts.configPath
		}
		tuiOpts.OpenService = func(ctx context.Context, instanceID string) (app.Service, string, error) {
			return openConfiguredService(ctx, configPath, instanceID)
		}
		tuiOpts.SaveConfig = func(c config.Config) error {
			return c.Save(configPath)
		}
		tuiOpts.RemoveCredentials = func(inst config.Instance) error {
			if inst.TokenSource != config.TokenSourceKeyring {
				return nil
			}
			return credentials.Remove(credentials.KeyFor(inst))
		}
		tuiOpts.StoreCredentials = func(inst config.Instance, token string) error {
			return credentials.Store(credentials.KeyFor(inst), credentials.NewToken(token))
		}
	}
	return tui.Run(ctx, tuiOpts)
}

// openConfiguredService resolves credentials and builds a Coolify service for
// an instance ID. Used for mid-session switches from the TUI.
func openConfiguredService(ctx context.Context, configPath, instanceID string) (app.Service, string, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, "", err
	}
	instance, err := cfg.Instance(instanceID)
	if err != nil {
		return nil, "", err
	}
	token, _, err := credentials.Resolve(ctx, instance)
	if err != nil {
		return nil, "", fmt.Errorf("resolve credentials for %q: %w", instance.ID, err)
	}
	service, err := coolify.New(coolify.Options{Instance: instance, Token: token})
	if err != nil {
		return nil, "", err
	}
	return service, instance.DisplayName(), nil
}

func resolveService(ctx context.Context, opts *options) (config.Config, app.Service, string, error) {
	if opts.demo {
		cfg := config.Default()
		service := demo.New(demo.Options{Latency: 350 * time.Millisecond})
		return cfg, service, "Demo", nil
	}

	cfg, err := config.Load(opts.configPath)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			return config.Config{}, nil, "", fmt.Errorf("%w; run with --demo or create %s", err, config.FileName)
		}
		return config.Config{}, nil, "", err
	}
	instance, err := cfg.Instance(opts.instance)
	if err != nil {
		return config.Config{}, nil, "", err
	}
	token, _, err := credentials.Resolve(ctx, instance)
	if err != nil {
		return config.Config{}, nil, "", fmt.Errorf("resolve credentials for %q: %w", instance.ID, err)
	}
	service, err := coolify.New(coolify.Options{Instance: instance, Token: token})
	if err != nil {
		return config.Config{}, nil, "", err
	}
	return cfg, service, instance.DisplayName(), nil
}

func applyThemeOverride(cfg *config.Config, raw string) error {
	if raw == "" {
		return nil
	}
	theme := config.Theme(raw)
	switch theme {
	case config.ThemeAuto, config.ThemeDark, config.ThemeLight,
		config.ThemeDracula, config.ThemeCatppuccin, config.ThemeNord,
		config.ThemeGruvbox, config.ThemeTokyoNight:
		cfg.Theme = theme
		return nil
	default:
		return fmt.Errorf("invalid theme %q: use auto, dark, light, dracula, catppuccin, nord, gruvbox, or tokyo-night", raw)
	}
}
