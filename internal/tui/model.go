package tui

import (
	"context"
	"log/slog"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
	"github.com/resetnak/cooldeck/internal/tui/views"
)

// requestTimeout bounds every individual API call. It is generous enough for a
// slow instance over a slow link, but short enough that a dead host does not
// leave the UI waiting indefinitely.
const requestTimeout = 20 * time.Second

// frameInterval drives spinner animation and relative-time updates. Two ticks
// per second is enough for both and costs almost nothing over SSH.
const frameInterval = 500 * time.Millisecond

// Section is a top-level destination.
type Section int

// Available sections.
const (
	SectionApplications Section = iota
	SectionDeployments
	SectionInstances
	SectionDiagnostics
	sectionCount
)

// Label names the section for navigation.
func (s Section) Label() string {
	switch s {
	case SectionDeployments:
		return "Deployments"
	case SectionInstances:
		return "Instances"
	case SectionDiagnostics:
		return "Diagnostics"
	default:
		return "Applications"
	}
}

// Short returns the abbreviated label used in the compact tab bar.
func (s Section) Short() string {
	switch s {
	case SectionDeployments:
		return "Deploys"
	case SectionInstances:
		return "Inst"
	case SectionDiagnostics:
		return "Diag"
	default:
		return "Apps"
	}
}

// screen is the current level within a section.
type screen int

const (
	screenList screen = iota
	screenDetail
	screenTail
)

// focusTarget is the pane keystrokes are routed to.
type focusTarget int

const (
	focusContent focusTarget = iota
	focusSidebar
)

// OpenServiceFunc builds an app.Service for a configured instance ID. The TUI
// calls it when the user switches instances mid-session.
type OpenServiceFunc func(ctx context.Context, instanceID string) (app.Service, string, error)

// SaveConfigFunc persists the in-memory config after local instance edits.
type SaveConfigFunc func(cfg config.Config) error

// RemoveCredentialsFunc deletes the OS keyring entry (or equivalent) for an
// instance after it is removed from the config. Missing credentials are OK.
type RemoveCredentialsFunc func(inst config.Instance) error

// Options configures the root model.
type Options struct {
	Config  config.Config
	Service app.Service
	Logger  *slog.Logger

	// InstanceName labels the active instance in the header.
	InstanceName string
	// Demo marks the run as using generated data.
	Demo bool
	// Theme is the requested colour scheme; "auto" is resolved from the
	// terminal's reported background colour.
	Theme config.Theme
	// Mouse enables mouse reporting.
	Mouse bool
	// ASCII forces the plain-text glyph set for terminals that cannot render
	// box-drawing and geometric characters.
	ASCII bool

	// ConfigPath is the absolute path of the loaded config file, for diagnostics.
	ConfigPath string
	// LogPath is the application log file path, for diagnostics.
	LogPath string
	// RecentErrors feeds the diagnostics screen. Optional; when nil only
	// in-session errors recorded by the model are shown.
	RecentErrors func() []string

	// OpenService enables mid-session instance switching. When nil, Enter on
	// another instance only explains how to switch from the shell.
	OpenService OpenServiceFunc
	// SaveConfig writes instance-list changes. When nil, delete is disabled.
	SaveConfig SaveConfigFunc
	// RemoveCredentials cleans the keyring after a local instance delete.
	RemoveCredentials RemoveCredentialsFunc
	// StoreCredentials saves a token when adding an instance from the TUI.
	StoreCredentials StoreCredentialsFunc

	// Now injects the clock. Nil uses time.Now. Tests pin it so that relative
	// timestamps and golden snapshots are stable.
	Now func() time.Time
}

