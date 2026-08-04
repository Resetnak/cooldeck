package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

type actionKind string

const (
	actionDeploy         actionKind = "deploy"
	actionRestart        actionKind = "restart"
	actionStart          actionKind = "start"
	actionStop           actionKind = "stop"
	actionDeleteInstance actionKind = "delete-instance"
)

type pendingAction struct {
	kind  actionKind
	app   domain.Application
	force bool

	// Instance fields are set for local config mutations.
	instanceID   string
	instanceName string
}

func (m *Model) stageAction(kind actionKind, force bool) tea.Cmd {
	if m.operationInFlight {
		return m.pushToast(components.ToastWarning, "Action already running", "Wait for the current request to finish.")
	}
	application, ok := m.currentApplication()
	if !ok {
		return nil
	}
	if !m.actionAllowed(kind) {
		return m.pushToast(components.ToastWarning, "Action unavailable", "The token does not have permission for this action.")
	}
	m.pendingAction = &pendingAction{kind: kind, app: application, force: force}
	return nil
}

func (m *Model) stageDeleteInstance(id, name string) tea.Cmd {
	if m.operationInFlight {
		return m.pushToast(components.ToastWarning, "Action already running", "Wait for the current request to finish.")
	}
	if m.opts.Demo {
		return m.pushToast(components.ToastWarning, "Cannot delete demo instance", "Demo mode has no local configuration.")
	}
	if m.opts.SaveConfig == nil {
		return m.pushToast(components.ToastWarning, "Cannot delete instance", "Config writing is not available in this session.")
	}
	if id == "" {
		return nil
	}
	m.pendingAction = &pendingAction{
		kind:         actionDeleteInstance,
		instanceID:   id,
		instanceName: name,
	}
	return nil
}

func (m *Model) actionAllowed(kind actionKind) bool {
	switch kind {
	case actionDeploy:
		return m.capabilities.Deploy
	case actionRestart:
		return m.capabilities.Restart
	case actionDeleteInstance:
		return m.opts.SaveConfig != nil && !m.opts.Demo
	default:
		return m.capabilities.StartStop
	}
}

func (m *Model) runAction(action pendingAction) tea.Cmd {
	if action.kind == actionDeleteInstance {
		return m.runDeleteInstance(action)
	}
	if m.cancelOperation != nil {
		m.cancelOperation()
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	m.cancelOperation = cancel
	m.operationSeq++
	seq := m.operationSeq
	service := m.service

	return func() tea.Msg {
		defer cancel()
		var result app.OperationResult
		var err error
		switch action.kind {
		case actionDeploy:
			result, err = service.Deploy(ctx, action.app.UUID, app.DeployOptions{Force: action.force})
		case actionRestart:
			result, err = service.Restart(ctx, action.app.UUID)
		case actionStart:
			result, err = service.Start(ctx, action.app.UUID)
		case actionStop:
			result, err = service.Stop(ctx, action.app.UUID)
		}
		if err != nil {
			return actionFailedMsg{Seq: seq, Err: domain.AsError(err)}
		}
		return actionCompletedMsg{Seq: seq, Result: result}
	}
}

func (m *Model) handleConfirmationKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.pendingAction == nil {
		return m, nil
	}
	switch msg.String() {
	case "esc", "q":
		m.pendingAction = nil
		return m, nil
	case "enter":
		action := *m.pendingAction
		m.pendingAction = nil
		if action.kind == actionDeleteInstance {
			// Local config work; no remote "in flight" spinner needed beyond the toast.
			return m, m.runDeleteInstance(action)
		}
		m.operationInFlight = true
		return m, m.runAction(action)
	}
	return m, nil
}

func (a pendingAction) title() string {
	if a.kind == actionDeleteInstance {
		return "Delete local instance?"
	}
	verb := strings.ToUpper(string(a.kind[:1])) + string(a.kind[1:])
	if a.force {
		return "Force " + strings.ToLower(verb) + " application?"
	}
	return verb + " application?"
}

func (a pendingAction) body() string {
	if a.kind == actionDeleteInstance {
		name := a.instanceName
		if name == "" {
			name = a.instanceID
		}
		return "Instance: " + name + "\n" +
			"Removes the local config entry and optional keyring token.\n" +
			"Coolify itself is not modified."
	}
	name := a.app.Name
	body := "Application: " + name
	if a.force {
		body += "\nThis bypasses Coolify's normal deployment cache."
	}
	return body
}
