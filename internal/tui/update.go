package tui

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/platform"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/views"
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
	if m.pendingAction != nil {
		return m.handleConfirmationKey(msg)
	}
	if m.terminalPicker != nil {
		return m.handleTerminalPickerKey(msg)
	}
	if m.instanceForm != nil {
		return m.handleInstanceFormKey(msg)
	}
	if m.helpOpen {
		return m.handleHelpKey(msg)
	}
	if m.envDiff != nil {
		return m.handleEnvDiffKey(msg)
	}
	if m.paletteOpen {
		return m.handlePaletteKey(msg)
	}

	if m.filtering {
		return m.handleFilterKey(msg)
	}
	if m.logSearching {
		return m.handleLogSearchKey(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Help):
		m.openHelp()
		return m, nil

	case key.Matches(msg, m.keys.Palette):
		m.openPalette()
		return m, nil

	case key.Matches(msg, m.keys.Theme):
		m.cycleTheme()
		m.refreshDiagnostics()
		return m, m.pushToast(components.ToastInfo, "Theme: "+string(m.themeMode), "")

	case key.Matches(msg, m.keys.Compact):
		m.forceCompact = !m.forceCompact
		m.recomputeLayout()
		m.refreshDiagnostics()
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, m.manualRefresh()

	case key.Matches(msg, m.keys.Snapshot):
		return m, m.copyFleetSnapshot()

	case key.Matches(msg, m.keys.NextPane):
		m.cycleFocus(1)
		return m, nil

	case key.Matches(msg, m.keys.PrevPane):
		m.cycleFocus(-1)
		return m, nil

	case key.Matches(msg, m.keys.SectionApplications):
		return m.gotoSection(SectionApplications)
	case key.Matches(msg, m.keys.SectionDeployments):
		return m.gotoSection(SectionDeployments)
	case key.Matches(msg, m.keys.SectionInstances):
		return m.gotoSection(SectionInstances)
	case key.Matches(msg, m.keys.SectionDiagnostics):
		return m.gotoSection(SectionDiagnostics)

	case key.Matches(msg, m.keys.DemoToggleOffline):
		type offlineToggler interface {
			SetOffline(bool)
			Offline() bool
		}
		if m.opts.Demo {
			if d, ok := m.service.(offlineToggler); ok {
				next := !d.Offline()
				d.SetOffline(next)
				state := "online"
				if next {
					state = "offline"
				}
				return m, tea.Batch(
					m.pushToast(components.ToastInfo, "Demo outage "+state, ""),
					m.manualRefresh(),
				)
			}
		}
	}

	if m.focus == focusSidebar {
		return m.handleSidebarKey(msg)
	}

	switch m.screen {
	case screenTail:
		return m.handleTailKey(msg)
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

	// String() renders space as "space"; Text carries the literal input,
	// and is empty for keys that produce none.
	if text := msg.Key().Text; text != "" {
		m.filterText += text
		m.apps.SetFilter(m.filterText)
	}
	return m, nil
}