// Model is the Bubble Tea root. It owns the data, the async lifecycle and the
// chrome; the views own their own selection and rendering.
type Model struct {
	opts    Options
	keys    KeyMap
	now     func() time.Time
	service app.Service
	log     *slog.Logger

	theme        *theme.Theme
	themeMode    config.Theme
	darkTerm     bool
	layout       theme.Layout
	width        int
	height       int
	forceCompact bool

	section Section
	screen  screen
	focus   focusTarget

	apps        *views.Applications
	detail      *views.Detail
	deployments *views.Deployments
	instances   *views.Instances
	diagnostics *views.Diagnostics
	tail        *views.Tail

	// Sequence numbers and cancel functions implement request supersession:
	// starting a new request of a kind cancels the previous one and bumps the
	// sequence, so a late reply is both stopped and ignored.
	connectSeq     uint64
	dashboardSeq   uint64
	detailSeq      uint64
	runtimeLogsSeq uint64
	// tailSeq identifies one fleet-tail session. Every source's replies carry
	// it, so leaving the tail orphans all of them at once without having to
	// track a cancel function per application.
	tailSeq              uint64
	deploymentLogsSeq    uint64
	operationSeq         uint64
	cancelDashboard      context.CancelFunc
	cancelDetail         context.CancelFunc
	cancelRuntimeLogs    context.CancelFunc
	cancelDeploymentLogs context.CancelFunc
	cancelOperation      context.CancelFunc

	connection    components.ConnectionState
	connectionErr *domain.Error
	capabilities  app.Capabilities
	coolifyVer    string
	// lastConnection holds the non-secret fields from the most recent Connect.
	lastConnection app.Connection

	loading    bool
	refreshing bool
	lastError  *domain.Error
	// staleSince records when the last successful load happened, so cached
	// data can be labelled instead of being silently presented as current.
	lastSuccess time.Time
	// recentErrors is a short ring of user-visible failures for diagnostics.
	recentErrors []string

	filtering     bool
	filterText    string
	logSearching  bool
	logSearchText string
	// logLines is the session override of config.LogLines. +/- on the log
	// view changes it without writing config back to disk.
	logLines int

	tailSearching  bool
	tailSearchText string

	paletteOpen     bool
	paletteQuery    string
	paletteSelected int

	helpOpen   bool
	helpScroll int

	// instanceForm is non-nil while the add/edit instance modal is open.
	instanceForm *instanceForm

	pendingAction     *pendingAction
	operationInFlight bool

	// deployStates is the deployment status seen on the previous refresh, keyed
	// by deployment UUID. It is rebuilt from every snapshot, so it cannot grow
	// past what the instance itself reports.
	deployStates map[string]domain.DeploymentStatus
	// ownDeploys marks deployments this session triggered. Their success is
	// worth announcing; someone else's is not.
	ownDeploys map[string]bool

	toasts     []components.Toast
	nextToast  int
	spinnerIdx int

	quitting bool
}

var _ tea.Model = (*Model)(nil)

// New builds the root model.
func New(opts Options) *Model {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	log := opts.Logger
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	logLines := opts.Config.LogLines
	if logLines <= 0 {
		logLines = config.DefaultLogLines
	}

	m := &Model{
		opts:        opts,
		keys:        DefaultKeyMap(),
		now:         now,
		service:     opts.Service,
		log:         log,
		themeMode:   opts.Theme,
		apps:        views.NewApplications(),
		detail:      views.NewDetail(),
		deployments: views.NewDeployments(),
		instances:   views.NewInstances(),
		diagnostics: views.NewDiagnostics(),
		tail:        views.NewTail(),
		logLines:    logLines,
		// Until the terminal reports its background colour, assume dark: it is
		// by far the more common terminal configuration, so the wrong guess is
		// visible for at most one frame.
		darkTerm:     true,
		connection:   components.ConnectionConnecting,
		loading:      true,
		capabilities: app.FullCapabilities(),
		deployStates: map[string]domain.DeploymentStatus{},
		ownDeploys:   map[string]bool{},
	}
	m.forceCompact = opts.Config.UI.CompactMode == config.TristateOn
	m.rebuildTheme()
	m.refreshInstances()
	m.refreshDiagnostics()
	return m
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		tea.RequestBackgroundColor,
		m.connect(),
		m.frameTick(),
	)
}

