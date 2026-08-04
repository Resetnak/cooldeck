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

var redactPatterns = []*regexp.Regexp{
	// Authorization: Bearer <token> (header dumps).
	regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=|-]{8,}`),
	// Laravel Sanctum personal access tokens, which is what Coolify issues.
	regexp.MustCompile(`\b\d+\|[A-Za-z0-9]{20,}\b`),
	// token=... / api_key=... in query strings and key/value dumps.
	regexp.MustCompile(`(?i)\b(token|api_key|apikey|secret|password)([=:]\s?)[^\s&"']+`),
}

// Redact removes credential-shaped substrings from arbitrary text. It is
// applied to every log message, every string attribute and to error text
// before it reaches a file or the screen.
func Redact(s string) string {
	if s == "" {
		return s
	}
	for _, re := range redactPatterns {
		s = re.ReplaceAllString(s, "${1}${2}"+Mask)
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