func (m *Model) handleLogSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Confirm) {
		m.logSearching = false
		return m, nil
	}

	switch msg.String() {
	case "backspace":
		if len(m.logSearchText) > 0 {
			runes := []rune(m.logSearchText)
			m.logSearchText = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.logSearchText = ""
	default:
		// String() renders space as "space"; Text carries the literal input,
		// and is empty for keys that produce none.
		if text := msg.Key().Text; text != "" {
			m.logSearchText += text
		}
	}
	m.detail.SetRuntimeSearch(m.logSearchText)
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

// handleListKey drives the list-level screens for every top-level section.
func (m *Model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.section {
	case SectionDeployments:
		return m.handleDeploymentsKey(msg)
	case SectionInstances:
		return m.handleInstancesKey(msg)
	case SectionDiagnostics:
		return m.handleDiagnosticsKey(msg)
	case SectionApplications:
		return m.handleApplicationsKey(msg)
	default:
		return m.handleApplicationsKey(msg)
	}
}

func (m *Model) handleApplicationsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	page := max(m.contentHeight()-2, 1)

	switch {
	case key.Matches(msg, m.keys.Mark):
		m.apps.ToggleMark()
		return m, nil
	case key.Matches(msg, m.keys.Tail):
		return m.openTail()
	case key.Matches(msg, m.keys.CompareEnv):
		return m, m.handleCompareEnv()
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
	case key.Matches(msg, m.keys.Deploy):
		return m, m.stageAction(actionDeploy, false)
	case key.Matches(msg, m.keys.ForceDeploy):
		return m, m.stageAction(actionDeploy, true)
	case key.Matches(msg, m.keys.Restart):
		return m, m.stageAction(actionRestart, false)
	case key.Matches(msg, m.keys.StartStop):
		if application, ok := m.currentApplication(); ok && application.Status.IsRunning() {
			return m, m.stageAction(actionStop, false)
		}
		return m, m.stageAction(actionStart, false)

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

	case key.Matches(msg, m.keys.Terminal):
		return m, m.openTerminal()

	case key.Matches(msg, m.keys.CopyUUID):
		return m, m.copyApplicationUUID()

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

func (m *Model) handleDeploymentsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	page := max(m.contentHeight()-2, 1)
	switch {
	case key.Matches(msg, m.keys.Up):
		m.deployments.Move(-1)
	case key.Matches(msg, m.keys.Down):
		m.deployments.Move(1)
	case key.Matches(msg, m.keys.Top):
		m.deployments.MoveTo(0)
	case key.Matches(msg, m.keys.Bottom):
		m.deployments.MoveTo(m.deployments.Count() - 1)
	case key.Matches(msg, m.keys.PageDown):
		m.deployments.Move(page)
	case key.Matches(msg, m.keys.PageUp):
		m.deployments.Move(-page)
	case key.Matches(msg, m.keys.Left):
		if m.layout.ShowSidebar {
			m.focus = focusSidebar
		}
	case key.Matches(msg, m.keys.Enter), key.Matches(msg, m.keys.BuildLogs):
		return m.openSelectedDeployment()
	case key.Matches(msg, m.keys.CopyUUID):
		if d, ok := m.deployments.Selected(); ok && d.UUID != "" {
			return m, m.copyText(d.UUID, "Copied deployment UUID")
		}
	case key.Matches(msg, m.keys.DeploymentsActive):
		only := m.deployments.ToggleActiveOnly()
		label := "recent history"
		if only {
			label = "active only"
		}
		return m, m.pushToast(components.ToastInfo, "Deployments filter: "+label, "")
	case key.Matches(msg, m.keys.DeploymentsTimeline):
		label := "table"
		if m.deployments.ToggleTimeline() {
			label = "timeline"
		}
		return m, m.pushToast(components.ToastInfo, "Deployments view: "+label, "")
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Back):
		return m.gotoSection(SectionApplications)
	}
	return m, nil
}

func (m *Model) handleInstancesKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	page := max(m.contentHeight()-2, 1)
	switch {
	case key.Matches(msg, m.keys.Up):
		m.instances.Move(-1)
	case key.Matches(msg, m.keys.Down):
		m.instances.Move(1)
	case key.Matches(msg, m.keys.Top):
		m.instances.MoveTo(0)
	case key.Matches(msg, m.keys.Bottom):
		m.instances.MoveTo(m.instances.Count() - 1)
	case key.Matches(msg, m.keys.PageDown):
		m.instances.Move(page)
	case key.Matches(msg, m.keys.PageUp):
		m.instances.Move(-page)
	case key.Matches(msg, m.keys.Left):
		if m.layout.ShowSidebar {
			m.focus = focusSidebar
		}
	case key.Matches(msg, m.keys.Enter):
		return m, m.activateSelectedInstance()
	case key.Matches(msg, m.keys.TestConnection):
		return m, tea.Batch(
			m.pushToast(components.ToastInfo, "Testing connection…", ""),
			m.testActiveConnection(),
		)
	case key.Matches(msg, m.keys.AddInstance):
		return m, m.openAddInstanceForm()
	case key.Matches(msg, m.keys.EditInstance):
		return m, m.openEditInstanceForm()
	case key.Matches(msg, m.keys.DeleteInstance):
		if row, ok := m.instances.Selected(); ok {
			return m, m.stageDeleteInstance(row.ID, row.Name)
		}
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Back):
		return m.gotoSection(SectionApplications)
	}
	return m, nil
}

