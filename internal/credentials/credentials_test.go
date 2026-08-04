package credentials

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/resetnak/cooldeck/internal/config"
)

const secret = "7|qA3xZk9LmPb2NrTvWy8Hc4Ju6Ef1Sd0Gh5Ki2Lo"

// TestTokenNeverFormats is the guard that keeps a token out of logs, errors and
// snapshots: every formatting path must yield the placeholder.
func TestTokenNeverFormats(t *testing.T) {
	tok := NewToken(secret)

	renders := []string{
		fmt.Sprint(tok),
		fmt.Sprintf("%v", tok),
		fmt.Sprintf("%+v", tok),
		fmt.Sprintf("%#v", tok),
		fmt.Sprintf("%v", struct{ T Token }{tok}),
		fmt.Sprintf("%v", []Token{tok}),
		fmt.Sprintf("%v", map[string]Token{"prod": tok}),
	}
	if b, err := json.Marshal(tok); err == nil {
		renders = append(renders, string(b))
	} else {
		t.Errorf("json.Marshal: %v", err)
	}
	if b, err := json.Marshal(struct {
		Token Token `json:"token"`
	}{tok}); err == nil {
		renders = append(renders, string(b))
	}

	for _, got := range renders {
		if strings.Contains(got, secret) {
			t.Errorf("token leaked through formatting: %q", got)
		}
		if !strings.Contains(got, Redacted) {
			t.Errorf("expected placeholder, got %q", got)
		}
	}
	if tok.Secret() != secret {
		t.Error("Secret() must return the real value")
	}
	if !NewToken("").IsZero() {
		t.Error("empty token should be zero")
	}
}

func TestResolveEnvOverrideWinsOverEverything(t *testing.T) {
	t.Setenv(EnvOverride, "  "+secret+"\n")

	inst := config.Instance{ID: "prod", TokenSource: config.TokenSourcePlaintext, Token: "different"}
	tok, origin, err := Resolve(t.Context(), inst)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if origin != OriginEnvOverride {
		t.Errorf("origin = %q, want %q", origin, OriginEnvOverride)
	}
	if tok.Secret() != secret {
		t.Error("override value should win and be whitespace-trimmed")
	}
}

func TestResolveEnvSource(t *testing.T) {
	t.Setenv(EnvOverride, "")
	t.Setenv("MY_COOLIFY_TOKEN", secret)

	inst := config.Instance{ID: "prod", TokenSource: config.TokenSourceEnv, TokenEnv: "MY_COOLIFY_TOKEN"}
	tok, origin, err := Resolve(t.Context(), inst)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if origin != OriginEnv || tok.Secret() != secret {
		t.Fatalf("got (%q, %v)", origin, tok)
	}

	inst.TokenEnv = "DEFINITELY_UNSET_TOKEN_VAR"
	if _, _, err := Resolve(t.Context(), inst); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestResolvePlaintextSource(t *testing.T) {
	t.Setenv(EnvOverride, "")
	inst := config.Instance{ID: "prod", TokenSource: config.TokenSourcePlaintext, Token: secret}
	tok, origin, err := Resolve(t.Context(), inst)
	if err != nil || origin != OriginPlaintext || tok.Secret() != secret {
		t.Fatalf("got (%v, %q, %v)", tok, origin, err)
	}
}

func TestResolveCommandSource(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell-free echo helper")
	}
	t.Setenv(EnvOverride, "")

	// printf writes the token plus trailing noise the resolver must discard.
	inst := config.Instance{
		ID:           "prod",
		TokenSource:  config.TokenSourceCommand,
		TokenCommand: []string{"printf", secret + "\nnoise\n"},
	}
	tok, origin, err := Resolve(t.Context(), inst)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if origin != OriginCommand {
		t.Errorf("origin = %q", origin)
	}
	if tok.Secret() != secret {
		t.Errorf("got %q, want the first line only", tok.Secret())
	}
}