// View implements tea.Model.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "cooldeck"
	if m.opts.Mouse {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.recomputeLayout()
		return m, nil

	case tea.BackgroundColorMsg:
		// Only "auto" follows the terminal; an explicit choice is the user's.
		if m.themeMode == config.ThemeAuto {
			m.darkTerm = msg.IsDark()
			m.rebuildTheme()
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case connectedMsg:
		if msg.Seq != m.connectSeq {
			return m, nil
		}
		m.connection = components.ConnectionOnline
		m.connectionErr = nil
		m.capabilities = msg.Connection.Capabilities
		m.coolifyVer = msg.Connection.Version
		m.lastConnection = msg.Connection
		m.refreshInstances()
		m.refreshDiagnostics()
		m.log.Info("connected",
			"instance", msg.Connection.InstanceID,
			"version", msg.Connection.Version,
			"latency_ms", msg.Connection.Latency.Milliseconds())
		return m, tea.Batch(m.loadDashboard(), m.refreshTick())

	case connectFailedMsg:
		if msg.Seq != m.connectSeq {
			return m, nil
		}
		m.loading = false
		m.connectionErr = msg.Err
		m.connection = connectionStateFor(msg.Err)
		m.lastError = msg.Err
		m.rememberError(msg.Err)
		m.refreshInstances()
		m.refreshDiagnostics()
		m.log.Warn("connect failed", "kind", string(msg.Err.Kind), "err", msg.Err.Error())
		return m, tea.Batch(m.pushToast(components.ToastError, msg.Err.Title, msg.Err.Message), m.refreshTick())

	case dashboardLoadedMsg:
		if msg.Seq != m.dashboardSeq {
			return m, nil
		}
		m.loading, m.refreshing = false, false
		m.lastError = nil
		m.lastSuccess = msg.Snapshot.LoadedAt
		m.connection = components.ConnectionOnline
		m.apps.SetApplications(msg.Snapshot.Applications, msg.Snapshot.ActiveDeployments, msg.Snapshot.LoadedAt)
		recent := msg.Snapshot.RecentDeployments
		if len(recent) == 0 {
			recent = msg.Snapshot.ActiveDeployments
		}
		m.deployments.SetItems(recent, msg.Snapshot.LoadedAt)
		m.syncDetailFromList()
		m.refreshDiagnostics()
		cmds := m.deploymentOutcomes(msg.Snapshot)
		for _, w := range msg.Snapshot.Warnings {
			cmds = append(cmds, m.pushToast(components.ToastWarning, w, ""))
		}
		return m, tea.Batch(cmds...)

	case dashboardFailedMsg:
		if msg.Seq != m.dashboardSeq {
			return m, nil
		}
		m.loading, m.refreshing = false, false
		if msg.Err.Kind == domain.ErrorCancelled {
			// A cancelled request was superseded on purpose; it is not a fault.
			return m, nil
		}
		m.lastError = msg.Err
		m.connection = connectionStateFor(msg.Err)
		m.rememberError(msg.Err)
		m.refreshInstances()
		m.refreshDiagnostics()
		m.log.Warn("dashboard load failed", "kind", string(msg.Err.Kind), "err", msg.Err.Error())
		// With cached data on screen the failure is a toast, not a takeover.
		if m.apps.Loaded() {
			return m, m.pushToast(components.ToastError, msg.Err.Title, msg.Err.Message)
		}
		return m, nil

	case detailLoadedMsg:
		if msg.Seq != m.detailSeq {
			return m, nil
		}
		m.detail.SetApplication(msg.Detail.Application, msg.Detail.LoadedAt)
		m.detail.SetDeployments(msg.Detail.Deployments)
		return m, nil

	case detailFailedMsg:
		if msg.Seq != m.detailSeq || msg.Err.Kind == domain.ErrorCancelled {
			return m, nil
		}
		return m, m.pushToast(components.ToastError, msg.Err.Title, msg.Err.Message)

	case runtimeLogsLoadedMsg:
		if msg.Seq != m.runtimeLogsSeq || msg.AppUUID != m.detail.Application().UUID {
			return m, nil
		}
		m.detail.SetRuntimeLogs(msg.Snapshot.Lines, msg.Snapshot.LoadedAt, msg.Snapshot.Truncated)
		if m.runtimeLogsVisible(msg.AppUUID) {
			return m, m.runtimeLogsTick(msg.AppUUID)
		}
		return m, nil

	case runtimeLogsFailedMsg:
		if msg.Seq != m.runtimeLogsSeq || msg.Err.Kind == domain.ErrorCancelled {
			return m, nil
		}
		return m, m.pushToast(components.ToastError, msg.Err.Title, msg.Err.Message)

	case tailTickMsg:
		if msg.Seq != m.tailSeq || m.screen != screenTail || m.tail.Paused() {
			return m, nil
		}
		return m, m.loadTailLogs(msg.AppUUID)

	case tailLoadedMsg:
		return m, m.applyTailLoaded(msg)

	case tailFailedMsg:
		return m, m.applyTailFailed(msg)

	case runtimeLogsTickMsg:
		if !m.runtimeLogsVisible(msg.AppUUID) {
			return m, nil
		}
		return m, m.loadRuntimeLogs(msg.AppUUID)

	case deploymentLogsLoadedMsg:
		if msg.Seq != m.deploymentLogsSeq || !m.detail.DeploymentLogsOpen() {
			return m, nil
		}
		m.detail.SetDeploymentLogs(msg.DeploymentUUID, msg.Snapshot.Lines, msg.Snapshot.LoadedAt, msg.Snapshot.Truncated)
		return m, nil

	case deploymentLogsFailedMsg:
		if msg.Seq != m.deploymentLogsSeq || msg.Err.Kind == domain.ErrorCancelled {
			return m, nil
		}
		return m, m.pushToast(components.ToastError, msg.Err.Title, msg.Err.Message)

	case actionCompletedMsg:
		if msg.Seq != m.operationSeq {
			return m, nil
		}
		m.operationInFlight = false
		m.cancelOperation = nil
		m.capabilities = m.service.Capabilities()
		if msg.Result.DeploymentUUID != "" {
			// Remember it so the outcome is announced, not just the acceptance.
			m.ownDeploys[msg.Result.DeploymentUUID] = true
		}
		return m, tea.Batch(
			m.pushToast(components.ToastSuccess, "Action queued", "Coolify accepted the "+msg.Result.Operation+" request."),
			m.manualRefresh(),
		)

	case actionFailedMsg:
		if msg.Seq != m.operationSeq || msg.Err.Kind == domain.ErrorCancelled {
			return m, nil
		}
		m.operationInFlight = false
		m.cancelOperation = nil
		m.capabilities = m.service.Capabilities()
		return m, m.pushToast(components.ToastError, msg.Err.Title, msg.Err.Message)

	case refreshTickMsg:
		cmds := []tea.Cmd{m.refreshTick()}
		if m.connection == components.ConnectionOnline || m.connectionErr != nil {
			cmds = append(cmds, m.backgroundRefresh())
		}
		return m, tea.Batch(cmds...)

	case frameTickMsg:
		m.spinnerIdx++
		m.toasts = components.PruneToasts(m.toasts, m.now())
		return m, m.frameTick()

	case toastMsg:
		m.appendToast(components.ToastKind(msg.Kind), msg.Text, msg.Detail)
		return m, nil

	case openURLFailedMsg:
		return m, m.pushToast(components.ToastError, "Could not open link", msg.Err.Error())

	case instanceSwitchedMsg:
		return m, m.applyInstanceSwitch(msg)

	case instanceSwitchFailedMsg:
		return m, m.applyInstanceSwitchFailed(msg)

	case instanceFormSavedMsg:
		return m, m.applyInstanceFormSaved(msg)

	case instanceRemovedMsg:
		cmds := []tea.Cmd{}
		if msg.CredWarning != "" {
			cmds = append(cmds, m.pushToast(components.ToastWarning,
				"Credential cleanup failed", msg.CredWarning))
		}
		switch {
		case msg.EmptyFleet:
			m.apps = views.NewApplications()
			m.deployments = views.NewDeployments()
			m.connection = components.ConnectionOffline
			m.loading = false
			cmds = append(cmds, m.pushToast(components.ToastSuccess, "Instance removed",
				"No instances remain. Run cooldeck setup to add one."))
		case msg.NextID != "":
			cmds = append(cmds,
				m.pushToast(components.ToastSuccess, "Instance removed", msg.Name),
				m.switchInstance(msg.NextID))
		default:
			cmds = append(cmds, m.pushToast(components.ToastSuccess, "Instance removed",
				msg.Name+" deleted from local config"))
		}
		return m, tea.Batch(cmds...)

	case instanceDeleteFailedMsg:
		m.opts.Config.Instances = msg.PrevInstances
		m.opts.Config.DefaultInstance = msg.PrevDefault
		m.refreshInstances()
		m.refreshDiagnostics()
		return m, m.pushToast(components.ToastError, "Could not write config", msg.Err.Error())
	}

	return m, nil
}

