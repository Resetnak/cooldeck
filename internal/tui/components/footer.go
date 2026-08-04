package components

import (
	"strings"

	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// KeyHint is one "key description" pair in the footer or help screen.
type KeyHint struct {
	Key  string
	Desc string
	// Short is an abbreviated description for narrow terminals.
	Short string
	// Disabled greys the hint out; the key exists but is unavailable here.
	Disabled bool
}

// Footer renders the contextual key hints. On a narrow terminal it shows only
// what fits and appends "? more", rather than wrapping onto a second line and
// stealing a row from the content.
func Footer(th *theme.Theme, l theme.Layout, hints []KeyHint, status string) string {
	if !l.ShowFooter {
		return ""
	}

	width := l.Width - 2*theme.SpaceXS
	sep := th.FooterSep.Render("  ")

	statusWidth := 0
	if status != "" {
		statusWidth = Width(status) + 3
	}

	var parts []string
	used := 0
	overflow := false
	for _, h := range hints {
		rendered := renderHint(th, l, h)
		w := Width(rendered)
		if len(parts) > 0 {
			w += Width("  ")
		}
		if used+w+statusWidth > width {
			overflow = true
			break
		}
		parts = append(parts, rendered)
		used += w
	}

	if overflow {
		more := renderHint(th, l, KeyHint{Key: "?", Desc: "more"})
		// Drop trailing hints until the "more" marker fits.
		for len(parts) > 0 && used+Width(more)+Width("  ")+statusWidth > width {
			used -= Width(parts[len(parts)-1]) + Width("  ")
			parts = parts[:len(parts)-1]
		}
		parts = append(parts, more)
	}

	line := strings.Join(parts, sep)
	if status != "" {
		gap := max(width-Width(line)-Width(status), 1)
		line += strings.Repeat(" ", gap) + status
	}
	bar := Pad(th.FooterBar.Render(line), l.Width)
	rule := th.HeaderRule.Render(strings.Repeat("─", l.Width))
	return rule + "\n" + bar
}

func renderHint(th *theme.Theme, l theme.Layout, h KeyHint) string {
	desc := h.Desc
	if l.IsCompact() && h.Short != "" {
		desc = h.Short
	}
	if h.Disabled {
		return th.Subtle.Render(h.Key + " " + desc)
	}
	return th.FooterKey.Render(h.Key) + th.FooterDesc.Render(" "+desc)
}
