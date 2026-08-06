// Package platform holds the small amount of OS integration cooldeck needs.
package platform

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
)

// OpenURL opens a URL in the user's default browser.
//
// Only http and https are accepted, and the URL is passed as a separate
// argument vector element, never interpolated into a shell string. Both
// matter: the URL originates from API data, so it must be treated as
// untrusted input.
func OpenURL(rawURL string) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("not a valid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("refusing to open a %q URL; only http and https are allowed", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("refusing to open a URL without a host")
	}
	// A leading dash would be read as a flag by the opener command.
	clean := u.String()
	if strings.HasPrefix(clean, "-") {
		return fmt.Errorf("refusing to open a URL starting with '-'")
	}

	name, args := openerCommand(clean)
	if name == "" {
		return fmt.Errorf("no way to open a browser on %s", runtime.GOOS)
	}
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	// Reap the opener once it exits; Start without Wait leaves a zombie per
	// opened link for the lifetime of the session.
	go func() { _ = cmd.Wait() }()
	return nil
}

func openerCommand(target string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{target}
	case "windows":
		// rundll32 avoids cmd.exe entirely, so no shell parsing is involved.
		return "rundll32", []string{"url.dll,FileProtocolHandler", target}
	default:
		return "xdg-open", []string{target}
	}
}
