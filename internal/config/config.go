// Package config loads, validates and writes the cooldeck TOML configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// envPrefix prefixes every environment variable the application reads.
const envPrefix = "COOLDECK_"

// SchemaVersion is the current config file format. A file carrying a newer
// version is rejected rather than silently misread.
const SchemaVersion = 1

// MinRefreshInterval protects the shared Coolify API rate limit. Coolify
// applies a global limit per instance, so aggressive polling degrades the web
// dashboard for everyone on that server.
const MinRefreshInterval = 3 * time.Second

// Defaults used when a field is absent from the file.
const (
	DefaultRefreshInterval    = 10 * time.Second
	DefaultLogRefreshInterval = 2 * time.Second
	DefaultLogLines           = 300
)

// TokenSource identifies where an instance token is read from.
type TokenSource string

const (
	// TokenSourceKeyring stores the token in the OS keychain. Recommended.
	TokenSourceKeyring TokenSource = "keyring"
	// TokenSourceCommand executes an external command that prints the token.
	TokenSourceCommand TokenSource = "command"
	// TokenSourceEnv reads the token from a named environment variable.
	TokenSourceEnv TokenSource = "env"
	// TokenSourcePlaintext stores the token in the config file. Opt-in only.
	TokenSourcePlaintext TokenSource = "plaintext"
)

// Theme selects the colour scheme.
type Theme string

// Supported theme values.
const (
	ThemeAuto  Theme = "auto"
	ThemeDark  Theme = "dark"
	ThemeLight Theme = "light"

	// Named themes – curated palettes from popular open-source colour schemes.
	ThemeDracula    Theme = "dracula"
	ThemeCatppuccin Theme = "catppuccin"
	ThemeNord       Theme = "nord"
	ThemeGruvbox    Theme = "gruvbox"
	ThemeTokyoNight Theme = "tokyo-night"
)

// Tristate models the "auto" | "on" | "off" settings used for capabilities
// that can be detected but also forced by the user.
type Tristate string

// Supported tristate values.
const (
	TristateAuto Tristate = "auto"
	TristateOn   Tristate = "on"
	TristateOff  Tristate = "off"
)

// Enabled resolves a tristate against a detected value.
func (t Tristate) Enabled(detected bool) bool {
	switch t {
	case TristateOn:
		return true
	case TristateOff:
		return false
	default:
		return detected
	}
}

// Config is the whole configuration file.
type Config struct {
	Version                   int                 `toml:"version"`
	DefaultInstance           string              `toml:"default_instance"`
	RefreshInterval           Duration            `toml:"refresh_interval"`
	LogRefreshInterval        Duration            `toml:"log_refresh_interval"`
	LogLines                  int                 `toml:"log_lines"`
	Theme                     Theme               `toml:"theme"`
	ConfirmDestructiveActions bool                `toml:"confirm_destructive_actions"`
	ConfirmDeploy             bool                `toml:"confirm_deploy"`
	UI                        UI                  `toml:"ui"`
	Instances                 map[string]Instance `toml:"instances"`

	// path is where the config was loaded from; not serialised.
	path string
}

// UI holds presentation preferences.
type UI struct {
	ShowHeader  bool     `toml:"show_header"`
	ShowFooter  bool     `toml:"show_footer"`
	NerdFont    Tristate `toml:"nerd_font"`
	CompactMode Tristate `toml:"compact_mode"`
	Mouse       bool     `toml:"mouse"`
}

// Instance is a single Coolify deployment cooldeck can talk to.
type Instance struct {
	// ID is the config table key. Populated on load, not serialised.
	ID string `toml:"-"`

	Name         string      `toml:"name"`
	URL          string      `toml:"url"`
	TokenSource  TokenSource `toml:"token_source"`
	TokenKey     string      `toml:"token_key,omitempty"`
	TokenCommand []string    `toml:"token_command,omitempty"`
	TokenEnv     string      `toml:"token_env,omitempty"`
	// Token is only read when TokenSource is plaintext.
	Token string `toml:"token,omitempty"`
	// Insecure disables TLS certificate verification. Opt-in, for self-signed
	// certificates on a trusted network only.
	Insecure bool `toml:"insecure_skip_verify,omitempty"`
	// RefreshInterval overrides the global interval for this instance.
	RefreshInterval Duration `toml:"refresh_interval,omitzero"`
	// SSHHost is the user@host destination the in-app container terminal
	// connects through. The Coolify API has no exec endpoint, so the terminal
	// reaches containers over SSH instead. Empty disables the terminal.
	SSHHost string `toml:"ssh_host,omitempty"`
}

// DisplayName returns the human label for the instance, falling back to its ID.
func (i Instance) DisplayName() string {
	if i.Name != "" {
		return i.Name
	}
	return i.ID
}

// Default returns a configuration with every field populated, used both as the
// starting point for loading and as the template written by onboarding.
func Default() Config {
	return Config{
		Version:                   SchemaVersion,
		RefreshInterval:           Duration(DefaultRefreshInterval),
		LogRefreshInterval:        Duration(DefaultLogRefreshInterval),
		LogLines:                  DefaultLogLines,
		Theme:                     ThemeAuto,
		ConfirmDestructiveActions: true,
		ConfirmDeploy:             false,
		UI: UI{
			ShowHeader:  true,
			ShowFooter:  true,
			NerdFont:    TristateAuto,
			CompactMode: TristateAuto,
			Mouse:       true,
		},
		Instances: map[string]Instance{},
	}
}

