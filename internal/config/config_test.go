package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNormalizeBaseURL(t *testing.T) {
	ok := []struct{ in, want string }{
		{"https://coolify.example.com", "https://coolify.example.com"},
		{"https://coolify.example.com/", "https://coolify.example.com"},
		{"https://coolify.example.com/api/v1", "https://coolify.example.com"},
		{"https://coolify.example.com/api/v1/", "https://coolify.example.com"},
		{"https://coolify.example.com/api", "https://coolify.example.com"},
		{"coolify.example.com", "https://coolify.example.com"},
		{"  https://coolify.example.com  ", "https://coolify.example.com"},
		{"http://localhost:8000", "http://localhost:8000"},
		// A reverse-proxy sub-path must survive; only the API prefix is cut.
		{"https://example.com/coolify/api/v1", "https://example.com/coolify"},
	}
	for _, tt := range ok {
		got, err := NormalizeBaseURL(tt.in)
		if err != nil {
			t.Errorf("NormalizeBaseURL(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("NormalizeBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	bad := []string{
		"",
		"   ",
		"ftp://coolify.example.com",
		"file:///etc/passwd",
		"https://",
		"https://user:pass@coolify.example.com",
		"https://coolify.example.com?token=abc",
	}
	for _, in := range bad {
		if got, err := NormalizeBaseURL(in); err == nil {
			t.Errorf("NormalizeBaseURL(%q) = %q, want error", in, got)
		}
	}
}

func TestInstanceValidate(t *testing.T) {
	base := Instance{URL: "https://coolify.example.com"}

	valid := []Instance{
		{ID: "a", URL: base.URL, TokenSource: TokenSourceKeyring},
		{ID: "b", URL: base.URL, TokenSource: TokenSourceCommand, TokenCommand: []string{"gopass", "show", "x"}},
		{ID: "c", URL: base.URL, TokenSource: TokenSourceEnv, TokenEnv: "MY_TOKEN"},
		{ID: "d", URL: base.URL, TokenSource: TokenSourcePlaintext, Token: "7|abc"},
	}
	for _, inst := range valid {
		if err := inst.Validate(); err != nil {
			t.Errorf("instance %q should be valid: %v", inst.ID, err)
		}
	}

	invalid := map[string]Instance{
		"missing url":         {TokenSource: TokenSourceKeyring},
		"missing source":      {URL: base.URL},
		"unknown source":      {URL: base.URL, TokenSource: "vault"},
		"command without cmd": {URL: base.URL, TokenSource: TokenSourceCommand},
		"env without name":    {URL: base.URL, TokenSource: TokenSourceEnv},
		"plaintext w/o token": {URL: base.URL, TokenSource: TokenSourcePlaintext},
		// A token present under a non-plaintext source is inert but leaked.
		"stray token": {URL: base.URL, TokenSource: TokenSourceKeyring, Token: "7|abc"},
		"fast refresh": {URL: base.URL, TokenSource: TokenSourceKeyring,
			RefreshInterval: Duration(time.Second)},
	}
	for name, inst := range invalid {
		if err := inst.Validate(); err == nil {
			t.Errorf("%s should be invalid", name)
		}
	}
}

func TestConfigValidateCollectsAllProblems(t *testing.T) {
	cfg := Default()
	cfg.Theme = "neon"
	cfg.RefreshInterval = Duration(time.Second)
	cfg.DefaultInstance = "ghost"
	cfg.Instances["bad name!"] = Instance{ID: "bad name!", URL: "nope://x", TokenSource: TokenSourceKeyring}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation to fail")
	}
	msg := err.Error()
	for _, want := range []string{"theme", "refresh_interval", "default_instance", "bad name!"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error should mention %q:\n%s", want, msg)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "absent.toml"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	path := writeConfig(t, `
version = 1
default_instance = "prod"

[instances.prod]
name = "Production"
url = "https://coolify.example.com/api/v1/"
token_source = "keyring"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if time.Duration(cfg.RefreshInterval) != DefaultRefreshInterval {
		t.Errorf("refresh interval = %s, want default %s", cfg.RefreshInterval, DefaultRefreshInterval)
	}
	if cfg.LogLines != DefaultLogLines {
		t.Errorf("log_lines = %d, want %d", cfg.LogLines, DefaultLogLines)
	}
	if !cfg.ConfirmDestructiveActions {
		t.Error("confirm_destructive_actions should default to true")
	}
	inst, err := cfg.Instance("")
	if err != nil {
		t.Fatalf("Instance: %v", err)
	}
	if inst.ID != "prod" {
		t.Errorf("instance ID = %q, want prod", inst.ID)
	}
	if inst.DisplayName() != "Production" {
		t.Errorf("display name = %q", inst.DisplayName())
	}
}

func TestLoadReportsUnknownKeys(t *testing.T) {
	path := writeConfig(t, `
version = 1
refresh_intervall = "10s"

[instances.prod]
url = "https://coolify.example.com"
token_source = "keyring"
`)
	cfg, err := Load(path)
	if !errors.Is(err, ErrUnknownKeys) {
		t.Fatalf("got %v, want ErrUnknownKeys", err)
	}
	// Despite the warning the config must remain usable.
	if len(cfg.Instances) != 1 {
		t.Fatalf("config should still be usable, got %d instances", len(cfg.Instances))
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	cfg := Default()
	cfg.DefaultInstance = "prod"
	cfg.RefreshInterval = Duration(15 * time.Second)
	cfg.Instances["prod"] = Instance{
		ID:          "prod",
		Name:        "Production",
		URL:         "https://coolify.example.com",
		TokenSource: TokenSourceKeyring,
		TokenKey:    "prod",
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	// The file may hold a plaintext token, so it must not be group/world readable.
	// Windows does not enforce Unix permission bits the same way; skip there.
	if runtime.GOOS != "windows" {
		if perm := fi.Mode().Perm(); perm != 0o600 {
			t.Errorf("config permissions = %o, want 600", perm)
		}
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if time.Duration(got.RefreshInterval) != 15*time.Second {
		t.Errorf("refresh interval = %s, want 15s", got.RefreshInterval)
	}
	if got.Instances["prod"].URL != "https://coolify.example.com" {
		t.Errorf("url = %q", got.Instances["prod"].URL)
	}
}

func TestRemoveInstanceUpdatesDefault(t *testing.T) {
	cfg := Default()
	cfg.DefaultInstance = "prod"
	cfg.Instances = map[string]Instance{
		"prod": {ID: "prod", Name: "Prod", URL: "https://a.example"},
		"dev":  {ID: "dev", Name: "Dev", URL: "https://b.example"},
	}
	if err := cfg.RemoveInstance("prod"); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Instances["prod"]; ok {
		t.Fatal("prod still present")
	}
	if cfg.DefaultInstance != "dev" {
		t.Fatalf("default = %q, want dev", cfg.DefaultInstance)
	}
	if err := cfg.RemoveInstance("dev"); err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultInstance != "" || len(cfg.Instances) != 0 {
		t.Fatalf("expected empty config, default=%q n=%d", cfg.DefaultInstance, len(cfg.Instances))
	}
	if err := cfg.RemoveInstance("missing"); err == nil {
		t.Fatal("expected error for missing instance")
	}
}

func TestEffectiveRefreshIntervalClampsToMinimum(t *testing.T) {
	cfg := Default()
	inst := Instance{ID: "prod", RefreshInterval: Duration(500 * time.Millisecond)}
	if got := cfg.EffectiveRefreshInterval(inst); got != MinRefreshInterval {
		t.Errorf("got %s, want clamp to %s", got, MinRefreshInterval)
	}
	inst.RefreshInterval = Duration(30 * time.Second)
	if got := cfg.EffectiveRefreshInterval(inst); got != 30*time.Second {
		t.Errorf("got %s, want 30s override", got)
	}
	if got := cfg.EffectiveRefreshInterval(Instance{ID: "x"}); got != DefaultRefreshInterval {
		t.Errorf("got %s, want global %s", got, DefaultRefreshInterval)
	}
}

func TestInstanceLookup(t *testing.T) {
	cfg := Default()
	if _, err := cfg.Instance(""); err == nil {
		t.Error("empty config should not resolve an instance")
	}

	cfg.Instances["only"] = Instance{ID: "only"}
	// A single instance is unambiguous even without default_instance.
	if inst, err := cfg.Instance(""); err != nil || inst.ID != "only" {
		t.Errorf("got (%v, %v), want the sole instance", inst.ID, err)
	}

	cfg.Instances["second"] = Instance{ID: "second"}
	if _, err := cfg.Instance(""); err == nil {
		t.Error("ambiguous lookup should fail without default_instance")
	}
	if _, err := cfg.Instance("nope"); err == nil {
		t.Error("unknown instance should fail")
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
