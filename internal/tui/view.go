package tui

import (
	"strconv"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/views"
)

// render composes one frame. It performs no I/O and no data processing: every
// value it needs was computed when the data or the layout last changed.
func (m *Model) render() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 || m.height == 0 {
		// The first frame arrives before the terminal reports its size.
		return ""
	}
	if !m.layout.Usable() {
		return components.TooSmall(m.theme, m.width, m.height)
	}

	header := components.Header(m.theme, m.layout, m.headerData())
	body := m.renderBody()
	footer := components.Footer(m.theme, m.layout, m.footerHints(), m.footerStatus())

	frame := joinRows(header, body, footer)
	frame = m.overlayToasts(frame)
	frame = m.overlayConfirmation(frame)
	return trimTrailingBlank(frame)
}

func (m *Model) headerData() components.HeaderData {
	noun := "apps"
	count := m.apps.Count()
	if !m.apps.Filter().IsEmpty() && m.apps.Total() != count {
		noun = "of " + strconv.Itoa(m.apps.Total()) + " apps"
	}

	return components.HeaderData{
		InstanceName:  m.instanceLabel(),
		Connection:    m.connection,
		Demo:          m.opts.Demo,
		ResourceCount: count,
		ResourceNoun:  noun,
		LastRefresh:   m.lastSuccess,
		Now:           m.now(),
		Refreshing:    m.refreshing,
		Spinner:       m.spinnerFrame(),
		Filter:        m.apps.Filter().Raw,
	}
}

// renderBody draws the navigation, the active view and the optional preview.
// Banner rows are laid out first so the main content can be given exactly the
// height that remains, which is what keeps the footer pinned to the last line.
func (m *Model) renderBody() string {
	var banners []string

	if !m.layout.ShowSidebar {
		banners = append(banners, components.Tabs(
			m.theme, m.sections(), int(m.section), m.layout.Width, m.layout.IsCompact()))
	}
	if m.filtering {
		banners = append(banners, m.filterPrompt(m.layout.Width))
	}
	if m.logSearching {
		banners = append(banners, m.logSearchPrompt(m.layout.Width))
	}
	if stale := m.staleFor(); stale > 0 {
		banners = append(banners, components.Pad(
			components.StaleBanner(m.theme, "", stale), m.layout.Width))
	}

	// ComputeLayout already reserved one row for the tab bar in the non-wide
	// layouts, so only the extra banners are subtracted here.
	extra := len(banners)
	if !m.layout.ShowSidebar {
		extra--
	}
	contentHeight := max(m.layout.ContentHeight-extra, 1)

	main := m.renderMain(m.layout.ContentWidth, contentHeight)

	var content string
	if m.layout.ShowSidebar {
		parts := []string{
			components.Sidebar(m.theme, m.sections(), int(m.section),
				m.layout.SidebarWidth, contentHeight, m.focus == focusSidebar),
			components.FitBlock(main, m.layout.ContentWidth, contentHeight),
		}
		if m.layout.ShowPreview {
			parts = append(parts, m.renderPreview(m.layout.PreviewWidth, contentHeight))
		}
		content = lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	} else {
		content = components.FitBlock(main, m.layout.ContentWidth, contentHeight)
	}

	return lipgloss.JoinVertical(lipgloss.Left, append(banners, content)...)
}

// renderMain draws the active screen.
func (m *Model) renderMain(width, height int) string {
	th := m.theme
	now := m.now()

	// A hard failure with no cached data takes over the content area. With
	// cached data on screen the error becomes a toast plus the stale banner,
	// because wiping readable data to show an error helps nobody.
	if m.lastError != nil && !m.apps.Loaded() {
		return components.ErrorState(th, width, height, m.lastError, []components.KeyHint{
			{Key: "R", Desc: "retry now"},
			{Key: "3", Desc: "switch instance"},
			{Key: "q", Desc: "quit"},
		})
	}

	if m.screen == screenDetail {
		return m.renderDetail(width, height, now)
	}
	return m.apps.Render(th, width, height, m.focus == focusContent, now)
}

func (m *Model) renderDetail(width, height int, now time.Time) string {
	th := m.theme
	a := m.detail.Application()

	crumb := components.Breadcrumb(th, width,
		m.instanceLabel(), a.Project.String(), a.Environment.String(), a.Name)
	tabs := m.detail.TabBar(th, width, m.layout.IsCompact())

	bodyHeight := max(height-3, 1)
	body := m.detail.Render(th, width, bodyHeight, now)

	return lipgloss.JoinVertical(lipgloss.Left,
		components.Pad(crumb, width), tabs, "", body)
}

// renderPreview draws the side panel next to the table on wide terminals.
func (m *Model) renderPreview(width, height int) string {
	a, ok := m.apps.Selected()
	if !ok || m.screen == screenDetail {
		return components.FitBlock("", width, height)
	}
	return components.FitBlock(
		views.Preview(m.theme, a, width, height, m.now()), width, height)
}

// overlayToasts splices the notification stack over the frame, above the
// footer, so it never hides the keys needed to react to it.
func (m *Model) overlayToasts(frame string) string {
	if len(m.toasts) == 0 {
		return frame
	}
	block := components.RenderToasts(m.theme, m.toasts, min(m.layout.Width-4, 52))
	if block == "" {
		return frame
	}

	lines := components.Lines(block)
	blockWidth := 0
	for _, l := range lines {
		blockWidth = max(blockWidth, components.Width(l))
	}

	x := max(m.layout.Width-blockWidth-2, 0)
	y := max(len(components.Lines(frame))-m.layout.FooterHeight-len(lines)-1, 0)
	return components.Overlay(frame, block, x, y)
}

