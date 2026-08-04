package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/resetnak/cooldeck/internal/version"
)

// FileName is the config file looked up inside the config directory.
const FileName = "config.toml"

// Dir returns the OS-appropriate configuration directory:
//
//	Linux    ~/.config/cooldeck
//	macOS    ~/Library/Application Support/cooldeck
//	Windows  %AppData%\cooldeck
//
// COOLDECK_CONFIG_DIR overrides it, which keeps tests hermetic.
func Dir() (string, error) {
	if custom := os.Getenv(envPrefix + "CONFIG_DIR"); custom != "" {
		return custom, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}
	return filepath.Join(base, version.AppName), nil
}

// DefaultPath returns the default config file path.
func DefaultPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}

// StateDir returns the directory for logs and other non-configuration state.
func StateDir() (string, error) {
	if custom := os.Getenv(envPrefix + "STATE_DIR"); custom != "" {
		return custom, nil
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate user cache directory: %w", err)
	}
	return filepath.Join(base, version.AppName), nil
}

// LogPath returns the path of the application log file.
func LogPath() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, version.AppName+".log"), nil
}
