package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// secret mimics the shape of a Coolify (Laravel Sanctum) personal access token.
const secret = "7|qA3xZk9LmPb2NrTvWy8Hc4Ju6Ef1Sd0Gh5Ki2Lo"

func TestRedact(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"bearer header", "Authorization: Bearer " + secret},
		{"sanctum token in prose", "using token " + secret + " for production"},
		{"query parameter", "https://coolify.example.com/api/v1/applications?token=" + secret},
		{"json field", `{"api_key":"` + secret + `"}`},
		{"lowercase bearer", "bearer " + secret},
		{"env dump", "boot: DATABASE_PASSWORD=" + secret + " ok"},
		{"quoted json pair", `{"aws_secret_access_key": "` + secret + `"}`},
		{"jwt", "session eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r_wW1gFWFOEjXk"},
		{"basic auth url", "cloning https://ci:" + secret + "@git.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Redact(tt.in)
			if strings.Contains(got, secret) {
				t.Fatalf("token survived redaction: %q", got)
			}
			if !strings.Contains(got, Mask) {
				t.Fatalf("no mask inserted: %q", got)
			}
		})
	}
}

func TestRedactKeepsHarmlessText(t *testing.T) {
	for _, in := range []string{
		"GET https://coolify.example.com/api/v1/applications -> 200 in 84ms",
		"credentials.go:42 keyring unavailable",
		"password reset email sent",
		"GET /api/v1/security?limit=10",
		`{"url":"https://coolify.example.com:8000","email":"ops@example.com"}`,
	} {
		if got := Redact(in); got != in {
			t.Fatalf("harmless text was altered:\n got %q\nwant %q", got, in)
		}
	}
}

func TestIsSensitiveKey(t *testing.T) {
	for _, k := range []string{"token", "Authorization", "app_token", "X-Auth-Password", "private_key"} {
		if !IsSensitiveKey(k) {
			t.Errorf("expected %q to be sensitive", k)
		}
	}
	for _, k := range []string{"url", "status", "instance", "duration_ms"} {
		if IsSensitiveKey(k) {
			t.Errorf("expected %q to be safe", k)
		}
	}
}

// TestLoggerNeverWritesSecrets exercises the whole handler chain: a token
// passed as an attribute value, as part of a message and inside an error must
// not reach the encoded output.
func TestLoggerNeverWritesSecrets(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{ReplaceAttr: redactAttr})
	l := &Logger{}
	l.Logger = slog.New(&tapHandler{next: h, owner: l})

	l.Info("connecting", "token", secret, "url", "https://coolify.example.com")
	l.Warn("request failed with Bearer " + secret)
	l.Error("auth error", "err", errString("invalid token "+secret))

	out := buf.String()
	if strings.Contains(out, secret) {
		t.Fatalf("secret leaked into log output:\n%s", out)
	}
	// The buffered records feeding the diagnostics screen must be clean too.
	for _, r := range l.Recent() {
		if strings.Contains(r.Message, secret) {
			t.Fatalf("secret leaked into recent records: %q", r.Message)
		}
	}
	// Output must still be valid JSON lines carrying the non-secret context.
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("invalid JSON log line %q: %v", line, err)
		}
	}
	if !strings.Contains(out, "coolify.example.com") {
		t.Fatal("redaction removed non-sensitive context")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
