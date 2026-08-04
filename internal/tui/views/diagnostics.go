package views

import (
	"fmt"
	"strings"

	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// DiagnosticField is one label/value pair on the diagnostics screen.
type DiagnosticField struct {
	Label string
	Value string
	// Section starts a new group heading when set.
	Section string
}

// Diagnostics is a scrollable read-only dump of environment and connection
// facts. It never includes tokens or other secrets.
type Diagnostics struct {
	fields []DiagnosticField
	scroll int
}

// NewDiagnostics returns an empty diagnostics view.
func NewDiagnostics() *Diagnostics {
	return &Diagnostics{}
}

// SetFields replaces the displayed fields and clamps scroll.
func (v *Diagnostics) SetFields(fields []DiagnosticField) {
	v.fields = append([]DiagnosticField{}, fields...)
	v.scroll = min(v.scroll, max(len(v.renderLines(nil, 80))-1, 0))
}

// Fields returns a copy of the current fields.
func (v *Diagnostics) Fields() []DiagnosticField {
	return append([]DiagnosticField{}, v.fields...)
}

// Scroll moves the body by delta lines.
func (v *Diagnostics) Scroll(delta, viewport int) {
	lines := len(v.renderLines(nil, 80))
	maxScroll := max(lines-viewport, 0)
	v.scroll = min(max(v.scroll+delta, 0), maxScroll)
}

// ScrollOffset returns the current scroll position.
func (v *Diagnostics) ScrollOffset() int { return v.scroll }

// PlainText renders the diagnostics as plain text suitable for export/copy.
func (v *Diagnostics) PlainText() string {
	var b strings.Builder
	var section string
	for _, f := range v.fields {
		if f.Section != "" && f.Section != section {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString("## ")
			b.WriteString(f.Section)
			b.WriteByte('\n')
			section = f.Section
		}
		if f.Label == "" {
			continue
		}
		b.WriteString(f.Label)
		b.WriteString(": ")
		b.WriteString(f.Value)
		b.WriteByte('\n')
	}
	return b.String()
}

// Render draws the diagnostics panel.
func (v *Diagnostics) Render(th *theme.Theme, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(v.fields) == 0 {
		return components.EmptyState(th, width, height, "Diagnostics unavailable", "No diagnostic data has been collected yet.", nil)
	}

	header := th.Title.Render("DIAGNOSTICS")
	hint := th.Subtle.Render("c copy  ·  e export  ·  ↑↓ scroll")
	bodyLines := v.renderLines(th, width)
	maxScroll := max(len(bodyLines)-max(height-3, 1), 0)
	v.scroll = min(v.scroll, maxScroll)
	visible := bodyLines[v.scroll:]
	body := strings.Join(visible, "\n")
	content := header + "\n" + hint + "\n\n" + body
	return components.FitBlock(content, width, height)
}

func (v *Diagnostics) renderLines(th *theme.Theme, width int) []string {
	labelW := 18
	for _, f := range v.fields {
		if w := components.Width(f.Label); w > labelW && w < 28 {
			labelW = w
		}
	}
	valueW := max(width-labelW-2, 8)

	var lines []string
	var section string
	for _, f := range v.fields {
		if f.Section != "" && f.Section != section {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			title := f.Section
			if th != nil {
				title = th.Title.Render(strings.ToUpper(f.Section))
			}
			lines = append(lines, title)
			section = f.Section
		}
		if f.Label == "" {
			continue
		}
		if th != nil {
			label := th.Subtle.Render(components.Pad(f.Label, labelW))
			value := th.Value.Render(components.Truncate(f.Value, valueW, th.Sym.Ellipsis))
			lines = append(lines, label+"  "+value)
			continue
		}
		lines = append(lines, fmt.Sprintf("%-*s  %s", labelW, f.Label, f.Value))
	}
	return lines
}