func (m *Model) runtimeLogsVisible(appUUID string) bool {
	return m.screen == screenDetail &&
		m.detail.Tab() == views.TabRuntimeLogs &&
		!m.detail.RuntimeLogsPaused() &&
		m.detail.Application().UUID == appUUID
}

// recomputeLayout resolves geometry after a resize or a compact-mode toggle.
func (m *Model) recomputeLayout() {
	m.layout = theme.ComputeLayout(m.width, m.height, theme.LayoutOptions{
		ShowHeader:   m.opts.Config.UI.ShowHeader,
		ShowFooter:   m.opts.Config.UI.ShowFooter,
		ForceCompact: m.forceCompact,
		ForceRoomy:   m.opts.Config.UI.CompactMode == config.TristateOff,
	})
}

// rebuildTheme recreates the styles. It runs on startup and on a theme change,
// never per frame.
func (m *Model) rebuildTheme() {
	mode := theme.ModeDark
	paletteName := ""
	switch m.themeMode {
	case config.ThemeLight:
		mode = theme.ModeLight
	case config.ThemeDark:
		mode = theme.ModeDark
	case config.ThemeDracula, config.ThemeCatppuccin, config.ThemeNord, config.ThemeGruvbox, config.ThemeTokyoNight:
		paletteName = string(m.themeMode)
	default:
		if !m.darkTerm {
			mode = theme.ModeLight
		}
	}
	m.theme = theme.New(theme.Options{
		Mode:        mode,
		PaletteName: paletteName,
		NerdFont:    m.opts.Config.UI.NerdFont == config.TristateOn,
		ASCII:       m.opts.ASCII,
	})
}

