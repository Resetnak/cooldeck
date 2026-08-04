package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/resetnak/cooldeck/internal/version"
)

func TestVersionCommand(t *testing.T) {
	out := &bytes.Buffer{}
	err := Execute(context.Background(), []string{"version"}, strings.NewReader(""), out, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("Execute(version) error = %v", err)
	}
	if !strings.Contains(out.String(), version.AppName+" ") {
		t.Fatalf("version output = %q", out.String())
	}
}

func TestConfigPathHonoursOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "custom.toml")
	out := &bytes.Buffer{}
	err := Execute(
		context.Background(),
		[]string{"--config", want, "config", "path"},
		strings.NewReader(""),
		out,
		&bytes.Buffer{},
	)
	if err != nil {
		t.Fatalf("Execute(config path) error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != want {
		t.Fatalf("config path = %q, want %q", got, want)
	}
}

func TestConfigValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `
version = 1
default_instance = "demo"

[instances.demo]
url = "https://coolify.example.com"
token_source = "env"
token_env = "COOLIFY_TOKEN"
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	err := Execute(
		context.Background(),
		[]string{"--config", path, "config", "validate"},
		strings.NewReader(""),
		out,
		&bytes.Buffer{},
	)
	if err != nil {
		t.Fatalf("Execute(config validate) error = %v", err)
	}
	if !strings.Contains(out.String(), "valid: "+path+" (1 instances)") {
		t.Fatalf("validate output = %q", out.String())
	}
}

func TestInvalidThemeFailsBeforeStartingTUI(t *testing.T) {
	err := Execute(
		context.Background(),
		[]string{"--demo", "--theme", "sepia"},
		strings.NewReader(""),
		&bytes.Buffer{},
		&bytes.Buffer{},
	)
	if err == nil || !strings.Contains(err.Error(), "invalid theme") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadTokenFromPipe(t *testing.T) {
	got, err := readToken(strings.NewReader("  secret-value  \n"), &bytes.Buffer{})
	if err != nil {
		t.Fatalf("readToken() error = %v", err)
	}
	if got != "secret-value" {
		t.Fatalf("readToken() = %q", got)
	}
}
