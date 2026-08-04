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
	actionDeploy  actionKind = "deploy"
	actionRestart actionKind = "restart"
	actionStart   actionKind = "start"
	actionStop    actionKind = "stop"
)

type pendingAction struct {
	kind  actionKind
	app   domain.Application
	force bool
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

func (m *Model) actionAllowed(kind actionKind) bool {
	switch kind {
	case actionDeploy:
		return m.capabilities.Deploy
	case actionRestart:
		return m.capabilities.Restart
	default:
		return m.capabilities.StartStop
	}
}

func (m *Model) runAction(action pendingAction) tea.Cmd {
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
		m.operationInFlight = true
		return m, m.runAction(action)
	}
	return m, nil
}

func (a pendingAction) title() string {
	verb := strings.ToUpper(string(a.kind[:1])) + string(a.kind[1:])
	if a.force {
		return "Force " + strings.ToLower(verb) + " application?"
	}
	return verb + " application?"
}