// syncDetailFromList keeps an open detail screen in step with the list data,
// so a background refresh updates the detail without a second request.
func (m *Model) syncDetailFromList() {
	if m.screen != screenDetail || !m.detail.Loaded() {
		return
	}
	uuid := m.detail.Application().UUID
	if a, ok := m.apps.Selected(); ok && a.UUID == uuid {
		m.detail.SetApplication(a, m.apps.LoadedAt())
	}
}

func connectionStateFor(err *domain.Error) components.ConnectionState {
	if err == nil {
		return components.ConnectionOnline
	}
	switch err.Kind {
	case domain.ErrorUnauthorized, domain.ErrorForbidden:
		return components.ConnectionUnauthorized
	default:
		return components.ConnectionOffline
	}
}

// spinnerFrame returns the current spinner glyph.
func (m *Model) spinnerFrame() string {
	frames := m.theme.Sym.Spinner
	return frames[m.spinnerIdx%len(frames)]
}

// sections returns the navigation items with their current counts and
// availability. A section the token cannot serve is shown disabled rather than
// hidden, so the user learns the feature exists.
func (m *Model) sections() []components.NavItem {
	apps := components.NavItem{
		Label:   SectionApplications.Label(),
		Short:   SectionApplications.Short(),
		Count:   m.apps.Count(),
		Enabled: m.capabilities.Applications,
	}
	if !apps.Enabled {
		apps.Reason = "the token cannot list applications"
	}
	deploys := components.NavItem{
		Label:   SectionDeployments.Label(),
		Short:   SectionDeployments.Short(),
		Count:   m.deployments.Count(),
		Enabled: m.capabilities.Deployments,
	}
	if !deploys.Enabled {
		deploys.Reason = "the token cannot list deployments"
	}
	instances := components.NavItem{
		Label:   SectionInstances.Label(),
		Short:   SectionInstances.Short(),
		Count:   m.instances.Count(),
		Enabled: true,
	}
	diagnostics := components.NavItem{
		Label:   SectionDiagnostics.Label(),
		Short:   SectionDiagnostics.Short(),
		Count:   -1,
		Enabled: true,
	}
	return []components.NavItem{apps, deploys, instances, diagnostics}
}

// rememberError keeps a short, redacted trail of failures for the diagnostics
// screen. Titles only - never raw response bodies.
func (m *Model) rememberError(err *domain.Error) {
	if err == nil {
		return
	}
	line := err.Title
	if err.Message != "" {
		line += ": " + err.Message
	}
	line = domain.SanitizeLogText(line)
	m.recentErrors = append(m.recentErrors, line)
	const maxRecent = 12
	if len(m.recentErrors) > maxRecent {
		m.recentErrors = m.recentErrors[len(m.recentErrors)-maxRecent:]
	}
}

