package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/platform"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

// Every command below runs its API call on Bubble Tea's own goroutine pool, so
// the event loop is never blocked. Each call gets its own context with a
// timeout, and each request kind keeps a cancel function so that starting a
// new request aborts the one it replaces instead of leaving it running.

// connect verifies the instance and detects capabilities.
func (m *Model) connect() tea.Cmd {
	m.connectSeq++
	seq := m.connectSeq
	svc := m.service

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		conn, err := svc.Connect(ctx)
		if err != nil {
			return connectFailedMsg{Seq: seq, Err: domain.AsError(err)}
		}
		return connectedMsg{Seq: seq, Connection: conn}
	}
}

// loadDashboard starts a foreground load, replacing any in-flight one.
func (m *Model) loadDashboard() tea.Cmd {
	m.loading = !m.apps.Loaded()
	m.refreshing = m.apps.Loaded()
	return m.dashboardCmd()
}

// backgroundRefresh reloads without clearing the screen. If a refresh is
// already in flight it is left alone, so holding R cannot fan out into a
// storm of parallel requests.
func (m *Model) backgroundRefresh() tea.Cmd {
	if m.refreshing || m.loading {
		return nil
	}
	m.refreshing = true
	return m.dashboardCmd()
}

// manualRefresh is the explicit R action: immediate, and it supersedes any
// in-flight request rather than queueing behind it.
func (m *Model) manualRefresh() tea.Cmd {
	m.refreshing = true
	return m.dashboardCmd()
}

func (m *Model) dashboardCmd() tea.Cmd {
	if m.cancelDashboard != nil {
		m.cancelDashboard()
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	m.cancelDashboard = cancel

	m.dashboardSeq++
	seq := m.dashboardSeq
	svc := m.service

	return func() tea.Msg {
		defer cancel()
		snap, err := svc.Dashboard(ctx)
		if err != nil {
			return dashboardFailedMsg{Seq: seq, Err: domain.AsError(err)}
		}
		return dashboardLoadedMsg{Seq: seq, Snapshot: snap}
	}
}

// loadDetail fetches one application's detail.
func (m *Model) loadDetail(appUUID string) tea.Cmd {
	if m.cancelDetail != nil {
		m.cancelDetail()
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	m.cancelDetail = cancel

	m.detailSeq++
	seq := m.detailSeq
	svc := m.service

	return func() tea.Msg {
		defer cancel()
		d, err := svc.ApplicationDetail(ctx, appUUID)
		if err != nil {
			return detailFailedMsg{Seq: seq, AppUUID: appUUID, Err: domain.AsError(err)}
		}
		return detailLoadedMsg{Seq: seq, AppUUID: appUUID, Detail: d}
	}
}

// loadRuntimeLogs fetches one bounded log snapshot for the open application.
func (m *Model) loadRuntimeLogs(appUUID string) tea.Cmd {
	m.cancelRuntimeLogsRequest()
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	m.cancelRuntimeLogs = cancel

	m.runtimeLogsSeq++
	seq := m.runtimeLogsSeq
	service := m.service
	lines := m.opts.Config.LogLines

	return func() tea.Msg {
		defer cancel()
		snapshot, err := service.RuntimeLogs(ctx, appUUID, lines)
		if err != nil {
			return runtimeLogsFailedMsg{Seq: seq, AppUUID: appUUID, Err: domain.AsError(err)}
		}
		return runtimeLogsLoadedMsg{Seq: seq, AppUUID: appUUID, Snapshot: snapshot}
	}
}

func (m *Model) cancelRuntimeLogsRequest() {
	if m.cancelRuntimeLogs == nil {
		return
	}
	m.cancelRuntimeLogs()
	m.cancelRuntimeLogs = nil
}

func (m *Model) loadDeploymentLogs(deploymentUUID string) tea.Cmd {
	m.cancelDeploymentLogsRequest()
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	m.cancelDeploymentLogs = cancel
	m.deploymentLogsSeq++
	seq := m.deploymentLogsSeq
	service := m.service

	return func() tea.Msg {
		defer cancel()
		snapshot, err := service.DeploymentLogs(ctx, deploymentUUID)
		if err != nil {
			return deploymentLogsFailedMsg{Seq: seq, DeploymentUUID: deploymentUUID, Err: domain.AsError(err)}
		}
		return deploymentLogsLoadedMsg{Seq: seq, DeploymentUUID: deploymentUUID, Snapshot: snapshot}
	}
}

func (m *Model) cancelDeploymentLogsRequest() {
	if m.cancelDeploymentLogs == nil {
		return
	}
	m.cancelDeploymentLogs()
	m.cancelDeploymentLogs = nil
}

func (m *Model) runtimeLogsTick(appUUID string) tea.Cmd {
	interval := time.Duration(m.opts.Config.LogRefreshInterval)
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return runtimeLogsTickMsg{AppUUID: appUUID}
	})
}

// refreshTick schedules the next automatic refresh at the configured interval.
func (m *Model) refreshTick() tea.Cmd {
	interval := m.opts.Config.EffectiveRefreshInterval(activeInstance(m))
	return tea.Tick(interval, func(t time.Time) tea.Msg { return refreshTickMsg(t) })
}

// frameTick schedules the next animation frame.
func (m *Model) frameTick() tea.Cmd {
	return tea.Tick(frameInterval, func(t time.Time) tea.Msg { return frameTickMsg(t) })
}

// pushToast queues a notification.
func (m *Model) pushToast(kind components.ToastKind, text, detail string) tea.Cmd {
	return func() tea.Msg {
		return toastMsg{Kind: int(kind), Text: text, Detail: detail}
	}
}

// appendToast adds a notification to the stack, applying the lifetime and the
// stack limit.
func (m *Model) appendToast(kind components.ToastKind, text, detail string) {
	m.nextToast++
	m.toasts = append(m.toasts, components.Toast{
		ID:        m.nextToast,
		Kind:      kind,
		Text:      text,
		Detail:    detail,
		ExpiresAt: m.now().Add(components.ToastLifetime(kind)),
	})
	m.toasts = components.PruneToasts(m.toasts, m.now())
}

// openURL launches the OS browser. The URL has already been validated by the
// domain layer; platform.OpenURL validates the scheme again before exec.
func (m *Model) openURL(url string) tea.Cmd {
	return func() tea.Msg {
		if err := platform.OpenURL(url); err != nil {
			return openURLFailedMsg{Err: err}
		}
		return toastMsg{Kind: int(components.ToastInfo), Text: "Opened " + url}
	}
}

// shutdown cancels in-flight requests so no goroutine outlives the program.
func (m *Model) shutdown() {
	if m.cancelDashboard != nil {
		m.cancelDashboard()
		m.cancelDashboard = nil
	}
	if m.cancelDetail != nil {
		m.cancelDetail()
		m.cancelDetail = nil
	}
	m.cancelRuntimeLogsRequest()
	m.cancelDeploymentLogsRequest()
	if m.cancelOperation != nil {
		m.cancelOperation()
		m.cancelOperation = nil
	}
}
