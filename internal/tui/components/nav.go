package components

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// NavItem is one destination in the sidebar or the tab bar.
type NavItem struct {
	Label string
	// Short is used in the tab bar and on narrow terminals.
	Short string
	// Count is shown as a trailing badge; negative means "unknown", which
	// renders as nothing rather than as a misleading zero.
	Count int
	// Enabled is false when the instance or token cannot serve this section.
	Enabled bool
	// Reason explains why a disabled item is unavailable.
	Reason string
}

// Sidebar renders the vertical navigation used in the wide layout.
func Sidebar(th *theme.Theme, items []NavItem, active, width, height int, focused bool) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	inner := width - 1 // one column reserved for the divider

	lines := make([]string, 0, height)
	lines = append(lines, th.NavTitle.Render(Pad("RESOURCES", inner-2)))
	lines = append(lines, "")

	for i, it := range items {
		label := it.Label
		badge := ""
		if it.Count >= 0 {
			badge = strconv.Itoa(it.Count)
		}
		if !it.Enabled {
			badge = th.Sym.Lock
		}

		text := Fit(label, inner-4-Width(badge), th.Sym.Ellipsis)
		row := " " + text + " " + badge + " "

		switch {
		case i == active && focused:
			lines = append(lines, th.NavItemActive.Render(Pad(row, inner-2)))
		case i == active:
			lines = append(lines, th.NavItemBlurred.Render(Pad(row, inner-2)))
		case !it.Enabled:
			lines = append(lines, th.Subtle.Render(Pad(row, inner-2)))
		default:
			lines = append(lines, th.NavItem.Render(Pad(row, inner-2)))
		}
	}

	body := FitBlock(strings.Join(lines, "\n"), inner, height)

	// A hairline divider rather than a full border: it separates the panels
	// without spending two columns and a boxed-in look.
	divider := strings.Join(repeat(th.HeaderRule.Render("│"), height), "\n")
	return lipgloss.JoinHorizontal(lipgloss.Top, body, divider)
}

// Tabs renders the horizontal navigation used in the standard and compact
// layouts, where a sidebar would cost too much width.
func Tabs(th *theme.Theme, items []NavItem, active, width int, compact bool) string {
	parts := make([]string, 0, len(items))
	for i, it := range items {
		label := it.Label
		if compact && it.Short != "" {
			label = it.Short
		}
		if it.Count >= 0 && !compact {
			label += " " + th.NavCount.Render(strconv.Itoa(it.Count))
		}
		switch {
		case !it.Enabled:
			parts = append(parts, th.Subtle.Render(label+" "+th.Sym.Lock))
		case i == active:
			parts = append(parts, th.TabActive.Render(label))
		default:
			parts = append(parts, th.TabInactive.Render(label))
		}
	}
	return Pad(" "+strings.Join(parts, th.HeaderRule.Render(" "+th.Sym.Separator+" ")), width)
}

// Breadcrumb renders the "instance / project / app" trail that keeps the
// active context visible on detail screens.
func Breadcrumb(th *theme.Theme, width int, parts ...string) string {
	kept := parts[:0]
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	if len(kept) == 0 {
		return ""
	}

	sep := th.Subtle.Render(" " + th.Sym.ArrowRight + " ")
	rendered := make([]string, 0, len(kept))
	for i, p := range kept {
		if i == len(kept)-1 {
			rendered = append(rendered, th.Strong.Render(p))
			continue
		}
		rendered = append(rendered, th.Muted.Render(p))
	}
	return Fit(strings.Join(rendered, sep), width, th.Sym.Ellipsis)
}

func repeat(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}
