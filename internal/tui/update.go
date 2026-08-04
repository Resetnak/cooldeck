package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

// activeInstance resolves the config entry for the running instance, falling
// back to an empty one in demo mode where there is no configured instance.
func activeInstance(m *Model) config.Instance {
	inst, err := m.opts.Config.Instance(m.service.InstanceID())
	if err != nil {
		return config.Instance{}
	}
	return inst
}

// handleKey routes a keypress. The order matters: modal-like states (filtering)
// consume input first, then global bindings, then the active screen's.
func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Ctrl+C always quits, even mid-request, and cancels in-flight work first.
	if key.Matches(msg, m.keys.ForceQuit) {
		m.quitting = true
		m.shutdown()
		return m, tea.Quit
	}

	if m.filtering {
		return m.handleFilterKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Help):
		return m, m.pushToast(components.ToastInfo,
			"Keys: j/k move, enter detail, / filter, R refresh, q back", "")

	case key.Matches(msg, m.keys.Theme):
		m.cycleTheme()
		return m, m.pushToast(components.ToastInfo, "Theme: "+string(m.themeMode), "")

	case key.Matches(msg, m.keys.Compact):
		m.forceCompact = !m.forceCompact
		m.recomputeLayout()
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, m.manualRefresh()

	case key.Matches(msg, m.keys.NextPane):
		m.cycleFocus(1)
		return m, nil

	case key.Matches(msg, m.keys.PrevPane):
		m.cycleFocus(-1)
		return m, nil

	case key.Matches(msg, m.keys.SectionApplications):
		return m.gotoSection(SectionApplications)
	}

	if m.focus == focusSidebar {
		return m.handleSidebarKey(msg)
	}

	switch m.screen {
	case screenDetail:
		return m.handleDetailKey(msg)
	default:
		return m.handleListKey(msg)
	}
}

// handleFilterKey runs the inline filter editor. Filtering is applied on every
// keystroke so the result is visible as the query is typed.
func (m *Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		// Esc abandons the query entirely, restoring the unfiltered list.
		m.filtering = false
		m.filterText = ""
		m.apps.SetFilter("")
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		// Enter keeps the filter and returns focus to the table.
		m.filtering = false
		return m, nil
	}

	switch msg.String() {
	case "backspace":
		if n := len(m.filterText); n > 0 {
			// Trim a whole rune, not a byte, or a multi-byte character breaks.
			r := []rune(m.filterText)
			m.filterText = string(r[:len(r)-1])
			m.apps.SetFilter(m.filterText)
		}
		return m, nil
	case "ctrl+u":
		m.filterText = ""
		m.apps.SetFilter("")
		return m, nil
	case "ctrl+w":
		m.filterText = trimLastWord(m.filterText)
		m.apps.SetFilter(m.filterText)
		return m, nil
	}

	if text := msg.String(); len([]rune(text)) == 1 {
		m.filterText += text
		m.apps.SetFilter(m.filterText)
	}
	return m, nil
}

// handleSidebarKey moves between sections when the sidebar has focus.
func (m *Model) handleSidebarKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		return m.gotoSection(clampSection(m.section - 1))
	case key.Matches(msg, m.keys.Down):
		return m.gotoSection(clampSection(m.section + 1))
	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.Right):
		m.focus = focusContent
		return m, nil
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
		m.focus = focusContent
		return m, nil
	}
	return m, nil
}

// handleListKey drives the applications table.
func (m *Model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.section != SectionApplications {
		// The remaining sections land in Milestone 8; until then their keys
		// fall through to the globals rather than pretending to work.
		switch {
		case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Back):
			return m.gotoSection(SectionApplications)
		}
		return m, nil
	}

	page := max(m.contentHeight()-2, 1)

	switch {
	case key.Matches(msg, m.keys.Up):
		m.apps.Move(-1)
	case key.Matches(msg, m.keys.Down):
		m.apps.Move(1)
	case key.Matches(msg, m.keys.Top):
		m.apps.MoveTo(0)
	case key.Matches(msg, m.keys.Bottom):
		m.apps.MoveTo(m.apps.Count() - 1)
	case key.Matches(msg, m.keys.PageDown):
		m.apps.Move(page)
	case key.Matches(msg, m.keys.PageUp):
		m.apps.Move(-page)

	case key.Matches(msg, m.keys.Left):
		if m.layout.ShowSidebar {
			m.focus = focusSidebar
		}

	case key.Matches(msg, m.keys.Filter):
		m.filtering = true
		m.filterText = m.apps.Filter().Raw

	case key.Matches(msg, m.keys.Sort):
		next := m.apps.SortMode().Next()
		m.apps.SetSort(next)
		return m, m.pushToast(components.ToastInfo, "Sorted by "+next.Label(), "")

	case key.Matches(msg, m.keys.Enter):
		a, ok := m.apps.Selected()
		if !ok {
			return m, nil
		}
		m.screen = screenDetail
		m.detail.SetApplication(a, m.apps.LoadedAt())
		return m, m.loadDetail(a.UUID)

	case key.Matches(msg, m.keys.OpenBrowser):
		return m, m.openSelectedDomain()

	case key.Matches(msg, m.keys.OpenRepo):
		return m, m.openSelectedRepository()

	case key.Matches(msg, m.keys.Back):
		// Esc clears an active filter before it does anything else, which is
		// what the empty state tells the user it will do.
		if !m.apps.Filter().IsEmpty() {
			m.filterText = ""
			m.apps.SetFilter("")
			return m, nil
		}

	case key.Matches(msg, m.keys.Quit):
		if !m.apps.Filter().IsEmpty() {
			m.filterText = ""
			m.apps.SetFilter("")
			return m, nil
		}
		m.quitting = true
		m.shutdown()
		return m, tea.Quit
	}
	return m, nil
}

