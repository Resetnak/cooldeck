// Package components holds the reusable rendering primitives the screens are
// assembled from. Components are pure: they take a theme, a layout and data,
// and return a string. They never perform I/O and never mutate global state.
package components

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Width returns the rendered cell width of a styled string, accounting for
// wide CJK characters, combining marks and embedded ANSI sequences.
func Width(s string) int { return ansi.StringWidth(s) }

// Truncate shortens a styled string to width cells, appending an ellipsis when
// anything was cut. It is ANSI- and Unicode-aware, so styling survives and a
// double-width glyph is never split in half.
func Truncate(s string, width int, ellipsis string) string {
	if width <= 0 {
		return ""
	}
	if Width(s) <= width {
		return s
	}
	if Width(ellipsis) >= width {
		// No room for both content and the marker; show as much as fits.
		return ansi.Truncate(s, width, "")
	}
	return ansi.Truncate(s, width, ellipsis)
}

// Pad right-pads a styled string to exactly width cells, truncating when it is
// too long. Every table cell goes through this, which is what keeps columns
// aligned regardless of the content.
func Pad(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if w := Width(s); w > width {
		// Truncation can land short of the target when the cut falls inside a
		// double-width glyph, which cannot be split; pad the remainder.
		s = ansi.Truncate(s, width, "")
	}
	return s + strings.Repeat(" ", max(width-Width(s), 0))
}

// PadLeft left-pads a styled string to exactly width cells, for right-aligned
// columns such as durations and counts.
func PadLeft(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if w := Width(s); w > width {
		s = ansi.Truncate(s, width, "")
	}
	return strings.Repeat(" ", max(width-Width(s), 0)) + s
}

// Fit truncates and then pads, producing a cell of exactly width cells.
func Fit(s string, width int, ellipsis string) string {
	return Pad(Truncate(s, width, ellipsis), width)
}

// Wrap hard-wraps text to width cells, breaking on word boundaries where
// possible. Used for modal bodies and empty-state copy.
func Wrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	return ansi.Wrap(s, width, "")
}

// Lines splits a rendered block into its lines.
func Lines(s string) []string { return strings.Split(s, "\n") }

// FitBlock forces a rendered block to exactly height lines, padding with
// blanks or cutting the overflow. Panels rely on this so a frame never shifts
// because one region rendered a line more than expected.
func FitBlock(s string, width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := Lines(s)
	if len(lines) > height {
		lines = lines[:height]
	}
	out := make([]string, 0, height)
	for _, l := range lines {
		out = append(out, Pad(l, width))
	}
	blank := strings.Repeat(" ", max(width, 0))
	for len(out) < height {
		out = append(out, blank)
	}
	return strings.Join(out, "\n")
}

// cut returns the cells [left, right) of a styled string, preserving styling.
func cut(s string, left, right int) string { return ansi.Cut(s, left, right) }

// stripANSI removes styling from a string, keeping its printable content. It
// is used where a container needs to impose one uniform style, such as a
// selected table row.
func stripANSI(s string) string { return ansi.Strip(s) }

// Dash is what an absent value renders as, so empty cells read as intentional
// rather than broken.
const Dash = "-"

// OrDash returns s, or the em dash when s is blank.
func OrDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return Dash
	}
	return s
}