// refreshInstances rebuilds the instances list from config and live connection state.
func (m *Model) refreshInstances() {
	rows := make([]views.InstanceRow, 0)
	activeID := m.service.InstanceID()

	if m.opts.Demo {
		rows = append(rows, views.InstanceRow{
			ID:          "demo",
			Name:        m.instanceLabel(),
			URL:         m.lastConnection.BaseURL,
			TokenSource: "demo",
			Active:      true,
			Demo:        true,
			Version:     m.coolifyVer,
			Team:        m.lastConnection.TeamName,
			Latency:     m.lastConnection.Latency,
			Connection:  m.connection,
			ConnectedAt: m.lastConnection.ConnectedAt,
		})
		if m.connectionErr != nil {
			rows[0].LastError = m.connectionErr.Title
		}
		m.instances.SetItems(rows)
		return
	}

	// Stable order by ID so the list does not jump between refreshes.
	for _, id := range m.opts.Config.InstanceIDs() {
		inst := m.opts.Config.Instances[id]
		inst.ID = id
		row := views.InstanceRow{
			ID:          id,
			Name:        inst.DisplayName(),
			URL:         inst.URL,
			TokenSource: string(inst.TokenSource),
			Active:      id == activeID,
			Connection:  components.ConnectionConnecting,
		}
		if row.Active {
			row.Version = m.coolifyVer
			row.Team = m.lastConnection.TeamName
			row.Latency = m.lastConnection.Latency
			row.Connection = m.connection
			row.ConnectedAt = m.lastConnection.ConnectedAt
			if m.lastConnection.BaseURL != "" {
				row.URL = m.lastConnection.BaseURL
			}
			if m.connectionErr != nil {
				row.LastError = m.connectionErr.Title
			}
		}
		rows = append(rows, row)
	}
	// Active service with no matching config entry (edge case / tests).
	if len(rows) == 0 && activeID != "" {
		rows = append(rows, views.InstanceRow{
			ID:          activeID,
			Name:        m.instanceLabel(),
			URL:         m.lastConnection.BaseURL,
			Active:      true,
			Version:     m.coolifyVer,
			Team:        m.lastConnection.TeamName,
			Latency:     m.lastConnection.Latency,
			Connection:  m.connection,
			ConnectedAt: m.lastConnection.ConnectedAt,
		})
	}
	m.instances.SetItems(rows)
}

// refreshDiagnostics rebuilds the diagnostics field list from live state.
func (m *Model) refreshDiagnostics() {
	m.diagnostics.SetFields(m.buildDiagnosticFields())
}

// instanceLabel is the header's instance name.
func (m *Model) instanceLabel() string {
	if m.opts.InstanceName != "" {
		return m.opts.InstanceName
	}
	return "cooldeck"
}

// staleFor reports how long the displayed data has been stale, or zero when it
// is current.
func (m *Model) staleFor() time.Duration {
	if m.lastError == nil || m.lastSuccess.IsZero() {
		return 0
	}
	return m.now().Sub(m.lastSuccess)
}

// filterPrompt renders the filter input line while filtering is active.
func (m *Model) filterPrompt(width int) string {
	th := m.theme
	prompt := th.FilterPrompt.Render(th.Sym.Filter + " ")
	text := th.FilterText.Render(m.filterText) + th.FilterPrompt.Render("▏")
	hint := th.FilterHint.Render("  status: branch: domain: name:  " +
		th.Sym.Separator + "  esc clear  enter apply")

	line := prompt + text
	if components.Width(line)+components.Width(hint) <= width {
		line += hint
	}
	return components.Pad(line, width)
}

func (m *Model) logSearchPrompt(width int) string {
	th := m.theme
	prompt := th.FilterPrompt.Render("/ ")
	query := m.logSearchText
	if m.tailSearching {
		query = m.tailSearchText
	}
	text := th.FilterText.Render(query) + th.FilterPrompt.Render("▏")
	hint := th.FilterHint.Render("  enter apply  " + th.Sym.Separator + "  esc close")
	line := prompt + text
	if components.Width(line)+components.Width(hint) <= width {
		line += hint
	}
	return components.Pad(line, width)
}

// joinRows stacks rendered blocks, dropping empty ones so a hidden header does
// not leave a blank line behind.
func joinRows(rows ...string) string {
	kept := make([]string, 0, len(rows))
	for _, r := range rows {
		if r != "" {
			kept = append(kept, r)
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, kept...)
}

// trimTrailingBlank removes a trailing newline left by an empty region.
func trimTrailingBlank(s string) string { return strings.TrimRight(s, "\n") }