// handleDetailKey drives the application detail screen.
func (m *Model) handleDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
		m.screen = screenList
		return m, nil

	case key.Matches(msg, m.keys.Right):
		m.detail.NextTab()
	case key.Matches(msg, m.keys.Left):
		m.detail.PrevTab()

	case key.Matches(msg, m.keys.Down):
		m.detail.Scroll(1)
	case key.Matches(msg, m.keys.Up):
		m.detail.Scroll(-1)
	case key.Matches(msg, m.keys.PageDown):
		m.detail.Scroll(max(m.contentHeight()-2, 1))
	case key.Matches(msg, m.keys.PageUp):
		m.detail.Scroll(-max(m.contentHeight()-2, 1))
	case key.Matches(msg, m.keys.Top):
		m.detail.Scroll(-1 << 20)

	case key.Matches(msg, m.keys.OpenBrowser):
		return m, m.openSelectedDomain()
	case key.Matches(msg, m.keys.OpenRepo):
		return m, m.openSelectedRepository()
	}
	return m, nil
}

// openSelectedDomain opens the primary domain of the current application.
func (m *Model) openSelectedDomain() tea.Cmd {
	a, ok := m.currentApplication()
	if !ok {
		return nil
	}
	url := domain.DomainURL(a.PrimaryDomain())
	if url == "" {
		return m.pushToast(components.ToastWarning, "No domain to open", "")
	}
	return m.openURL(url)
}

// openSelectedRepository opens the git repository of the current application.
func (m *Model) openSelectedRepository() tea.Cmd {
	a, ok := m.currentApplication()
	if !ok {
		return nil
	}
	url := domain.WebURLForRepository(a.RepositoryURL)
	if url == "" {
		return m.pushToast(components.ToastWarning, "No browsable repository URL", a.RepositoryURL)
	}
	return m.openURL(url)
}

// currentApplication returns whichever application the user is acting on,
// whether they are on the list or in the detail screen.
func (m *Model) currentApplication() (domain.Application, bool) {
	if m.screen == screenDetail && m.detail.Loaded() {
		return m.detail.Application(), true
	}
	return m.apps.Selected()
}

func (m *Model) gotoSection(s Section) (tea.Model, tea.Cmd) {
	if s == m.section && m.screen == screenList {
		return m, nil
	}
	m.section = s
	m.screen = screenList
	m.focus = focusContent
	return m, nil
}

func (m *Model) cycleFocus(delta int) {
	if !m.layout.ShowSidebar {
		m.focus = focusContent
		return
	}
	if delta > 0 {
		m.focus = focusTarget((int(m.focus) + 1) % 2)
		return
	}
	m.focus = focusTarget((int(m.focus) + 1) % 2)
}

// cycleTheme steps through auto, dark and light.
func (m *Model) cycleTheme() {
	switch m.themeMode {
	case config.ThemeAuto:
		m.themeMode = config.ThemeDark
	case config.ThemeDark:
		m.themeMode = config.ThemeLight
	default:
		m.themeMode = config.ThemeAuto
	}
	m.rebuildTheme()
}

// contentHeight is the number of rows available to the active view.
func (m *Model) contentHeight() int {
	h := m.layout.ContentHeight
	if m.filtering {
		h--
	}
	if m.staleFor() > 0 {
		h--
	}
	return max(h, 1)
}

func clampSection(s Section) Section {
	return SectionApplications
}

func trimLastWord(s string) string {
	s = strings.TrimRight(s, " ")
	if i := strings.LastIndexByte(s, ' '); i >= 0 {
		return s[:i+1]
	}
	return ""
}