// ErrNotFound is returned by Load when the config file does not exist. Callers
// use it to trigger the onboarding wizard rather than failing.
var ErrNotFound = errors.New("configuration file not found")

// Load reads and validates the config at path. An empty path resolves to the
// OS default location.
func Load(path string) (Config, error) {
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return Config{}, err
		}
		path = p
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("%w: %s", ErrNotFound, path)
		}
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	cfg.path = path

	// Propagate the table key onto each instance so it can identify itself.
	for id, inst := range cfg.Instances {
		inst.ID = id
		cfg.Instances[id] = inst
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		// Unknown keys are almost always typos. Report them, but keep the
		// config usable rather than refusing to start.
		return cfg, fmt.Errorf("%w: %s", ErrUnknownKeys, formatKeys(undecoded))
	}
	return cfg, nil
}

// ErrUnknownKeys signals that the file parsed but contained unrecognised keys.
// It is a warning: Load still returns a usable Config alongside it.
var ErrUnknownKeys = errors.New("unrecognised configuration keys")

// Path returns the file the config was loaded from.
func (c Config) Path() string { return c.path }

// Save atomically writes the config to path (or its original path if empty).
// The file is written with 0600 because it may hold a plaintext token.
func (c Config) Save(path string) error {
	if path == "" {
		path = c.path
	}
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return err
		}
		path = p
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	var b strings.Builder
	enc := toml.NewEncoder(&b)
	enc.Indent = ""
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	// Write to a temporary file in the same directory, then rename, so a
	// crash mid-write cannot truncate an existing configuration.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.toml")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup: after a successful rename the temp file is gone, so
	// a failure here carries no information the caller could act on.
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("secure temporary config: %w", err)
	}
	if _, err := tmp.WriteString(b.String()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

// Instance resolves an instance by ID, falling back to the default instance
// when id is empty. It returns a descriptive error when nothing matches.
func (c Config) Instance(id string) (Instance, error) {
	if len(c.Instances) == 0 {
		return Instance{}, errors.New("no instances configured")
	}
	if id == "" {
		id = c.DefaultInstance
	}
	if id == "" && len(c.Instances) == 1 {
		for _, inst := range c.Instances {
			return inst, nil
		}
	}
	if id == "" {
		return Instance{}, fmt.Errorf("no default_instance set; available: %s", strings.Join(c.InstanceIDs(), ", "))
	}
	inst, ok := c.Instances[id]
	if !ok {
		return Instance{}, fmt.Errorf("unknown instance %q; available: %s", id, strings.Join(c.InstanceIDs(), ", "))
	}
	return inst, nil
}

// InstanceIDs returns instance keys in stable alphabetical order, so that
// menus and error messages do not shuffle between runs.
func (c Config) InstanceIDs() []string {
	ids := make([]string, 0, len(c.Instances))
	for id := range c.Instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// OrderedInstances returns instances in the same stable order as InstanceIDs.
func (c Config) OrderedInstances() []Instance {
	ids := c.InstanceIDs()
	out := make([]Instance, 0, len(ids))
	for _, id := range ids {
		out = append(out, c.Instances[id])
	}
	return out
}

// RemoveInstance drops a configured instance from the in-memory config. It does
// not touch Coolify itself or the credential store - callers must clean those
// up separately. If the removed instance was the default, the default falls
// back to the first remaining ID (or empty when none remain).
func (c *Config) RemoveInstance(id string) error {
	if c.Instances == nil {
		return fmt.Errorf("unknown instance %q", id)
	}
	if _, ok := c.Instances[id]; !ok {
		return fmt.Errorf("unknown instance %q; available: %s", id, strings.Join(c.InstanceIDs(), ", "))
	}
	delete(c.Instances, id)
	if c.DefaultInstance == id {
		c.DefaultInstance = ""
		if ids := c.InstanceIDs(); len(ids) > 0 {
			c.DefaultInstance = ids[0]
		}
	}
	return nil
}

// EffectiveRefreshInterval returns the interval to use for an instance,
// applying the per-instance override and the rate-limit floor.
func (c Config) EffectiveRefreshInterval(inst Instance) time.Duration {
	d := time.Duration(c.RefreshInterval)
	if inst.RefreshInterval > 0 {
		d = time.Duration(inst.RefreshInterval)
	}
	if d < MinRefreshInterval {
		return MinRefreshInterval
	}
	return d
}

// EffectiveLogRefreshInterval returns the log poll interval with the same kind
// of floor as EffectiveRefreshInterval: an explicit "0s" would otherwise turn
// tea.Tick into a hot loop against the API.
func (c Config) EffectiveLogRefreshInterval() time.Duration {
	d := time.Duration(c.LogRefreshInterval)
	if d < time.Second {
		return DefaultLogRefreshInterval
	}
	return d
}

func formatKeys(keys []toml.Key) string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k.String())
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}
