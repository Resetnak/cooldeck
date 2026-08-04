package tui

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Run starts the TUI and blocks until the user quits or ctx is cancelled.
func Run(ctx context.Context, opts Options) error {
	if opts.Service == nil {
		return fmt.Errorf("no service configured")
	}
	if !opts.ASCII {
		opts.ASCII = !terminalSupportsUnicode()
	}

	p := tea.NewProgram(New(opts), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

// terminalSupportsUnicode guesses whether the terminal can render the
// box-drawing and geometric glyphs the default symbol set uses. A wrong guess
// only changes glyphs, never behaviour, so a cheap heuristic is the right
// trade-off against probing the terminal.
func terminalSupportsUnicode() bool {
	for _, v := range []string{os.Getenv("LC_ALL"), os.Getenv("LC_CTYPE"), os.Getenv("LANG")} {
		if v == "" {
			continue
		}
		u := strings.ToUpper(v)
		return strings.Contains(u, "UTF-8") || strings.Contains(u, "UTF8")
	}
	// Windows Terminal and modern macOS terminals do not always set a locale
	// but do handle UTF-8, so an unset locale is not treated as a refusal.
	return true
}
