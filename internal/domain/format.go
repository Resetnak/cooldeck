package domain

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// HumanizeAge renders how long ago t happened in the compact form used by
// tables: "3m ago", "2h ago", "never".
func HumanizeAge(t time.Time, now time.Time) string {
	if t.IsZero() {
		return "never"
	}
	d := now.Sub(t)
	if d < 0 {
		// Clock skew between the server and this machine; do not print a
		// negative age, which reads as a bug.
		return "just now"
	}
	if d < 5*time.Second {
		return "just now"
	}
	return HumanizeDuration(d) + " ago"
}

// HumanizeDuration renders a duration compactly with at most two units:
// "18s", "1m 24s", "2h 5m", "3d 4h".
func HumanizeDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		m := int(d.Minutes())
		s := int(d.Seconds()) - m*60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm %ds", m, s)
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) - h*60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh %dm", h, m)
	default:
		days := int(d.Hours()) / 24
		h := int(d.Hours()) - days*24
		if h == 0 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd %dh", days, h)
	}
}

// HumanizeTime renders an absolute local time for detail screens.
func HumanizeTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("2006-01-02 15:04:05")
}

// gitURLRewrites converts the SSH and git-protocol remotes Coolify stores into
// a browsable HTTPS URL. Only well-known forges are rewritten; anything else
// is left alone rather than guessed at.
var gitHosts = []string{"github.com", "gitlab.com", "bitbucket.org", "codeberg.org", "git.sr.ht"}

// WebURLForRepository converts a git remote into an https:// URL suitable for
// opening in a browser, or "" when the remote cannot be mapped safely.
//
// It deliberately refuses anything it does not recognise: the result is handed
// to the OS URL opener, so a wrong guess is worse than no link.
func WebURLForRepository(remote string) string {
	r := strings.TrimSpace(remote)
	if r == "" {
		return ""
	}
	r = strings.TrimSuffix(r, ".git")

	switch {
	case strings.HasPrefix(r, "https://"), strings.HasPrefix(r, "http://"):
		return r
	case strings.HasPrefix(r, "git@"):
		// git@github.com:owner/repo
		rest := strings.TrimPrefix(r, "git@")
		host, path, ok := strings.Cut(rest, ":")
		if !ok || !knownGitHost(host) || path == "" {
			return ""
		}
		return "https://" + host + "/" + strings.TrimPrefix(path, "/")
	case strings.HasPrefix(r, "ssh://"):
		rest := strings.TrimPrefix(r, "ssh://")
		rest = strings.TrimPrefix(rest, "git@")
		host, path, ok := strings.Cut(rest, "/")
		if !ok {
			return ""
		}
		host, _, _ = strings.Cut(host, ":") // drop an explicit port
		if !knownGitHost(host) || path == "" {
			return ""
		}
		return "https://" + host + "/" + path
	default:
		// A bare "owner/repo" is how Coolify stores GitHub App sources.
		if strings.Count(r, "/") == 1 && !strings.Contains(r, " ") && !strings.Contains(r, ":") {
			return "https://github.com/" + r
		}
		return ""
	}
}

func knownGitHost(host string) bool {
	h := strings.ToLower(host)
	for _, k := range gitHosts {
		if h == k || strings.HasSuffix(h, "."+k) {
			return true
		}
	}
	// Self-hosted forges are common with Coolify; accept a plain hostname but
	// not anything containing characters that suggest a malformed remote.
	return h != "" && !strings.ContainsAny(h, " \t/\\?#@") && strings.Contains(h, ".")
}

// DomainURL turns a Coolify FQDN entry into a browsable URL. Coolify stores
// domains either bare ("example.com") or with a scheme.
//
// The result is passed to the OS URL opener, so anything that is not
// unambiguously an http(s) address is rejected rather than coerced. In
// particular a bare "javascript:alert(1)" must not become a URL.
func DomainURL(fqdn string) string {
	f := strings.TrimSpace(fqdn)
	if f == "" {
		return ""
	}
	if !strings.HasPrefix(f, "http://") && !strings.HasPrefix(f, "https://") {
		if hasScheme(f) {
			return ""
		}
		f = "https://" + f
	}

	u, err := url.Parse(f)
	if err != nil || u.Host == "" || u.User != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	host := u.Hostname()
	if host != "localhost" && !strings.Contains(host, ".") {
		return ""
	}
	return u.String()
}

// hasScheme reports whether s starts with a "scheme:" prefix. It is stricter
// than looking for "://" because opaque URIs such as "javascript:" and
// "mailto:" have no authority component.
func hasScheme(s string) bool {
	scheme, rest, ok := strings.Cut(s, ":")
	if !ok || scheme == "" {
		return false
	}
	for _, r := range scheme {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '+' || r == '-' || r == '.') {
			return false
		}
	}
	// "example.com:8080" is a host and port, not a scheme.
	if rest != "" && rest[0] >= '0' && rest[0] <= '9' {
		return false
	}
	return true
}

// SplitFQDNs parses Coolify's comma-separated domain field.
func SplitFQDNs(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
