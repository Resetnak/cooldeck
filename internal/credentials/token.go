// Package credentials resolves Coolify API tokens from the OS keyring,
// environment variables or an external command, and stores them in the keyring.
package credentials

// Redacted is what a Token renders as in any formatted output.
const Redacted = "[REDACTED]"

// Token wraps a secret string. It deliberately does not expose the value
// through fmt: String, GoString and MarshalText all yield a placeholder, so a
// stray %v, %#v, log attribute or JSON encode cannot leak the credential. The
// value is only reachable through Secret, which is easy to grep for in review.
type Token struct {
	value string
}

// NewToken wraps a raw secret.
func NewToken(v string) Token { return Token{value: v} }

// Secret returns the raw token. Call it only where the credential is actually
// transmitted, never for display, logging or error text.
func (t Token) Secret() string { return t.value }

// IsZero reports whether no token was resolved.
func (t Token) IsZero() bool { return t.value == "" }

// String implements fmt.Stringer with a redacted placeholder.
func (t Token) String() string {
	if t.IsZero() {
		return "<empty>"
	}
	return Redacted
}

// GoString implements fmt.GoStringer so %#v is safe too.
func (t Token) GoString() string { return "credentials.Token(" + t.String() + ")" }

// MarshalText implements encoding.TextMarshaler so encoding/json is safe too.
func (t Token) MarshalText() ([]byte, error) { return []byte(t.String()), nil }
