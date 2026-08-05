package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

// deploymentOutcomes compares a freshly loaded snapshot against the statuses
// seen on the previous refresh and announces the deployments that just settled.
// The dashboard already polls this data, so noticing the transition costs one
// map instead of another request.
//
// Only a transition observed here counts. The first snapshot after startup or
// an instance switch is full of history that settled long ago, and replaying it
// as notifications would be worse than saying nothing.
func (m *Model) deploymentOutcomes(snapshot app.DashboardSnapshot) []tea.Cmd {
	seen := make(map[string]domain.DeploymentStatus,
		len(snapshot.ActiveDeployments)+len(snapshot.RecentDeployments))

	var cmds []tea.Cmd
	for _, list := range [][]domain.Deployment{snapshot.ActiveDeployments, snapshot.RecentDeployments} {
		for _, d := range list {
			if d.UUID == "" {
				continue
			}
			if _, dup := seen[d.UUID]; dup {
				// A deployment can appear in both lists; the first wins.
				continue
			}
			seen[d.UUID] = d.Status

			prev, known := m.deployStates[d.UUID]
			if !known || !prev.IsActive() || d.Status.IsActive() {
				continue
			}
			if cmd := m.deploymentOutcomeToast(d); cmd != nil {
				cmds = append(cmds, cmd)
			}
			// The deployment has settled, so the session no longer owns it.
			delete(m.ownDeploys, d.UUID)
		}
	}

	m.deployStates = seen
	return cmds
}

// deploymentOutcomeToast is the notification for one settled deployment, or nil
// when the outcome is not worth interrupting for. A failure always speaks up
// because it is the whole reason to watch. A success only does when this
// session triggered it: on an instance that CI and colleagues also deploy to,
// announcing every green build buries the one red one.
func (m *Model) deploymentOutcomeToast(d domain.Deployment) tea.Cmd {
	name := d.ApplicationName
	if name == "" {
		name = d.ApplicationUUID
	}

	switch d.Status {
	case domain.DeploymentFailed:
		return m.pushToast(components.ToastError, "Deploy failed", name+" - press 2 for the queue")
	case domain.DeploymentCancelled:
		if !m.ownDeploys[d.UUID] {
			return nil
		}
		return m.pushToast(components.ToastWarning, "Deploy cancelled", name)
	case domain.DeploymentFinished:
		if !m.ownDeploys[d.UUID] {
			return nil
		}
		return m.pushToast(components.ToastSuccess, "Deploy finished", name)
	default:
		return nil
	}
}

// forgetDeploymentStates drops the watch state so a different instance cannot
// inherit another fleet's deployments.
func (m *Model) forgetDeploymentStates() {
	m.deployStates = map[string]domain.DeploymentStatus{}
	m.ownDeploys = map[string]bool{}
}