func TestResolveCommandFailureIsDescriptiveAndClean(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX helper")
	}
	t.Setenv(EnvOverride, "")

	inst := config.Instance{
		ID:           "prod",
		TokenSource:  config.TokenSourceCommand,
		TokenCommand: []string{"false"},
	}
	_, _, err := Resolve(t.Context(), inst)
	if err == nil {
		t.Fatal("expected an error from a failing command")
	}
	if !strings.Contains(err.Error(), "false") {
		t.Errorf("error should name the command: %v", err)
	}
}

// TestResolveCommandDoesNotUseShell proves the argv is executed directly: a
// shell metacharacter must be treated as a literal argument, not a command.
func TestResolveCommandDoesNotUseShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX helper")
	}
	t.Setenv(EnvOverride, "")

	marker := t.TempDir() + "/pwned"
	inst := config.Instance{
		ID:           "prod",
		TokenSource:  config.TokenSourceCommand,
		TokenCommand: []string{"printf", "%s", "tok; touch " + marker},
	}
	tok, _, err := Resolve(t.Context(), inst)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if tok.Secret() != "tok; touch "+marker {
		t.Errorf("argument was reinterpreted: %q", tok.Secret())
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("the injected command executed; argv must not go through a shell")
	}
}

func TestKeyringRoundTrip(t *testing.T) {
	keyring.MockInit()
	t.Setenv(EnvOverride, "")

	inst := config.Instance{ID: "prod", TokenSource: config.TokenSourceKeyring}

	if _, _, err := Resolve(t.Context(), inst); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound before storing", err)
	}

	if err := Store(KeyFor(inst), NewToken(secret)); err != nil {
		t.Fatalf("Store: %v", err)
	}
	tok, origin, err := Resolve(t.Context(), inst)
	if err != nil || origin != OriginKeyring || tok.Secret() != secret {
		t.Fatalf("got (%v, %q, %v)", tok, origin, err)
	}

	if err := Remove(KeyFor(inst)); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	// Deleting twice must stay quiet so instance removal is idempotent.
	if err := Remove(KeyFor(inst)); err != nil {
		t.Fatalf("second Remove should be a no-op, got %v", err)
	}
	if _, _, err := Resolve(t.Context(), inst); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound after removal", err)
	}
}

func TestStoreRejectsEmptyToken(t *testing.T) {
	keyring.MockInit()
	if err := Store("prod", Token{}); err == nil {
		t.Error("storing an empty token should fail")
	}
	if err := Store("", NewToken(secret)); err == nil {
		t.Error("storing without a key should fail")
	}
}

func TestCheckReportsWithoutLeaking(t *testing.T) {
	keyring.MockInit()
	t.Setenv(EnvOverride, "")

	inst := config.Instance{ID: "prod", TokenSource: config.TokenSourceKeyring, TokenKey: "prod"}
	if st := Check(t.Context(), inst); st.Available {
		t.Error("expected unavailable before storing")
	}

	if err := Store("prod", NewToken(secret)); err != nil {
		t.Fatalf("Store: %v", err)
	}
	st := Check(t.Context(), inst)
	if !st.Available || st.Err != nil {
		t.Fatalf("expected available, got %+v", st)
	}
	if st.Detail != Service+"/prod" {
		t.Errorf("detail = %q", st.Detail)
	}
	if strings.Contains(fmt.Sprintf("%+v", st), secret) {
		t.Error("status must never contain the token")
	}
}

// TestCheckDetailHidesCommandArguments guards against a secret path (which can
// itself be sensitive) being echoed back in status output.
func TestCheckDetailHidesCommandArguments(t *testing.T) {
	inst := config.Instance{
		ID:           "prod",
		TokenSource:  config.TokenSourceCommand,
		TokenCommand: []string{"gopass", "show", "personal/coolify/prod"},
	}
	if got := detailFor(inst); strings.Contains(got, "personal/coolify/prod") {
		t.Errorf("detail leaked the secret path: %q", got)
	}
}
