package logging

import (
	"log/slog"
	"regexp"
	"strings"
)

// Mask is the placeholder substituted for every redacted value.
const Mask = "[REDACTED]"

// sensitiveKeys are attribute names whose value is never logged, regardless of
// its content. Matching is case-insensitive and substring-based so that
// "app_token" or "X-Auth-Password" are covered too.
var sensitiveKeys = []string{
	"authorization", "token", "secret", "password", "passwd",
	"credential", "api_key", "apikey", "private_key", "cookie",
}

// redactPattern is a match paired with the replacement that keeps the
// surrounding context readable - the point of redaction is to leave a message
// that still explains itself once the secret is gone.
type redactPattern struct {
	re   *regexp.Regexp
	with string
}

var redactPatterns = []redactPattern{
	// Authorization: Bearer <token> (header dumps).
	{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=|-]{8,}`), "${1}" + Mask},
	// Laravel Sanctum personal access tokens, which is what Coolify issues.
	{regexp.MustCompile(`\b\d+\|[A-Za-z0-9]{20,}\b`), Mask},
	// JSON Web Tokens, whatever key they were carried under.
	{regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_=-]*`), Mask},
	// Basic auth embedded in a URL, e.g. a git remote in a commit message.
	// Quotes and commas end the match, or the pattern would bridge from a
	// host:port through a JSON delimiter to an unrelated e-mail address.
	{regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^\s/@:"',]+:[^\s/@"',]+@`), "${1}" + Mask + "@"},
	// TOKEN=... / AWS_SECRET_ACCESS_KEY: ... / "db_password": "..." in query
	// strings, env dumps, JSON bodies and key/value prose. The surrounding
	// [\w-]* is what makes prefixed and suffixed names match; a bare \b would
	// stop at the underscore. Dots are deliberately not part of a name, or
	// "credentials.go:42" in a stack trace would lose its line number. The
	// optional quote in group 2 is what lets the value in a JSON pair be
	// reached - without it the match dies on the quote and only oddly-shaped
	// values get caught.
	{regexp.MustCompile(
		`(?i)([\w-]*(?:token|api[_-]?key|secret|password|passwd|credential|private[_-]?key)[\w-]*)("?\s*[=:]\s*"?)[^\s&"',\\]+`),
		"${1}${2}" + Mask},
}

// Redact removes credential-shaped substrings from arbitrary text. It is
// applied to every log message, every string attribute and to error text
// before it reaches a file or the screen.
func Redact(s string) string {
	if s == "" {
		return s
	}
	for _, p := range redactPatterns {
		s = p.re.ReplaceAllString(s, p.with)
	}
	return s
}

// IsSensitiveKey reports whether an attribute name identifies a secret.
func IsSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(k, s) {
			return true
		}
	}
	return false
}

// redactAttr is the slog.HandlerOptions.ReplaceAttr hook. Sensitive keys are
// masked wholesale; every other string or error value is passed through Redact
// so a token embedded in a URL or message body cannot leak.
func redactAttr(_ []string, a slog.Attr) slog.Attr {
	if IsSensitiveKey(a.Key) {
		return slog.String(a.Key, Mask)
	}
	switch v := a.Value.Any().(type) {
	case string:
		return slog.String(a.Key, Redact(v))
	case error:
		if v == nil {
			return a
		}
		return slog.String(a.Key, Redact(v.Error()))
	}
	return a
}
