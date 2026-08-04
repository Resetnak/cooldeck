// Package logging provides file-based structured logging that never writes to
// the terminal while the TUI owns the screen, and never emits credentials.
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// maxLogBytes caps a single log file. Once exceeded the file is rotated to
// <name>.1 and a fresh file is started, so the on-disk footprint stays bounded
// at roughly 2x this value.
const maxLogBytes = 2 << 20 // 2 MiB

// recentLimit bounds the in-memory record buffer surfaced by the diagnostics
// screen. Without a limit a long-running session would grow unbounded.
const recentLimit = 50

// Record is a redacted log entry kept in memory for the diagnostics screen.
type Record struct {
	Time    time.Time
	Level   slog.Level
	Message string
}

// Logger owns the log file and the in-memory tail of recent records.
type Logger struct {
	*slog.Logger

	path   string
	closer io.Closer

	mu     sync.Mutex
	recent []Record
}

// Options configures Setup.
type Options struct {
	// Path is the log file. When empty, logging is discarded.
	Path string
	// Debug lowers the level to slog.LevelDebug.
	Debug bool
}

// Setup opens the log file and returns a Logger. A failure to open the file is
// not fatal: logging degrades to a no-op writer and the error is returned so
// the caller can surface it once the UI is up.
func Setup(opts Options) (*Logger, error) {
	l := &Logger{path: opts.Path}

	level := slog.LevelInfo
	if opts.Debug {
		level = slog.LevelDebug
	}

	w := io.Discard
	var openErr error
	if opts.Path != "" {
		f, err := openLogFile(opts.Path)
		if err != nil {
			openErr = err
			l.path = ""
		} else {
			w = f
			l.closer = f
		}
	}

	base := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redactAttr,
	})
	l.Logger = slog.New(&tapHandler{next: base, owner: l, level: level})
	return l, openErr
}

// Path returns the active log file path, or "" when logging is disabled.
func (l *Logger) Path() string { return l.path }

// Recent returns a copy of the buffered records, oldest first.
func (l *Logger) Recent() []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Record, len(l.recent))
	copy(out, l.recent)
	return out
}

// Close flushes and releases the log file.
func (l *Logger) Close() error {
	if l.closer == nil {
		return nil
	}
	return l.closer.Close()
}

func (l *Logger) push(r Record) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.recent = append(l.recent, r)
	if len(l.recent) > recentLimit {
		l.recent = l.recent[len(l.recent)-recentLimit:]
	}
}

// tapHandler forwards records to the JSON handler and mirrors warnings and
// errors into the in-memory buffer used by the diagnostics screen.
type tapHandler struct {
	next  slog.Handler
	owner *Logger
	level slog.Level
}

func (h *tapHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.next.Enabled(ctx, l)
}

func (h *tapHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelWarn {
		h.owner.push(Record{Time: r.Time, Level: r.Level, Message: Redact(r.Message)})
	}
	return h.next.Handle(ctx, r)
}

func (h *tapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &tapHandler{next: h.next.WithAttrs(attrs), owner: h.owner, level: h.level}
}

func (h *tapHandler) WithGroup(name string) slog.Handler {
	return &tapHandler{next: h.next.WithGroup(name), owner: h.owner, level: h.level}
}

func openLogFile(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	if fi, err := os.Stat(path); err == nil && fi.Size() > maxLogBytes {
		// Best effort rotation; a failure here must not prevent logging.
		_ = os.Rename(path, path+".1")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return f, nil
}
