package config

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// instanceIDPattern keeps instance keys usable as TOML table keys, keyring
// entries and CLI arguments without quoting.
var instanceIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$`)

// Validate checks the whole configuration. Every problem is reported at once
// so the user does not have to fix issues one round-trip at a time.
func (c Config) Validate() error {
	var problems []error

	if c.Version > SchemaVersion {
		problems = append(problems, fmt.Errorf(
			"version %d is newer than this build understands (max %d); upgrade cooldeck",
			c.Version, SchemaVersion))
	}

	if d := time.Duration(c.RefreshInterval); d > 0 && d < MinRefreshInterval {
		problems = append(problems, fmt.Errorf(
			"refresh_interval %s is below the %s minimum that protects the Coolify API rate limit",
			d, MinRefreshInterval))
	}
	if d := time.Duration(c.LogRefreshInterval); d > 0 && d < time.Second {
		problems = append(problems, fmt.Errorf("log_refresh_interval %s is below the 1s minimum", d))
	}
	if c.LogLines < 10 || c.LogLines > 10000 {
		problems = append(problems, fmt.Errorf("log_lines %d is outside the supported range 10-10000", c.LogLines))
	}

	switch c.Theme {
	case ThemeAuto, ThemeDark, ThemeLight,
		ThemeDracula, ThemeCatppuccin, ThemeNord, ThemeGruvbox, ThemeTokyoNight:
	default:
		problems = append(problems, fmt.Errorf("theme %q must be one of auto, dark, light, dracula, catppuccin, nord, gruvbox, tokyo-night", c.Theme))
	}
	if err := validateTristate("ui.nerd_font", c.UI.NerdFont); err != nil {
		problems = append(problems, err)
	}
	if err := validateTristate("ui.compact_mode", c.UI.CompactMode); err != nil {
		problems = append(problems, err)
	}

	for _, id := range c.InstanceIDs() {
		if !instanceIDPattern.MatchString(id) {
			problems = append(problems, fmt.Errorf(
				"instance name %q must start alphanumerically and contain only letters, digits, '-', '_' or '.'", id))
		}
		if err := c.Instances[id].Validate(); err != nil {
			problems = append(problems, fmt.Errorf("instance %q: %w", id, err))
		}
	}

	if c.DefaultInstance != "" {
		if _, ok := c.Instances[c.DefaultInstance]; !ok {
			problems = append(problems, fmt.Errorf(
				"default_instance %q is not defined; available: %s",
				c.DefaultInstance, strings.Join(c.InstanceIDs(), ", ")))
		}
	}

	return errors.Join(problems...)
}

// Validate checks a single instance definition.
func (i Instance) Validate() error {
	var problems []error

	if i.URL == "" {
		problems = append(problems, errors.New("url is required"))
	} else if _, err := NormalizeBaseURL(i.URL); err != nil {
		problems = append(problems, err)
	}

	switch i.TokenSource {
	case TokenSourceKeyring:
		// token_key is optional; it defaults to the instance ID.
	case TokenSourceCommand:
		if len(i.TokenCommand) == 0 {
			problems = append(problems, errors.New(`token_source = "command" requires token_command, e.g. ["gopass", "show", "coolify/prod"]`))
		}
	case TokenSourceEnv:
		if i.TokenEnv == "" {
			problems = append(problems, errors.New(`token_source = "env" requires token_env with the variable name`))
		}
	case TokenSourcePlaintext:
		if i.Token == "" {
			problems = append(problems, errors.New(`token_source = "plaintext" requires token`))
		}
	case "":
		problems = append(problems, errors.New("token_source is required (keyring, command, env or plaintext)"))
	default:
		problems = append(problems, fmt.Errorf("token_source %q must be one of keyring, command, env, plaintext", i.TokenSource))
	}

	// A token in the file is only honoured with the matching source, so a
	// stray value is a leak with no effect and deserves a hard error.
	if i.Token != "" && i.TokenSource != TokenSourcePlaintext {
		problems = append(problems, fmt.Errorf(
			`token is set but token_source is %q; remove the token or set token_source = "plaintext"`, i.TokenSource))
	}

	if d := time.Duration(i.RefreshInterval); d > 0 && d < MinRefreshInterval {
		problems = append(problems, fmt.Errorf("refresh_interval %s is below the %s minimum", d, MinRefreshInterval))
	}

	return errors.Join(problems...)
}

// NormalizeBaseURL validates a Coolify base URL and returns it in canonical
// form: scheme + host + path, with any trailing slash and any already-present
// /api/v1 suffix removed. The API client appends the version prefix itself, so
// users may paste either the dashboard URL or the API URL.
func NormalizeBaseURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("url is empty")
	}
	// A bare host is a common paste; assume HTTPS rather than rejecting it.
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}

	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("url %q is not parseable: %w", raw, err)
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return "", fmt.Errorf("url scheme %q is not supported, use http or https", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("url %q has no host", raw)
	}
	if u.User != nil {
		return "", errors.New("url must not embed credentials; use a token source instead")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("url %q must not contain a query string or fragment", raw)
	}

	path := strings.TrimSuffix(u.Path, "/")
	path = strings.TrimSuffix(path, "/api/v1")
	path = strings.TrimSuffix(path, "/api")
	path = strings.TrimSuffix(path, "/")

	return (&url.URL{Scheme: u.Scheme, Host: u.Host, Path: path}).String(), nil
}

func validateTristate(field string, t Tristate) error {
	switch t {
	case TristateAuto, TristateOn, TristateOff:
		return nil
	default:
		return fmt.Errorf("%s %q must be one of auto, on, off", field, t)
	}
}
