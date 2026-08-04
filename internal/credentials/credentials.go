package credentials

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/zalando/go-keyring"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/version"
)

// Service is the keyring service name entries are filed under.
const Service = version.AppName

// EnvOverride is the environment variable that short-circuits every configured
// token source. It exists for CI, one-off shells and `COOLDECK_TOKEN=... cooldeck`.
const EnvOverride = "COOLDECK_TOKEN"

// commandTimeout bounds an external token command so a hanging password
// manager cannot freeze startup.
const commandTimeout = 30 * time.Second

// ErrNotFound means no token exists for the instance at its configured source.
var ErrNotFound = errors.New("no token found")

// Origin describes where a resolved token actually came from, which is what
// `auth status` and the diagnostics screen report.
type Origin string

// Recognised origins.
const (
	OriginEnvOverride Origin = "environment override"
	OriginKeyring     Origin = "OS keyring"
	OriginCommand     Origin = "external command"
	OriginEnv         Origin = "environment variable"
	OriginPlaintext   Origin = "config file (plaintext)"
)

// Resolve returns the token for an instance. Resolution order is:
//
//  1. the COOLDECK_TOKEN environment override,
//  2. whatever the instance's token_source specifies.
//
// The returned Origin is safe to display; the Token is not.
func Resolve(ctx context.Context, inst config.Instance) (Token, Origin, error) {
	if v := strings.TrimSpace(os.Getenv(EnvOverride)); v != "" {
		return NewToken(v), OriginEnvOverride, nil
	}

	switch inst.TokenSource {
	case config.TokenSourceKeyring:
		tok, err := fromKeyring(KeyFor(inst))
		return tok, OriginKeyring, err

	case config.TokenSourceCommand:
		tok, err := fromCommand(ctx, inst.TokenCommand)
		return tok, OriginCommand, err

	case config.TokenSourceEnv:
		v := strings.TrimSpace(os.Getenv(inst.TokenEnv))
		if v == "" {
			return Token{}, OriginEnv, fmt.Errorf("%w: environment variable %s is unset or empty", ErrNotFound, inst.TokenEnv)
		}
		return NewToken(v), OriginEnv, nil

	case config.TokenSourcePlaintext:
		v := strings.TrimSpace(inst.Token)
		if v == "" {
			return Token{}, OriginPlaintext, fmt.Errorf("%w: token is empty in the config file", ErrNotFound)
		}
		return NewToken(v), OriginPlaintext, nil

	default:
		return Token{}, "", fmt.Errorf("unsupported token_source %q", inst.TokenSource)
	}
}

// KeyFor returns the keyring account name for an instance: the explicit
// token_key when set, otherwise the instance ID.
func KeyFor(inst config.Instance) string {
	if inst.TokenKey != "" {
		return inst.TokenKey
	}
	return inst.ID
}

// Store writes a token to the OS keyring.
func Store(key string, tok Token) error {
	if key == "" {
		return errors.New("keyring key is empty")
	}
	if tok.IsZero() {
		return errors.New("refusing to store an empty token")
	}
	if err := keyring.Set(Service, key, tok.Secret()); err != nil {
		return fmt.Errorf("store token in keyring: %w", err)
	}
	return nil
}

// Remove deletes a token from the OS keyring. A missing entry is not an error,
// so deleting an instance is idempotent.
func Remove(key string) error {
	err := keyring.Delete(Service, key)
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return fmt.Errorf("delete token from keyring: %w", err)
}

// Status is a non-sensitive summary of an instance's credential state.
type Status struct {
	InstanceID string
	Source     config.TokenSource
	Origin     Origin
	// Detail names the concrete location without revealing the secret, e.g.
	// the keyring key or the environment variable name.
	Detail    string
	Available bool
	// Err explains why the token could not be resolved. It is already safe to
	// display: no resolver includes the secret in its error text.
	Err error
}

// Check resolves the token for an instance and reports whether it is
// available, without ever returning the value itself.
func Check(ctx context.Context, inst config.Instance) Status {
	st := Status{InstanceID: inst.ID, Source: inst.TokenSource, Detail: detailFor(inst)}
	tok, origin, err := Resolve(ctx, inst)
	st.Origin = origin
	if err != nil {
		st.Err = err
		return st
	}
	st.Available = !tok.IsZero()
	if !st.Available {
		st.Err = ErrNotFound
	}
	return st
}

func detailFor(inst config.Instance) string {
	switch inst.TokenSource {
	case config.TokenSourceKeyring:
		return Service + "/" + KeyFor(inst)
	case config.TokenSourceCommand:
		if len(inst.TokenCommand) == 0 {
			return ""
		}
		// Only the executable name; arguments may name a secret path.
		return inst.TokenCommand[0] + " …"
	case config.TokenSourceEnv:
		return "$" + inst.TokenEnv
	case config.TokenSourcePlaintext:
		return "config file"
	}
	return ""
}

func fromKeyring(key string) (Token, error) {
	if key == "" {
		return Token{}, errors.New("keyring key is empty")
	}
	v, err := keyring.Get(Service, key)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return Token{}, fmt.Errorf("%w: keyring entry %s/%s does not exist", ErrNotFound, Service, key)
		}
		return Token{}, fmt.Errorf("read token from keyring: %w", err)
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return Token{}, fmt.Errorf("%w: keyring entry %s/%s is empty", ErrNotFound, Service, key)
	}
	return NewToken(v), nil
}

// fromCommand runs an external command and reads the token from stdout. The
// command is executed directly with its argument vector, never through a
// shell, so a crafted config cannot inject shell metacharacters.
func fromCommand(ctx context.Context, argv []string) (Token, error) {
	if len(argv) == 0 {
		return Token{}, errors.New("token_command is empty")
	}
	ctx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	// The command may prompt on stderr (e.g. a GPG pinentry); let it through
	// only its own channel and never mix it into stdout.
	var stderr strings.Builder
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return Token{}, fmt.Errorf("token command %q timed out after %s", argv[0], commandTimeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return Token{}, fmt.Errorf("token command %q failed: %w: %s", argv[0], err, firstLine(msg))
		}
		return Token{}, fmt.Errorf("token command %q failed: %w", argv[0], err)
	}

	// Only surrounding whitespace is trimmed; the token itself is opaque.
	v := strings.TrimSpace(string(out))
	if v == "" {
		return Token{}, fmt.Errorf("%w: token command %q printed nothing", ErrNotFound, argv[0])
	}
	// Password managers often print extra lines; take the first non-empty one.
	if i := strings.IndexAny(v, "\r\n"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return NewToken(v), nil
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