func (m *Model) handleDiagnosticsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	page := max(m.contentHeight()-2, 1)
	switch {
	case key.Matches(msg, m.keys.Up):
		m.diagnostics.Scroll(-1, page)
	case key.Matches(msg, m.keys.Down):
		m.diagnostics.Scroll(1, page)
	case key.Matches(msg, m.keys.PageDown):
		m.diagnostics.Scroll(page, page)
	case key.Matches(msg, m.keys.PageUp):
		m.diagnostics.Scroll(-page, page)
	case key.Matches(msg, m.keys.Top):
		m.diagnostics.Scroll(-1<<20, page)
	case key.Matches(msg, m.keys.Left):
		if m.layout.ShowSidebar {
			m.focus = focusSidebar
		}
	case key.Matches(msg, m.keys.CopyUUID):
		return m, m.copyDiagnostics()
	case key.Matches(msg, m.keys.ExportDiagnostics):
		return m, m.exportDiagnostics()
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Back):
		return m.gotoSection(SectionApplications)
	}
	return m, nil
}

// openSelectedDeployment opens the application detail for the selected
// global deployment and jumps into its build log when possible.
func (m *Model) openSelectedDeployment() (tea.Model, tea.Cmd) {
	d, ok := m.deployments.Selected()
	if !ok {
		return m, nil
	}
	appUUID := d.ApplicationUUID
	if appUUID == "" {
		return m, m.pushToast(components.ToastWarning, "No application on deployment", "")
	}
	application := domain.Application{UUID: appUUID, Name: d.ApplicationName}
	if m.apps.SelectUUID(appUUID) {
		if a, ok := m.apps.Selected(); ok {
			application = a
		}
	}
	m.section = SectionApplications
	m.screen = screenDetail
	m.detail.SetApplication(application, m.apps.LoadedAt())
	m.detail.SetTab(views.TabDeployments)
	m.detail.SetDeployments([]domain.Deployment{d})
	if uuid, ok := m.detail.OpenSelectedDeploymentLogs(); ok {
		return m, tea.Batch(m.loadDetail(appUUID), m.loadDeploymentLogs(uuid))
	}
	return m, m.loadDetail(appUUID)
}

// handleDetailKey drives the application detail screen.
func (m *Model) handleDetailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
		if m.detail.DeploymentLogsOpen() {
			m.cancelDeploymentLogsRequest()
			m.detail.CloseDeploymentLogs()
			return m, nil
		}
		m.cancelRuntimeLogsRequest()
		m.cancelDeploymentLogsRequest()
		m.screen = screenList
		return m, nil

	case key.Matches(msg, m.keys.RuntimeLogs):
		m.detail.SetTab(views.TabRuntimeLogs)
		return m, m.loadActiveDetailTab()
	case key.Matches(msg, m.keys.BuildLogs):
		m.detail.SetTab(views.TabDeployments)
		if uuid, ok := m.detail.OpenSelectedDeploymentLogs(); ok {
			return m, m.loadDeploymentLogs(uuid)
		}
	case key.Matches(msg, m.keys.Deploy):
		return m, m.stageAction(actionDeploy, false)
	case key.Matches(msg, m.keys.ForceDeploy):
		return m, m.stageAction(actionDeploy, true)
	case key.Matches(msg, m.keys.Restart):
		return m, m.stageAction(actionRestart, false)
	case key.Matches(msg, m.keys.StartStop):
		if application, ok := m.currentApplication(); ok && application.Status.IsRunning() {
			return m, m.stageAction(actionStop, false)
		}
		return m, m.stageAction(actionStart, false)
	case key.Matches(msg, m.keys.Right):
		m.detail.NextTab()
		return m, m.loadActiveDetailTab()
	case key.Matches(msg, m.keys.Left):
		m.detail.PrevTab()
		return m, m.loadActiveDetailTab()
	case key.Matches(msg, m.keys.LogPause):
		if m.detail.Tab() == views.TabRuntimeLogs {
			if m.detail.ToggleRuntimePause() {
				m.cancelRuntimeLogsRequest()
				return m, nil
			}
			return m, m.loadRuntimeLogs(m.detail.Application().UUID)
		}
	case key.Matches(msg, m.keys.LogFollow):
		if m.detail.Tab() == views.TabRuntimeLogs {
			m.detail.ToggleRuntimeFollow()
		}
	case key.Matches(msg, m.keys.LogWrap):
		if m.detail.Tab() == views.TabRuntimeLogs {
			m.detail.ToggleRuntimeWrap()
		}
	case key.Matches(msg, m.keys.LogSearch):
		if m.detail.Tab() == views.TabRuntimeLogs {
			m.logSearching = true
			m.logSearchText = m.detail.RuntimeSearch()
		}
	case key.Matches(msg, m.keys.LogNext):
		if m.detail.Tab() == views.TabRuntimeLogs {
			m.detail.NextRuntimeMatch(1)
		}
	case key.Matches(msg, m.keys.LogPrev):
		if m.detail.Tab() == views.TabRuntimeLogs {
			m.detail.NextRuntimeMatch(-1)
		}
	case key.Matches(msg, m.keys.LogClear):
		return m, m.clearActiveLogs()
	case key.Matches(msg, m.keys.LogMore):
		return m, m.adjustLogLines(1)
	case key.Matches(msg, m.keys.LogLess):
		return m, m.adjustLogLines(-1)
	case key.Matches(msg, m.keys.CopyUUID):
		return m, m.copyFromDetail()
	case key.Matches(msg, m.keys.Enter):
		if m.detail.Tab() == views.TabDeployments && !m.detail.DeploymentLogsOpen() {
			if uuid, ok := m.detail.OpenSelectedDeploymentLogs(); ok {
				return m, m.loadDeploymentLogs(uuid)
			}
		}

	case key.Matches(msg, m.keys.Down):
		if m.detail.Tab() == views.TabDeployments && !m.detail.DeploymentLogsOpen() {
			m.detail.MoveDeployment(1)
		} else {
			m.detail.Scroll(1)
		}
	case key.Matches(msg, m.keys.Up):
		if m.detail.Tab() == views.TabDeployments && !m.detail.DeploymentLogsOpen() {
			m.detail.MoveDeployment(-1)
		} else {
			m.detail.Scroll(-1)
		}
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
	case key.Matches(msg, m.keys.Terminal):
		return m, m.openTerminal()
	}
	return m, nil
}

