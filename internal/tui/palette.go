package tui

import (
	"sort"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
	"github.com/resetnak/cooldeck/internal/tui/views"
)

// paletteCommand is one centrally registered command palette entry. The
// renderer never hardcodes the catalogue; it only filters and paints this list.
type paletteCommand struct {
	ID          string
	Title       string
	Description string
	Shortcut    string
	Group       string
	// Dangerous commands re-use the confirmation modal after selection.
	Dangerous bool
	// Available reports whether the command can run right now. When false,
	// Reason is shown next to the row so the user knows why.
	Available func(*Model) (ok bool, reason string)
	// Run performs the command. It must not assume the palette is still open.
	Run func(*Model) tea.Cmd
}

// paletteCommands returns every command the palette can offer in the current
// build. Context-dependent availability is resolved per row when filtering.
func paletteCommands() []paletteCommand {
	return []paletteCommand{
		{
			ID:          "deploy",
			Title:       "Deploy selected application",
			Description: "Queue a normal deployment",
			Shortcut:    "d",
			Group:       "Actions",
			Dangerous:   true,
			Available: func(m *Model) (bool, string) {
				if _, ok := m.currentApplication(); !ok {
					return false, "no application selected"
				}
				if !m.capabilities.Deploy {
					return false, "token cannot deploy"
				}
				if m.operationInFlight {
					return false, "another action is running"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd { return m.stageAction(actionDeploy, false) },
		},
		{
			ID:          "force-deploy",
			Title:       "Force deploy selected application",
			Description: "Bypass Coolify's deployment cache",
			Shortcut:    "D",
			Group:       "Actions",
			Dangerous:   true,
			Available: func(m *Model) (bool, string) {
				if _, ok := m.currentApplication(); !ok {
					return false, "no application selected"
				}
				if !m.capabilities.Deploy {
					return false, "token cannot deploy"
				}
				if m.operationInFlight {
					return false, "another action is running"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd { return m.stageAction(actionDeploy, true) },
		},
		{
			ID:          "restart",
			Title:       "Restart application",
			Description: "Restart the selected application",
			Shortcut:    "r",
			Group:       "Actions",
			Dangerous:   true,
			Available: func(m *Model) (bool, string) {
				if _, ok := m.currentApplication(); !ok {
					return false, "no application selected"
				}
				if !m.capabilities.Restart {
					return false, "token cannot restart"
				}
				if m.operationInFlight {
					return false, "another action is running"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd { return m.stageAction(actionRestart, false) },
		},
		{
			ID:          "start-stop",
			Title:       "Start or stop application",
			Description: "Toggle the application between running and stopped",
			Shortcut:    "s",
			Group:       "Actions",
			Dangerous:   true,
			Available: func(m *Model) (bool, string) {
				if _, ok := m.currentApplication(); !ok {
					return false, "no application selected"
				}
				if !m.capabilities.StartStop {
					return false, "token cannot start/stop"
				}
				if m.operationInFlight {
					return false, "another action is running"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd {
				if application, ok := m.currentApplication(); ok && application.Status.IsRunning() {
					return m.stageAction(actionStop, false)
				}
				return m.stageAction(actionStart, false)
			},
		},
		{
			ID:          "runtime-logs",
			Title:       "Open runtime logs",
			Description: "Show live application output",
			Shortcut:    "l",
			Group:       "Navigate",
			Available: func(m *Model) (bool, string) {
				if _, ok := m.currentApplication(); !ok {
					return false, "no application selected"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd {
				if m.screen != screenDetail {
					if a, ok := m.apps.Selected(); ok {
						m.screen = screenDetail
						m.detail.SetApplication(a, m.apps.LoadedAt())
						m.detail.SetTab(views.TabRuntimeLogs)
						return tea.Batch(m.loadDetail(a.UUID), m.loadRuntimeLogs(a.UUID))
					}
					return nil
				}
				m.detail.SetTab(views.TabRuntimeLogs)
				return m.loadActiveDetailTab()
			},
		},
		{
			ID:          "deployments",
			Title:       "Open deployments",
			Description: "Show recent deployments for the selected application",
			Shortcut:    "L",
			Group:       "Navigate",
			Available: func(m *Model) (bool, string) {
				if _, ok := m.currentApplication(); !ok {
					return false, "no application selected"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd {
				if m.screen != screenDetail {
					if a, ok := m.apps.Selected(); ok {
						m.screen = screenDetail
						m.detail.SetApplication(a, m.apps.LoadedAt())
						m.detail.SetTab(views.TabDeployments)
						return m.loadDetail(a.UUID)
					}
					return nil
				}
				m.detail.SetTab(views.TabDeployments)
				return m.loadActiveDetailTab()
			},
		},
		{
			ID:          "open-domain",
			Title:       "Open application domain",
			Description: "Open the primary domain in a browser",
			Shortcut:    "b",
			Group:       "Navigate",
			Available: func(m *Model) (bool, string) {
				a, ok := m.currentApplication()
				if !ok {
					return false, "no application selected"
				}
				if a.PrimaryDomain() == "" {
					return false, "no domain configured"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd { return m.openSelectedDomain() },
		},
		{
			ID:          "open-repo",
			Title:       "Open repository",
			Description: "Open the git repository in a browser",
			Shortcut:    "o",
			Group:       "Navigate",
			Available: func(m *Model) (bool, string) {
				a, ok := m.currentApplication()
				if !ok {
					return false, "no application selected"
				}
				if a.RepositoryURL == "" {
					return false, "no repository configured"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd { return m.openSelectedRepository() },
		},
		{
			ID:          "copy-uuid",
			Title:       "Copy application UUID",
			Description: "Copy the selected application's UUID to the clipboard",
			Shortcut:    "c",
			Group:       "Clipboard",
			Available: func(m *Model) (bool, string) {
				if _, ok := m.currentApplication(); !ok {
					return false, "no application selected"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd { return m.copyApplicationUUID() },
		},
		{
			ID:          "refresh",
			Title:       "Refresh",
			Description: "Reload the dashboard now",
			Shortcut:    "R",
			Group:       "View",
			Available:   func(*Model) (bool, string) { return true, "" },
			Run:         func(m *Model) tea.Cmd { return m.manualRefresh() },
		},
		{
			ID:          "filter",
			Title:       "Filter applications",
			Description: "Filter the applications table",
			Shortcut:    "/",
			Group:       "View",
			Available: func(m *Model) (bool, string) {
				if m.screen != screenList || m.section != SectionApplications {
					return false, "only on the applications list"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd {
				m.filtering = true
				m.filterText = m.apps.Filter().Raw
				return nil
			},
		},
		{
			ID:          "toggle-theme",
			Title:       "Toggle theme",
			Description: "Cycle the colour scheme",
			Shortcut:    "ctrl+t",
			Group:       "View",
			Available:   func(*Model) (bool, string) { return true, "" },
			Run: func(m *Model) tea.Cmd {
				m.cycleTheme()
				return m.pushToast(components.ToastInfo, "Theme: "+string(m.themeMode), "")
			},
		},
		{
			ID:          "toggle-compact",
			Title:       "Toggle compact mode",
			Description: "Force the compact layout on or off",
			Shortcut:    "ctrl+w",
			Group:       "View",
			Available:   func(*Model) (bool, string) { return true, "" },
			Run: func(m *Model) tea.Cmd {
				m.forceCompact = !m.forceCompact
				m.recomputeLayout()
				mode := "off"
				if m.forceCompact {
					mode = "on"
				}
				return m.pushToast(components.ToastInfo, "Compact mode "+mode, "")
			},
		},
		{
			ID:          "help",
			Title:       "Open help",
			Description: "Show the key binding summary",
			Shortcut:    "?",
			Group:       "View",
			Available:   func(*Model) (bool, string) { return true, "" },
			Run: func(m *Model) tea.Cmd {
				m.openHelp()
				return nil
			},
		},
		{
			ID:          "switch-instance",
			Title:       "Switch instance",
			Description: "Open the instances list and switch with enter",
			Shortcut:    "3",
			Group:       "Navigate",
			Available:   func(*Model) (bool, string) { return true, "" },
			Run: func(m *Model) tea.Cmd {
				_, cmd := m.gotoSection(SectionInstances)
				return cmd
			},
		},
		{
			ID:          "add-instance",
			Title:       "Add instance",
			Description: "Add a Coolify instance and store its token in the keyring",
			Shortcut:    "a",
			Group:       "Instances",
			Available: func(m *Model) (bool, string) {
				if m.opts.Demo {
					return false, "not available in demo mode"
				}
				if m.opts.SaveConfig == nil {
					return false, "config writing unavailable"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd {
				_, _ = m.gotoSection(SectionInstances)
				return m.openAddInstanceForm()
			},
		},
		{
			ID:          "delete-instance",
			Title:       "Delete local instance",
			Description: "Remove the selected instance from local config",
			Shortcut:    "d",
			Group:       "Instances",
			Dangerous:   true,
			Available: func(m *Model) (bool, string) {
				if m.opts.Demo {
					return false, "not available in demo mode"
				}
				if m.opts.SaveConfig == nil {
					return false, "config writing unavailable"
				}
				if m.section != SectionInstances {
					return false, "open the instances section first"
				}
				if _, ok := m.instances.Selected(); !ok {
					return false, "no instance selected"
				}
				return true, ""
			},
			Run: func(m *Model) tea.Cmd {
				row, ok := m.instances.Selected()
				if !ok {
					return nil
				}
				return m.stageDeleteInstance(row.ID, row.Name)
			},
		},
		{
			ID:          "open-deployments",
			Title:       "Open deployments overview",
			Description: "Show active deployments across applications",
			Shortcut:    "2",
			Group:       "Navigate",
			Available:   func(*Model) (bool, string) { return true, "" },
			Run: func(m *Model) tea.Cmd {
				_, cmd := m.gotoSection(SectionDeployments)
				return cmd
			},
		},
		{
			ID:          "open-diagnostics",
			Title:       "Open diagnostics",
			Description: "Show environment and connection diagnostics",
			Shortcut:    "4",
			Group:       "Navigate",
			Available:   func(*Model) (bool, string) { return true, "" },
			Run: func(m *Model) tea.Cmd {
				_, cmd := m.gotoSection(SectionDiagnostics)
				return cmd
			},
		},
	}
}

type rankedCommand struct {
	cmd    paletteCommand
	score  int
	ok     bool
	reason string
}

// filteredPaletteCommands ranks commands against the query. Empty query keeps
// the catalogue order; a query uses subsequence fuzzy matching.
func (m *Model) filteredPaletteCommands() []rankedCommand {
	all := paletteCommands()
	query := strings.TrimSpace(m.paletteQuery)
	out := make([]rankedCommand, 0, len(all))
	for _, cmd := range all {
		ok, reason := true, ""
		if cmd.Available != nil {
			ok, reason = cmd.Available(m)
		}
		if query == "" {
			out = append(out, rankedCommand{cmd: cmd, score: 0, ok: ok, reason: reason})
			continue
		}
		haystack := cmd.Title + " " + cmd.Description + " " + cmd.Group + " " + cmd.Shortcut
		score, match := fuzzyScore(query, haystack)
		if !match {
			continue
		}
		out = append(out, rankedCommand{cmd: cmd, score: score, ok: ok, reason: reason})
	}
	if query != "" {
		sort.SliceStable(out, func(i, j int) bool {
			if out[i].score != out[j].score {
				return out[i].score > out[j].score
			}
			return out[i].cmd.Title < out[j].cmd.Title
		})
	}
	return out
}

// openPalette opens the command palette and resets its editor state.
func (m *Model) openPalette() {
	m.paletteOpen = true
	m.paletteQuery = ""
	m.paletteSelected = 0
	m.filtering = false
	m.logSearching = false
}

// closePalette dismisses the palette without running a command.
func (m *Model) closePalette() {
	m.paletteOpen = false
	m.paletteQuery = ""
	m.paletteSelected = 0
}

// handlePaletteKey drives the command palette modal. Arrows and ctrl+n/p move
// the selection; j/k type into the filter so "deploy" can be entered normally.
func (m *Model) handlePaletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	items := m.filteredPaletteCommands()

	switch msg.String() {
	case "esc":
		m.closePalette()
		return m, nil
	case "up", "ctrl+p":
		if m.paletteSelected > 0 {
			m.paletteSelected--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.paletteSelected < len(items)-1 {
			m.paletteSelected++
		}
		return m, nil
	case "enter":
		if len(items) == 0 {
			return m, nil
		}
		if m.paletteSelected < 0 || m.paletteSelected >= len(items) {
			m.paletteSelected = 0
		}
		item := items[m.paletteSelected]
		if !item.ok {
			return m, m.pushToast(components.ToastWarning, "Unavailable", item.reason)
		}
		m.closePalette()
		if item.cmd.Run == nil {
			return m, nil
		}
		return m, item.cmd.Run(m)
	case "backspace":
		if n := len(m.paletteQuery); n > 0 {
			runes := []rune(m.paletteQuery)
			m.paletteQuery = string(runes[:len(runes)-1])
			m.paletteSelected = 0
		}
		return m, nil
	case "ctrl+u":
		m.paletteQuery = ""
		m.paletteSelected = 0
		return m, nil
	}

	// String() renders space as "space"; Text carries the literal input,
	// and is empty for keys that produce none.
	if text := msg.Key().Text; text != "" {
		m.paletteQuery += text
		m.paletteSelected = 0
	}
	return m, nil
}

// renderPalette draws the command palette modal over the frame.
func (m *Model) renderPalette() string {
	if !m.paletteOpen || m.layout.Width < 28 || m.layout.Height < 10 {
		return ""
	}
	th := m.theme
	width := min(m.layout.Width-6, 64)
	items := m.filteredPaletteCommands()
	if m.paletteSelected >= len(items) {
		m.paletteSelected = max(len(items)-1, 0)
	}

	maxRows := min(10, max(m.layout.Height-10, 4))
	prompt := th.PaletteInput.Render("> " + m.paletteQuery + "▏")

	rows := []string{
		th.ModalTitle.Render("Commands"),
		prompt,
		th.ModalHint.Render("type to filter · enter run · esc close"),
		"",
	}

	if len(items) == 0 {
		rows = append(rows, th.PaletteDisabled.Render("No matching commands"))
	} else {
		start := 0
		if m.paletteSelected >= maxRows {
			start = m.paletteSelected - maxRows + 1
		}
		end := min(start+maxRows, len(items))
		var lastGroup string
		for i := start; i < end; i++ {
			item := items[i]
			if item.cmd.Group != "" && item.cmd.Group != lastGroup && m.paletteQuery == "" {
				rows = append(rows, th.PaletteGroup.Render(item.cmd.Group))
				lastGroup = item.cmd.Group
			}
			rows = append(rows, m.renderPaletteRow(item, i == m.paletteSelected, width-4))
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return theme.Fill(th.Modal.Width(width), content)
}

func (m *Model) renderPaletteRow(item rankedCommand, active bool, width int) string {
	th := m.theme
	title := item.cmd.Title
	shortcut := item.cmd.Shortcut
	reason := item.reason

	// Leave room for the shortcut column on the right.
	shortcutW := max(components.Width(shortcut)+1, 4)
	titleW := max(width-shortcutW, 8)

	var left string
	if !item.ok {
		label := title
		if reason != "" {
			label = title + "  " + reason
		}
		left = th.PaletteDisabled.Render(components.Truncate(label, titleW, th.Sym.Ellipsis))
		right := th.PaletteDisabled.Render(components.PadLeft(shortcut, shortcutW))
		return left + right
	}

	if active {
		left = th.PaletteActive.Render(components.Pad(components.Truncate(title, titleW, th.Sym.Ellipsis), titleW))
		right := th.PaletteActive.Render(components.PadLeft(shortcut, shortcutW))
		return left + right
	}
	left = th.PaletteItem.Render(components.Truncate(title, titleW, th.Sym.Ellipsis))
	right := th.PaletteShortcut.Render(components.PadLeft(shortcut, shortcutW))
	return components.Pad(left, titleW) + right
}

// fuzzyScore ranks a candidate against a query using case-insensitive
// subsequence matching. Higher scores are better; ok is false when the query
// is not a subsequence of the candidate.
func fuzzyScore(query, candidate string) (score int, ok bool) {
	q := []rune(strings.ToLower(query))
	c := []rune(strings.ToLower(candidate))
	if len(q) == 0 {
		return 0, true
	}
	if len(q) > len(c) {
		return 0, false
	}

	qi := 0
	score = 0
	prevMatch := -2
	for ci, r := range c {
		if qi >= len(q) {
			break
		}
		if r != q[qi] {
			continue
		}
		// Prefer contiguous runs and earlier matches.
		bonus := 1
		if ci == prevMatch+1 {
			bonus += 4
		}
		if ci == 0 || !unicode.IsLetter(c[ci-1]) {
			bonus += 3
		}
		score += bonus + max(0, 32-ci)
		prevMatch = ci
		qi++
	}
	if qi < len(q) {
		return 0, false
	}
	// Prefer shorter candidates when the match is otherwise equal.
	score += max(0, 64-len(c))
	return score, true
}