func (m *Model) overlayConfirmation(frame string) string {
	if m.pendingAction == nil || m.layout.Width < 24 || m.layout.Height < 8 {
		return frame
	}
	action := m.pendingAction
	width := min(m.layout.Width-6, 54)
	name := domain.SanitizeLogText(action.app.Name)
	body := "Application: " + components.Truncate(name, width-13, m.theme.Sym.Ellipsis)
	if action.force {
		body += "\nThis bypasses Coolify's normal deployment cache."
	}
	buttons := m.theme.ButtonDanger.Render("enter confirm") + "  " + m.theme.ButtonGhost.Render("esc cancel")
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.theme.ModalTitle.Render(action.title()),
		"",
		m.theme.ModalBody.Render(body),
		"",
		buttons,
	)
	modal := m.theme.ModalDanger.Width(width).Render(content)
	x := max((m.layout.Width-lipgloss.Width(modal))/2, 0)
	y := max((m.layout.Height-lipgloss.Height(modal))/2, 0)
	return components.Overlay(frame, modal, x, y)
}

// footerHints returns the contextual key hints for the active screen.
func (m *Model) footerHints() []components.KeyHint {
	switch {
	case m.filtering:
		return []components.KeyHint{
			{Key: "enter", Desc: "apply"},
			{Key: "esc", Desc: "clear"},
			{Key: "ctrl+u", Desc: "clear line", Short: "clear"},
		}
	case m.logSearching:
		return []components.KeyHint{
			{Key: "enter", Desc: "apply"},
			{Key: "esc", Desc: "close"},
			{Key: "ctrl+u", Desc: "clear", Short: "clear"},
		}

	case m.focus == focusSidebar:
		return []components.KeyHint{
			{Key: "↑↓", Desc: "sections"},
			{Key: "enter", Desc: "open"},
			{Key: "tab", Desc: "back to list", Short: "back"},
			{Key: "?", Desc: "help"},
		}

	case m.screen == screenDetail:
		if m.detail.DeploymentLogsOpen() {
			return []components.KeyHint{
				{Key: "↑↓", Desc: "scroll"},
				{Key: "esc", Desc: "deployments"},
				{Key: "l", Desc: "runtime logs", Short: "runtime"},
			}
		}
		if m.detail.Tab() == views.TabDeployments {
			return []components.KeyHint{
				{Key: "↑↓", Desc: "select"},
				{Key: "enter", Desc: "build log", Short: "log"},
				{Key: "←→", Desc: "tabs"},
				{Key: "esc", Desc: "back"},
			}
		}
		if m.detail.Tab() == views.TabRuntimeLogs {
			pause := "pause"
			if m.detail.RuntimeLogsPaused() {
				pause = "resume"
			}
			return []components.KeyHint{
				{Key: "space", Desc: pause},
				{Key: "f", Desc: "follow"},
				{Key: "w", Desc: "wrap"},
				{Key: "/", Desc: "search"},
				{Key: "↑↓", Desc: "scroll"},
				{Key: "esc", Desc: "back"},
			}
		}
		return []components.KeyHint{
			{Key: "←→", Desc: "tabs"},
			{Key: "↑↓", Desc: "scroll"},
			{Key: "d", Desc: "deploy"},
			{Key: "r", Desc: "restart"},
			{Key: "s", Desc: "start/stop", Short: "state"},
			{Key: "b", Desc: "open domain", Short: "domain"},
			{Key: "o", Desc: "open repo", Short: "repo"},
			{Key: "esc", Desc: "back"},
			{Key: "?", Desc: "help"},
		}

	default:
		return []components.KeyHint{
			{Key: "↑↓", Desc: "navigate", Short: "move"},
			{Key: "enter", Desc: "details", Short: "detail"},
			{Key: "d", Desc: "deploy"},
			{Key: "r", Desc: "restart"},
			{Key: "s", Desc: "start/stop", Short: "state"},
			{Key: "/", Desc: "filter"},
			{Key: "S", Desc: "sort: " + m.apps.SortMode().Label(), Short: "sort"},
			{Key: "b", Desc: "open domain", Short: "domain"},
			{Key: "R", Desc: "refresh"},
			{Key: ":", Desc: "commands", Short: "cmds"},
			{Key: "?", Desc: "help"},
		}
	}
}

// footerStatus is the right-aligned status text: the cursor position within
// the list, or the active operation.
func (m *Model) footerStatus() string {
	th := m.theme
	switch {
	case m.loading:
		return th.Muted.Render(m.spinnerFrame() + " loading")
	case m.operationInFlight:
		return th.Muted.Render(m.spinnerFrame() + " sending action")
	case m.screen == screenDetail:
		return th.Subtle.Render(m.detail.Tab().Label())
	case m.apps.Count() == 0:
		return ""
	default:
		idx := m.apps.SelectedIndex()
		if idx < 0 {
			return ""
		}
		return th.Subtle.Render(strconv.Itoa(idx+1) + "/" + strconv.Itoa(m.apps.Count()))
	}
}