// logLinesStep is how many lines +/- changes the runtime log window by.
const logLinesStep = 50

const (
	minSessionLogLines = 10
	maxSessionLogLines = 10000
)

// adjustLogLines changes the session log window size and reloads when runtime
// logs are open. Outside the log view the binding is a no-op.
func (m *Model) adjustLogLines(direction int) tea.Cmd {
	if m.screen != screenDetail || m.detail.Tab() != views.TabRuntimeLogs {
		return nil
	}
	next := min(max(m.logLines+direction*logLinesStep, minSessionLogLines), maxSessionLogLines)
	if next == m.logLines {
		return m.pushToast(components.ToastInfo, "Log lines already at "+strconv.Itoa(m.logLines), "")
	}
	m.logLines = next
	toast := m.pushToast(components.ToastInfo, "Fetching "+strconv.Itoa(m.logLines)+" log lines", "")
	if m.detail.RuntimeLogsPaused() {
		return toast
	}
	return tea.Batch(toast, m.loadRuntimeLogs(m.detail.Application().UUID))
}

// clearActiveLogs clears the local log buffer for the open log pane.
func (m *Model) clearActiveLogs() tea.Cmd {
	if m.screen != screenDetail {
		return nil
	}
	if m.detail.DeploymentLogsOpen() {
		if m.detail.DeploymentLogCount() == 0 {
			return m.pushToast(components.ToastInfo, "Log buffer already empty", "")
		}
		m.detail.ClearDeploymentLogs()
		return m.pushToast(components.ToastInfo, "Cleared deployment log buffer", "")
	}
	if m.detail.Tab() == views.TabRuntimeLogs {
		if m.detail.RuntimeLogCount() == 0 {
			return m.pushToast(components.ToastInfo, "Log buffer already empty", "")
		}
		m.detail.ClearRuntimeLogs()
		return m.pushToast(components.ToastInfo, "Cleared runtime log buffer", "")
	}
	return nil
}

// copyFromDetail copies the most relevant detail-screen value: log text when a
// log pane is open, the selected deployment UUID on the deployments table, or
// the application UUID otherwise.
func (m *Model) copyFromDetail() tea.Cmd {
	if m.detail.DeploymentLogsOpen() {
		text := m.detail.DeploymentLogText()
		if text == "" {
			return m.pushToast(components.ToastWarning, "Nothing to copy", "No deployment log lines loaded.")
		}
		n := m.detail.DeploymentLogCount()
		return m.copyText(text, "Copied "+strconv.Itoa(n)+" log line"+plural(n))
	}
	if m.detail.Tab() == views.TabRuntimeLogs {
		text := m.detail.RuntimeLogText()
		if text == "" {
			return m.pushToast(components.ToastWarning, "Nothing to copy", "No runtime log lines loaded.")
		}
		if m.detail.RuntimeSearch() != "" {
			if current, total := m.detail.RuntimeSearchStatus(); total > 0 {
				return m.copyText(text, "Copied search match "+strconv.Itoa(current)+"/"+strconv.Itoa(total))
			}
		}
		n := m.detail.RuntimeLogCount()
		return m.copyText(text, "Copied "+strconv.Itoa(n)+" log line"+plural(n))
	}
	if m.detail.Tab() == views.TabDeployments {
		if deployment, ok := m.detail.SelectedDeployment(); ok && deployment.UUID != "" {
			return m.copyText(deployment.UUID, "Copied deployment UUID")
		}
	}
	return m.copyApplicationUUID()
}

