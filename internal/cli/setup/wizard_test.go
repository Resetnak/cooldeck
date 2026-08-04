package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/resetnak/cooldeck/internal/config"
)

func TestBuildConfigKeyring(t *testing.T) {
	w := &wizard{
		instanceID:   "production",
		instanceName: "Production",
		coolifyURL:   "https://coolify.example.com",
		tokenSource:  config.TokenSourceKeyring,
		token:        "secret-token",
		theme:        config.ThemeDark,
	}
	cfg := w.buildConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	inst := cfg.Instances["production"]
	if inst.TokenSource != config.TokenSourceKeyring {
		t.Fatalf("token source = %q", inst.TokenSource)
	}
	if inst.TokenKey != "production" {
		t.Fatalf("token key = %q", inst.TokenKey)
	}
	if inst.Token != "" {
		t.Fatal("keyring config must not embed the token")
	}
	if cfg.Theme != config.ThemeDark {
		t.Fatalf("theme = %q", cfg.Theme)
	}
}

func TestBuildConfigEnv(t *testing.T) {
	w := &wizard{
		instanceID:   "staging",
		instanceName: "Staging",
		coolifyURL:   "https://coolify.example.com",
		tokenSource:  config.TokenSourceEnv,
		tokenEnvVar:  "COOLIFY_API_TOKEN",
		theme:        config.ThemeAuto,
	}
	cfg := w.buildConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	inst := cfg.Instances["staging"]
	if inst.TokenEnv != "COOLIFY_API_TOKEN" {
		t.Fatalf("token env = %q", inst.TokenEnv)
	}
}

func TestBuildConfigPlaintext(t *testing.T) {
	w := &wizard{
		instanceID:   "local",
		instanceName: "Local",
		coolifyURL:   "http://localhost:8000",
		tokenSource:  config.TokenSourcePlaintext,
		token:        "plain-secret",
		theme:        config.ThemeNord,
	}
	cfg := w.buildConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.Instances["local"].Token != "plain-secret" {
		t.Fatal("plaintext token missing from config")
	}
}

func TestPersistConfigKeyringFailureDoesNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := config.Default()
	cfg.DefaultInstance = "prod"
	cfg.Theme = config.ThemeDark
	cfg.Instances = map[string]config.Instance{
		"prod": {
			ID:          "prod",
			Name:        "Prod",
			URL:         "https://coolify.example.com",
			TokenSource: config.TokenSourceKeyring,
			TokenKey:    "prod",
		},
	}

	// Empty token must fail before writing the file.
	err := persistConfig(cfg, "", path)
	if err == nil {
		t.Fatal("expected error for empty keyring token")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("config file should not exist after keyring failure, stat: %v", statErr)
	}
}

func TestPersistConfigEnvWritesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := config.Default()
	cfg.DefaultInstance = "prod"
	cfg.Theme = config.ThemeCatppuccin
	cfg.Instances = map[string]config.Instance{
		"prod": {
			ID:          "prod",
			Name:        "Prod",
			URL:         "https://coolify.example.com",
			TokenSource: config.TokenSourceEnv,
			TokenEnv:    "COOLIFY_API_TOKEN",
		},
	}

	if err := persistConfig(cfg, "", path); err != nil {
		t.Fatalf("persist: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "token_source = \"env\"") {
		t.Fatalf("missing token_source in:\n%s", body)
	}
	if strings.Contains(body, "token =") {
		t.Fatalf("env config must not write a token field:\n%s", body)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows does not preserve Unix 0600 the way POSIX filesystems do.
	if runtime.GOOS != "windows" {
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Fatalf("config perms = %o, want 0600", perm)
		}
	}
}

func TestPersistConfigPlaintextWritesToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := config.Default()
	cfg.DefaultInstance = "prod"
	cfg.Instances = map[string]config.Instance{
		"prod": {
			ID:          "prod",
			Name:        "Prod",
			URL:         "https://coolify.example.com",
			TokenSource: config.TokenSourcePlaintext,
			Token:       "top-secret",
		},
	}

	if err := persistConfig(cfg, "", path); err != nil {
		t.Fatalf("persist: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "top-secret") {
		t.Fatal("plaintext token not written")
	}
}

func TestWindowAround(t *testing.T) {
	start, end := windowAround(0, 8, 3)
	if start != 0 || end != 3 {
		t.Fatalf("got [%d,%d)", start, end)
	}
	start, end = windowAround(7, 8, 3)
	if start != 5 || end != 8 {
		t.Fatalf("got [%d,%d)", start, end)
	}
	start, end = windowAround(2, 3, 10)
	if start != 0 || end != 3 {
		t.Fatalf("got [%d,%d)", start, end)
	}
}

func TestEnvVarNameOK(t *testing.T) {
	ok := []string{"COOLIFY_API_TOKEN", "_X", "a1"}
	bad := []string{"", "1ABC", "HAS-DASH", "has space", "dollar$"}
	for _, n := range ok {
		if !envVarNameOK(n) {
			t.Fatalf("expected ok: %q", n)
		}
	}
	for _, n := range bad {
		if envVarNameOK(n) {
			t.Fatalf("expected bad: %q", n)
		}
	}
}

func TestClearSecrets(t *testing.T) {
	w := New()
	w.token = "secret"
	w.tokenInput.SetValue("secret")
	w.clearSecrets()
	if w.token != "" || w.tokenInput.Value() != "" {
		t.Fatal("secrets were not cleared")
	}
}
