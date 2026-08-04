package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/version"
)

// helpGroup is one titled block of key bindings on the help screen.
type helpGroup struct {
	Title string
	Rows  []helpRow
}

type helpRow struct {
	Keys string
	Desc string
}

// openHelp opens the full help overlay.
func (m *Model) openHelp() {
	m.helpOpen = true
	m.helpScroll = 0
	m.paletteOpen = false
	m.filtering = false
	m.logSearching = false
}

// closeHelp dismisses the help overlay.
func (m *Model) closeHelp() {
	m.helpOpen = false
	m.helpScroll = 0
}

// handleHelpKey drives scrolling inside the help overlay.
func (m *Model) handleHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?":
		m.closeHelp()
		return m, nil
	case "up", "k":
		m.helpScroll = max(m.helpScroll-1, 0)
	case "down", "j":
		m.helpScroll++
	case "pgup", "ctrl+u":
		m.helpScroll = max(m.helpScroll-8, 0)
	case "pgdown", "ctrl+d":
		m.helpScroll += 8
	case "g", "home":
		m.helpScroll = 0
	}
	return m, nil
}

func (m *Model) helpGroups() []helpGroup {
	return []helpGroup{
		{
			Title: "Global",
			Rows: []helpRow{
				{"↑↓ / j k", "move selection"},
				{"g / G", "first / last"},
				{"ctrl+d / ctrl+u", "page down / up"},
				{"enter", "open / confirm"},
				{"esc / q", "back / quit"},
				{"tab / shift+tab", "next / previous pane"},
				{"/ ", "filter or search"},
				{": / ctrl+k", "command palette"},
				{"?", "this help"},
				{"R", "manual refresh"},
				{"ctrl+t", "cycle theme"},
				{"ctrl+w", "toggle compact mode"},
				{"ctrl+c", "force quit"},
			},
		},
		{
			Title: "Sections",
			Rows: []helpRow{
				{"1", "applications"},
				{"2", "deployments"},
				{"3", "instances"},
				{"4", "diagnostics"},
			},
		},
		{
			Title: "Applications",
			Rows: []helpRow{
				{"d / D", "deploy / force deploy"},
				{"r", "restart"},
				{"s", "start or stop"},
				{"l / L", "runtime logs / build log"},
				{"b / o", "open domain / repository"},
				{"c", "copy UUID"},
				{"S", "cycle sort mode"},
			},
		},
		{
			Title: "Runtime logs",
			Rows: []helpRow{
				{"space", "pause / resume polling"},
				{"f", "follow tail"},
				{"w", "wrap long lines"},
				{"/ n N", "search / next / previous match"},
				{"c", "copy buffer or match"},
				{"ctrl+l", "clear local buffer"},
				{"+ / -", "more / fewer lines"},
			},
		},
		{
			Title: "Deployments",
			Rows: []helpRow{
				{"enter", "open build log for deployment"},
				{"a", "toggle active-only / recent history"},
				{"c", "copy deployment UUID"},
			},
		},
		{
			Title: "Instances",
			Rows: []helpRow{
				{"enter", "switch to selected instance"},
				{"T", "test active connection"},
				{"a", "add instance (keyring token)"},
				{"e", "edit name and URL"},
				{"d", "delete local instance config"},
			},
		},
		{
			Title: "Diagnostics",
			Rows: []helpRow{
				{"c", "copy diagnostics"},
				{"e", "export diagnostics to file"},
			},
		},
	}
}

// renderHelp builds the help modal body.
func (m *Model) renderHelp() string {
	if !m.helpOpen || m.layout.Width < 28 || m.layout.Height < 10 {
		return ""
	}
	th := m.theme
	width := min(m.layout.Width-4, 72)
	inner := max(width-4, 20)

	var lines []string
	lines = append(lines, th.ModalTitle.Render(version.AppName+" help"))
	lines = append(lines, th.ModalHint.Render(version.Tagline))
	lines = append(lines, th.ModalHint.Render("esc close  ·  ↑↓ scroll"))
	lines = append(lines, "")

	for _, group := range m.helpGroups() {
		lines = append(lines, th.PaletteGroup.Render(group.Title))
		for _, row := range group.Rows {
			keyCol := th.PaletteShortcut.Render(components.Pad(row.Keys, 18))
			desc := th.ModalBody.Render(components.Truncate(row.Desc, max(inner-20, 8), th.Sym.Ellipsis))
			lines = append(lines, keyCol+"  "+desc)
		}
		lines = append(lines, "")
	}

	// Drop trailing blank.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	maxBody := max(m.layout.Height-6, 6)
	maxScroll := max(len(lines)-maxBody, 0)
	m.helpScroll = min(m.helpScroll, maxScroll)
	visible := lines[m.helpScroll:]
	if len(visible) > maxBody {
		visible = visible[:maxBody]
	}

	content := lipgloss.JoinVertical(lipgloss.Left, visible...)
	return th.Modal.Width(width).Render(content)
}

// overlayHelp places the help modal over the frame.
func (m *Model) overlayHelp(frame string) string {
	modal := m.renderHelp()
	if modal == "" {
		return frame
	}
	// Dim the underlying frame by restyling it with the backdrop colour.
	dimmed := m.theme.Backdrop.Render(strings.Join(components.Lines(frame), "\n"))
	x := max((m.layout.Width-lipgloss.Width(modal))/2, 0)
	y := max((m.layout.Height-lipgloss.Height(modal))/6, 1)
	return components.Overlay(dimmed, modal, x, y)
}
