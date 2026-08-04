package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// LogLevel is the severity detected in a log line by heuristic, independent of
// the emitting application's log format.
type LogLevel int

// Detected log levels, ordered by severity.
const (
	LogLevelNone LogLevel = iota
	LogLevelTrace
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// LogLine is one sanitised line of output.
type LogLine struct {
	// Timestamp is the parsed leading timestamp, zero when the line had none.
	Timestamp time.Time
	// TimestampText is the raw leading timestamp as printed, kept so the view
	// can highlight it without re-formatting and changing the user's data.
	TimestampText string
	// Text is the sanitised remainder of the line.
	Text  string
	Level LogLevel
	// Stream is "stdout" or "stderr" when the source distinguishes them.
	Stream string
}

// String renders the line back to plain text.
func (l LogLine) String() string {
	if l.TimestampText == "" {
		return l.Text
	}
	return l.TimestampText + " " + l.Text
}

// maxLineRunes truncates pathological single lines (minified bundles, base64
// blobs) that would otherwise make wrapping and rendering quadratic.
const maxLineRunes = 8192

// ansiPattern matches CSI, OSC and the single-character escapes a hostile or
// merely enthusiastic log line might contain. Terminal escape sequences from
// an application's logs must never reach the terminal: they can reposition the
// cursor, repaint the screen, change the window title or, on some terminals,
// trigger a reply that is injected back as keyboard input.
var ansiPattern = regexp.MustCompile(
	"\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)" + // OSC ... BEL / ST
		"|\x1b\\[[0-?]*[ -/]*[@-~]" + // CSI
		"|\x1b[PX^_][^\x1b]*(?:\x1b\\\\)?" + // DCS / SOS / PM / APC
		"|\x1b[@-Z\\\\-_]" + // two-character escapes
		"|\x1b", // a bare, truncated escape
)

// timestampPattern matches the ISO-8601 / RFC3339 prefix Docker and Coolify
// emit when timestamps are requested.
var timestampPattern = regexp.MustCompile(
	`^(\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)\s+`)

var levelPattern = regexp.MustCompile(
	`(?i)(?:^|[\s\[\("'|<=,])(TRACE|DEBUG|DBG|INFO|INFORMATION|NOTICE|WARN|WARNING|ERROR|ERR|FATAL|CRITICAL|PANIC|EMERGENCY)(?:[\s\]\)"':|>.,-]|$)`)

// SanitizeLogText strips terminal escape sequences and control characters from
// untrusted log output, leaving printable text the renderer can safely style.
func SanitizeLogText(s string) string {
	if s == "" {
		return s
	}
	s = ansiPattern.ReplaceAllString(s, "")

	var b strings.Builder
	b.Grow(len(s))
	count := 0
	for _, r := range s {
		if count >= maxLineRunes {
			b.WriteString(" …[truncated]")
			break
		}
		switch {
		case r == '\t':
			// Tabs would be interpreted by the terminal against its own tab
			// stops and break column alignment; expand to a fixed width.
			b.WriteString("    ")
			count += 4
			continue
		case r == '\r' || r == '\n':
			// Carriage returns let a line overwrite what is already on screen.
			continue
		case r == utf8.RuneError:
			b.WriteRune('�')
		case unicode.IsControl(r):
			continue
		case r == '\u200b' || r == '\u2028' || r == '\u2029' || r == '\ufeff':
			// Zero-width and line/paragraph separators confuse width maths.
			continue
		default:
			b.WriteRune(r)
		}
		count++
	}
	return b.String()
}

// ParseLogLine sanitises a raw line and extracts its timestamp and level.
func ParseLogLine(raw string) LogLine {
	line := LogLine{Text: SanitizeLogText(raw)}

	if m := timestampPattern.FindStringSubmatch(line.Text); m != nil {
		line.TimestampText = m[1]
		line.Text = line.Text[len(m[0]):]
		if ts, err := parseTimestamp(m[1]); err == nil {
			line.Timestamp = ts
		}
	}
	line.Level = DetectLogLevel(line.Text)
	return line
}

// ParseLogPayload splits a raw multi-line log blob into sanitised lines,
// keeping at most maxLines of the newest output.
func ParseLogPayload(payload string, maxLines int) []LogLine {
	if payload == "" {
		return nil
	}
	raw := strings.Split(strings.ReplaceAll(payload, "\r\n", "\n"), "\n")
	// Drop the trailing empty element produced by a final newline.
	if n := len(raw); n > 0 && strings.TrimSpace(raw[n-1]) == "" {
		raw = raw[:n-1]
	}
	if maxLines > 0 && len(raw) > maxLines {
		raw = raw[len(raw)-maxLines:]
	}

	lines := make([]LogLine, 0, len(raw))
	for _, r := range raw {
		lines = append(lines, ParseLogLine(r))
	}
	return lines
}

// DetectLogLevel finds a severity token anywhere in the line. It is format
// agnostic on purpose: logfmt, JSON and plain prose all get a usable answer.
func DetectLogLevel(s string) LogLevel {
	m := levelPattern.FindStringSubmatch(s)
	if m == nil {
		return LogLevelNone
	}
	switch strings.ToUpper(m[1]) {
	case "TRACE":
		return LogLevelTrace
	case "DEBUG", "DBG":
		return LogLevelDebug
	case "INFO", "INFORMATION", "NOTICE":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR", "ERR":
		return LogLevelError
	case "FATAL", "CRITICAL", "PANIC", "EMERGENCY":
		return LogLevelFatal
	default:
		return LogLevelNone
	}
}

// Label returns the short uppercase name of the level.
func (l LogLevel) Label() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

var timestampLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
}

func parseTimestamp(s string) (time.Time, error) {
	var err error
	for _, layout := range timestampLayouts {
		var t time.Time
		if t, err = time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, err
}
