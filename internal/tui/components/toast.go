package components

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// ToastKind selects a toast's colour and icon.
type ToastKind int

// Toast kinds.
const (
	ToastSuccess ToastKind = iota
	ToastInfo
	ToastWarning
	ToastError
)

// Toast is a transient notification.
type Toast struct {
	ID   int
	Kind ToastKind
	Text string
	// Detail is optional extra context reachable with "e"; it is never shown
	// inline, so a long error cannot take over the screen.
	Detail    string
	ExpiresAt time.Time
}

// MaxToasts bounds the stack. Without a cap, a flapping instance would fill
// the screen with duplicate errors.
const MaxToasts = 3

// ToastLifetime is how long each kind stays on screen. Errors linger because
// they usually require the user to do something.
func ToastLifetime(kind ToastKind) time.Duration {
	switch kind {
	case ToastError:
		return 12 * time.Second
	case ToastWarning:
		return 8 * time.Second
	default:
		return 4 * time.Second
	}
}

// PruneToasts drops expired toasts and enforces the stack limit, keeping the
// newest. It returns a new slice; the input is not modified.
func PruneToasts(toasts []Toast, now time.Time) []Toast {
	out := make([]Toast, 0, len(toasts))
	for _, t := range toasts {
		if t.ExpiresAt.After(now) {
			out = append(out, t)
		}
	}
	if len(out) > MaxToasts {
		out = out[len(out)-MaxToasts:]
	}
	return out
}

// RenderToasts draws the toast stack. The caller overlays the result in the
// bottom-right corner of the content area, above the footer, so it never
// covers the key hints the user needs to dismiss it.
func RenderToasts(th *theme.Theme, toasts []Toast, maxWidth int) string {
	if len(toasts) == 0 {
		return ""
	}
	width := min(maxWidth, 48)
	if width < 12 {
		return ""
	}

	lines := make([]string, 0, len(toasts))
	for _, t := range toasts {
		style, icon := toastStyle(th, t.Kind)
		text := Truncate(t.Text, width-6, th.Sym.Ellipsis)
		body := icon + " " + text
		if t.Detail != "" {
			body += th.Subtle.Render("  [e]")
		}
		lines = append(lines, style.Render(Pad(body, width-4)))
	}
	return lipgloss.JoinVertical(lipgloss.Right, lines...)
}

func toastStyle(th *theme.Theme, kind ToastKind) (lipgloss.Style, string) {
	switch kind {
	case ToastError:
		return th.ToastError, th.Sym.Error
	case ToastWarning:
		return th.ToastWarning, th.Sym.Warning
	case ToastInfo:
		return th.ToastInfo, th.Sym.Info
	default:
		return th.ToastSuccess, th.Sym.Success
	}
}

// Overlay draws top over base at the given offset, without disturbing the
// lines it does not cover. Terminals do not support real transparency, so
// overlays are composed by splicing cells rather than by blending.
func Overlay(base, top string, x, y int) string {
	baseLines := Lines(base)
	topLines := Lines(top)

	for i, tl := range topLines {
		row := y + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		baseLines[row] = spliceLine(baseLines[row], tl, x)
	}
	return strings.Join(baseLines, "\n")
}

// spliceLine replaces the cells [x, x+width(top)) of base with top.
func spliceLine(base, top string, x int) string {
	tw := Width(top)
	if tw == 0 {
		return base
	}
	bw := Width(base)
	if x >= bw {
		return base + strings.Repeat(" ", x-bw) + top
	}

	left := ""
	if x > 0 {
		left = Pad(cut(base, 0, x), x)
	}
	right := ""
	if end := x + tw; end < bw {
		right = cut(base, end, bw)
	}
	return left + top + right
}
