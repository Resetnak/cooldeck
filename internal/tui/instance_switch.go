package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/views"
)

// switchInstance rebuilds the session against another configured instance.
func (m *Model) switchInstance(id string) tea.Cmd {
	if m.opts.Demo {
		return m.pushToast(components.ToastWarning, "Cannot switch in demo mode", "Restart without --demo to use real instances.")
	}
	if m.opts.OpenService == nil {
		return m.pushToast(components.ToastInfo, "Switch instance", "Restart with: cooldeck --instance "+id)
	}
	if id == "" || id == m.service.InstanceID() {
		return m.pushToast(components.ToastInfo, "Already active", "This instance is already selected.")
	}
	if m.operationInFlight {
		return m.pushToast(components.ToastWarning, "Action already running", "Wait for the current request to finish.")
	}

	// Cancel in-flight work bound to the outgoing service.
	m.shutdown()
	m.operationInFlight = true
	m.connection = components.ConnectionConnecting
	m.loading = true

	open := m.opts.OpenService
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()
		svc, name, err := open(ctx, id)
		if err != nil {
			return instanceSwitchFailedMsg{InstanceID: id, Err: domain.AsError(err)}
		}
		return instanceSwitchedMsg{Service: svc, InstanceID: id, InstanceName: name}
	}
}

func (m *Model) applyInstanceSwitch(msg instanceSwitchedMsg) tea.Cmd {
	m.operationInFlight = false
	m.service = msg.Service
	m.opts.InstanceName = msg.InstanceName
	m.opts.Service = msg.Service

	// Drop cached data from the previous instance so the UI never mixes fleets.
	m.apps = views.NewApplications()
	m.detail = views.NewDetail()
	m.deployments = views.NewDeployments()
	m.forgetDeploymentStates()
	m.screen = screenList
	m.section = SectionApplications
	m.focus = focusContent
	m.filtering = false
	m.filterText = ""
	m.logSearching = false
	m.logSearchText = ""
	m.lastError = nil
	m.lastSuccess = time.Time{}
	m.connectionErr = nil
	m.coolifyVer = ""
	m.lastConnection = app.Connection{}
	m.capabilities = app.FullCapabilities()
	m.connection = components.ConnectionConnecting
	m.loading = true
	m.refreshing = false

	m.refreshInstances()
	m.refreshDiagnostics()

	return tea.Batch(
		m.pushToast(components.ToastSuccess, "Switched instance", msg.InstanceName),
		m.connect(),
	)
}

func (m *Model) applyInstanceSwitchFailed(msg instanceSwitchFailedMsg) tea.Cmd {
	m.operationInFlight = false
	m.loading = false
	m.connection = connectionStateFor(msg.Err)
	m.rememberError(msg.Err)
	m.refreshInstances()
	m.refreshDiagnostics()
	return m.pushToast(components.ToastError, "Could not switch instance", msg.Err.Message)
}

// runDeleteInstance removes a local config entry and optionally its keyring token.
func (m *Model) runDeleteInstance(action pendingAction) tea.Cmd {
	id := action.instanceID
	wasActive := id == m.service.InstanceID()

	inst, err := m.opts.Config.Instance(id)
	if err != nil {
		return m.pushToast(components.ToastError, "Delete failed", err.Error())
	}

	// Snapshot enough state to roll back the in-memory config if Save fails.
	prevDefault := m.opts.Config.DefaultInstance
	prevInstances := cloneInstances(m.opts.Config.Instances)

	if err := m.opts.Config.RemoveInstance(id); err != nil {
		return m.pushToast(components.ToastError, "Delete failed", err.Error())
	}
	if m.opts.SaveConfig != nil {
		if err := m.opts.SaveConfig(m.opts.Config); err != nil {
			m.opts.Config.Instances = prevInstances
			m.opts.Config.DefaultInstance = prevDefault
			return m.pushToast(components.ToastError, "Could not write config", err.Error())
		}
	}
	if m.opts.RemoveCredentials != nil {
		if err := m.opts.RemoveCredentials(inst); err != nil {
			m.refreshInstances()
			return m.pushToast(components.ToastWarning,
				"Instance removed, credential cleanup failed", err.Error())
		}
	}

	m.refreshInstances()
	m.refreshDiagnostics()

	// Defer the follow-up (toast + optional switch) to a message so switchInstance
	// is not started while building a tea.Batch — that would set operationInFlight
	// before the event loop runs the command.
	next := ""
	empty := false
	if wasActive {
		next = m.opts.Config.DefaultInstance
		if next == "" {
			if ids := m.opts.Config.InstanceIDs(); len(ids) > 0 {
				next = ids[0]
			} else {
				empty = true
			}
		}
	}
	name := action.instanceName
	return func() tea.Msg {
		return instanceRemovedMsg{Name: name, NextID: next, EmptyFleet: empty}
	}
}

func cloneInstances(in map[string]config.Instance) map[string]config.Instance {
	out := make(map[string]config.Instance, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// activateSelectedInstance switches to the highlighted instances-list row.
func (m *Model) activateSelectedInstance() tea.Cmd {
	row, ok := m.instances.Selected()
	if !ok {
		return nil
	}
	if row.Active {
		return m.pushToast(components.ToastInfo, "Already active", row.Name+" is the current instance.")
	}
	if row.Demo {
		return m.pushToast(components.ToastInfo, "Demo instance", "Restart without --demo to use a real Coolify instance.")
	}
	if m.opts.OpenService == nil {
		return m.describeInstanceSwitch(row)
	}
	return m.switchInstance(row.ID)
}