// copyApplicationUUID copies the current application's UUID.
func (m *Model) copyApplicationUUID() tea.Cmd {
	a, ok := m.currentApplication()
	if !ok || a.UUID == "" {
		return m.pushToast(components.ToastWarning, "Nothing to copy", "No application selected.")
	}
	return m.copyText(a.UUID, "Copied application UUID")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func (m *Model) loadActiveDetailTab() tea.Cmd {
	if m.detail.Tab() == views.TabRuntimeLogs && !m.detail.RuntimeLogsLoaded() {
		return m.loadRuntimeLogs(m.detail.Application().UUID)
	}
	if m.cancelRuntimeLogs != nil && m.detail.Tab() != views.TabRuntimeLogs {
		m.cancelRuntimeLogsRequest()
	}
	return nil
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

// terminalPicker is open while the user chooses between the running containers
// of a multi-container application, mirroring the picker Coolify's web
// terminal offers.
type terminalPicker struct {
	app        domain.Application
	containers []string
	selected   int
}

// openTerminal starts a terminal session for the current application. The
// containers are listed over SSH first (the Coolify API has no exec or
// container endpoint): one match connects immediately, several open the
// picker. ssh_host on the instance config is what authorises the whole flow,
// so without one the key explains itself instead of failing.
func (m *Model) openTerminal() tea.Cmd {
	a, ok := m.currentApplication()
	if !ok {
		return nil
	}
	if m.cancelTerminal != nil {
		m.cancelTerminal()
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	m.cancelTerminal = cancel
	m.terminalSeq++
	seq := m.terminalSeq

	cmd, err := platform.ContainerListCommand(ctx, activeInstance(m).SSHHost, a.UUID)
	if err != nil {
		cancel()
		m.cancelTerminal = nil
		return m.pushToast(components.ToastWarning, "Terminal unavailable",
			err.Error()+`. Set ssh_host = "user@host" on the instance in the config file.`)
	}
	list := func() tea.Msg {
		defer cancel()
		out, err := cmd.Output()
		if err != nil {
			return terminalListFailedMsg{Seq: seq, Err: err}
		}
		return terminalContainersMsg{Seq: seq, App: a, Containers: platform.ParseContainerList(string(out))}
	}
	return tea.Batch(m.pushToast(components.ToastInfo, "Looking up containers", ""), list)
}

// execTerminal suspends the TUI and hands the screen to an interactive SSH
// session inside one container.
func (m *Model) execTerminal(container string) tea.Cmd {
	cmd, err := platform.TerminalCommand(activeInstance(m).SSHHost, container)
	if err != nil {
		return m.pushToast(components.ToastError, "Terminal unavailable", err.Error())
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return terminalDoneMsg{Err: err} })
}

// handleTerminalPickerKey drives the container choice modal.
func (m *Model) handleTerminalPickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := m.terminalPicker
	switch {
	case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Quit):
		m.terminalPicker = nil
	case key.Matches(msg, m.keys.Up):
		p.selected = max(p.selected-1, 0)
	case key.Matches(msg, m.keys.Down):
		p.selected = min(p.selected+1, len(p.containers)-1)
	case key.Matches(msg, m.keys.Confirm):
		container := p.containers[p.selected]
		m.terminalPicker = nil
		return m, m.execTerminal(container)
	}
	return m, nil
}

