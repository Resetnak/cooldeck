package domain

import (
	"strings"
	"testing"
	"time"
)

// TestSanitizeLogTextStripsEscapeSequences is the security guard for the log
// views: nothing in an application's output may reach the terminal as a
// control sequence. A crafted log line could otherwise reposition the cursor,
// repaint the screen, set the window title, or -- worst -- trigger a terminal
// reply that gets injected back as if the user had typed it.
func TestSanitizeLogTextStripsEscapeSequences(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"SGR colour", "\x1b[31mred\x1b[0m", "red"},
		{"cursor move", "a\x1b[2Jb", "ab"},
		{"cursor home", "\x1b[H\x1b[2Jwiped", "wiped"},
		{"OSC window title", "\x1b]0;pwned\x07safe", "safe"},
		{"OSC with ST", "\x1b]8;;http://evil\x1b\\link", "link"},
		{"device control", "\x1bPq junk \x1b\\ok", "ok"},
		{"bare escape", "a\x1bb", "ab"},
		{"device status report", "\x1b[6n", ""},
		{"alt screen", "\x1b[?1049h", ""},
		{"bell", "ding\x07", "ding"},
		{"backspace", "ab\x08c", "abc"},
		{"carriage return overwrite", "real\rfake", "realfake"},
		{"null byte", "a\x00b", "ab"},
		{"tab expansion", "a\tb", "a    b"},
		{"zero width space", "a\u200bb", "ab"},
		{"plain text untouched", "GET /health 200", "GET /health 200"},
		{"unicode kept", "příliš žluťoučký kůň ✓", "příliš žluťoučký kůň ✓"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeLogText(tt.in)
			if got != tt.want {
				t.Errorf("SanitizeLogText(%q) = %q, want %q", tt.in, got, tt.want)
			}
			if strings.ContainsRune(got, 0x1b) {
				t.Errorf("escape byte survived: %q", got)
			}
		})
	}
}

func TestSanitizeLogTextTruncatesPathologicalLines(t *testing.T) {
	in := strings.Repeat("x", maxLineRunes*2)
	got := SanitizeLogText(in)
	if len([]rune(got)) > maxLineRunes+len("  …[truncated]") {
		t.Fatalf("line was not truncated, got %d runes", len([]rune(got)))
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Error("truncation should be visible to the user")
	}
}

func TestParseLogLineExtractsTimestamp(t *testing.T) {
	line := ParseLogLine("2026-08-04T08:15:30.123456789Z INFO server listening on :8080")
	if line.TimestampText != "2026-08-04T08:15:30.123456789Z" {
		t.Errorf("timestamp text = %q", line.TimestampText)
	}
	if line.Timestamp.IsZero() {
		t.Error("timestamp should have parsed")
	}
	if line.Text != "INFO server listening on :8080" {
		t.Errorf("text = %q", line.Text)
	}
	if line.Level != LogLevelInfo {
		t.Errorf("level = %v, want INFO", line.Level)
	}
	if got := line.String(); got != "2026-08-04T08:15:30.123456789Z INFO server listening on :8080" {
		t.Errorf("String() = %q", got)
	}

	// A line without a timestamp must keep all of its text.
	plain := ParseLogLine("just a message")
	if plain.TimestampText != "" || plain.Text != "just a message" {
		t.Errorf("plain line = %+v", plain)
	}
	if !plain.Timestamp.Equal(time.Time{}) {
		t.Error("plain line should have a zero timestamp")
	}
}

func TestDetectLogLevel(t *testing.T) {
	tests := map[string]LogLevel{
		"ERROR connection refused":                  LogLevelError,
		"[warn] disk almost full":                   LogLevelWarn,
		`{"level":"debug","msg":"x"}`:               LogLevelDebug,
		"level=info msg=started":                    LogLevelInfo,
		"time=... FATAL out of memory":              LogLevelFatal,
		"2026-08-04 panic: runtime error":           LogLevelFatal,
		"nothing noteworthy happened here":          LogLevelNone,
		"the error_handler module loaded":           LogLevelNone,
		"WARNING: certificate expires in 10 days":   LogLevelWarn,
		"request completed status=200 duration=4ms": LogLevelNone,
	}
	for line, want := range tests {
		if got := DetectLogLevel(line); got != want {
			t.Errorf("DetectLogLevel(%q) = %v, want %v", line, got, want)
		}
	}
}

func TestParseLogPayloadKeepsNewestLines(t *testing.T) {
	var b strings.Builder
	for i := range 100 {
		b.WriteString("line ")
		b.WriteString(string(rune('0' + i%10)))
		b.WriteByte('\n')
	}
	lines, truncated := ParseLogPayload(b.String(), 10)
	if !truncated {
		t.Error("dropping 90 of 100 lines must report truncated")
	}
	if len(lines) != 10 {
		t.Fatalf("got %d lines, want 10", len(lines))
	}
	// The tail is what matters in a log view.
	if lines[9].Text != "line 9" {
		t.Errorf("last line = %q, want %q", lines[9].Text, "line 9")
	}

	if got, _ := ParseLogPayload("", 10); got != nil {
		t.Errorf("empty payload should yield nil, got %v", got)
	}
	// A trailing newline must not produce a phantom blank line.
	if got, _ := ParseLogPayload("one\ntwo\n", 0); len(got) != 2 {
		t.Errorf("got %d lines, want 2", len(got))
	}
	// CRLF payloads are normalised.
	if got, _ := ParseLogPayload("one\r\ntwo\r\n", 0); len(got) != 2 || got[0].Text != "one" {
		t.Errorf("CRLF payload = %+v", got)
	}
}