// terminalErrDetail surfaces ssh's own stderr, which is where the actionable
// part of a connection failure lives; exec's generic "exit status 255" is not.
// The bytes are remote-controlled (sshd emits its banner pre-auth), so they
// pass through SanitizeLogText like every other externally-sourced text.
func terminalErrDetail(err error) string {
	var exit *exec.ExitError
	if errors.As(err, &exit) && len(exit.Stderr) > 0 {
		return domain.SanitizeLogText(strings.TrimSpace(string(exit.Stderr)))
	}
	return domain.SanitizeLogText(err.Error())
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
	s = clampSection(s)
	if s == m.section && m.screen == screenList {
		return m, nil
	}
	// Leaving detail cancels its log streams so they cannot keep polling.
	if m.screen == screenDetail {
		m.cancelRuntimeLogsRequest()
		m.cancelDeploymentLogsRequest()
	}
	m.section = s
	m.screen = screenList
	m.focus = focusContent
	if s == SectionInstances {
		m.refreshInstances()
	}
	if s == SectionDiagnostics {
		m.refreshDiagnostics()
	}
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

// cycleTheme steps through available themes.
func (m *Model) cycleTheme() {
	switch m.themeMode {
	case config.ThemeAuto:
		m.themeMode = config.ThemeDark
	case config.ThemeDark:
		m.themeMode = config.ThemeLight
	case config.ThemeLight:
		m.themeMode = config.ThemeDracula
	case config.ThemeDracula:
		m.themeMode = config.ThemeCatppuccin
	case config.ThemeCatppuccin:
		m.themeMode = config.ThemeNord
	case config.ThemeNord:
		m.themeMode = config.ThemeGruvbox
	case config.ThemeGruvbox:
		m.themeMode = config.ThemeTokyoNight
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
	if s < 0 {
		return SectionApplications
	}
	if s >= sectionCount {
		return sectionCount - 1
	}
	return s
}

func trimLastWord(s string) string {
	s = strings.TrimRight(s, " ")
	if i := strings.LastIndexByte(s, ' '); i >= 0 {
		return s[:i+1]
	}
	return ""
}

// handleTailKey drives the fleet tail. The bindings deliberately mirror the
// single-application log view, so what the user already learned still works.
func (m *Model) handleTailKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.tailSearching {
		return m.handleTailSearchKey(msg)
	}

	page := max(m.contentHeight()-2, 1)

	switch {
	case key.Matches(msg, m.keys.Back), key.Matches(msg, m.keys.Quit):
		m.closeTail()
		return m, nil
	case key.Matches(msg, m.keys.Up):
		m.tail.Scroll(-1)
	case key.Matches(msg, m.keys.Down):
		m.tail.Scroll(1)
	case key.Matches(msg, m.keys.PageUp):
		m.tail.Scroll(-page)
	case key.Matches(msg, m.keys.PageDown):
		m.tail.Scroll(page)
	case key.Matches(msg, m.keys.LogPause):
		if paused := m.tail.TogglePause(); !paused {
			// Resuming re-arms every source at once; they re-stagger from the
			// replies that follow.
			return m, m.resumeTail()
		}
		return m, nil
	case key.Matches(msg, m.keys.LogFollow):
		m.tail.ToggleFollow()
	case key.Matches(msg, m.keys.LogWrap):
		m.tail.ToggleWrap()
	case key.Matches(msg, m.keys.Filter):
		m.tailSearching = true
		m.tailSearchText = m.tail.Search()
		return m, nil
	case key.Matches(msg, m.keys.CopyUUID):
		return m, m.copyText(m.tail.Text(), "Tail copied")
	}
	return m, nil
}

func (m *Model) handleTailSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Confirm) {
		m.tailSearching = false
		return m, nil
	}

	switch msg.String() {
	case "backspace":
		if runes := []rune(m.tailSearchText); len(runes) > 0 {
			m.tailSearchText = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.tailSearchText = ""
	default:
		// String() renders space as "space"; Text carries the literal input.
		if text := msg.Key().Text; text != "" {
			m.tailSearchText += text
		}
	}
	m.tail.SetSearch(m.tailSearchText)
	return m, nil
}

// resumeTail re-arms every source after a pause. Bumping the sequence first
// orphans any tick scheduled before the pause - without it, a pending tick
// fires alongside the fresh ones and every pause/resume cycle doubles the
// polling rate.
func (m *Model) resumeTail() tea.Cmd {
	m.tailSeq++
	uuids := m.tail.UUIDs()
	cmds := make([]tea.Cmd, 0, len(uuids))
	for i, uuid := range uuids {
		cmds = append(cmds, m.tailTick(uuid, tailStagger(i, len(uuids))))
	}
	return tea.Batch(cmds...)
}
