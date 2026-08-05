package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

func newWatchModel() *Model {
	return New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
}

func activeSnapshot(uuid string) app.DashboardSnapshot {
	return app.DashboardSnapshot{ActiveDeployments: []domain.Deployment{
		{UUID: uuid, ApplicationName: "api", Status: domain.DeploymentInProgress},
	}}
}

func settledSnapshot(uuid string, status domain.DeploymentStatus) app.DashboardSnapshot {
	return app.DashboardSnapshot{RecentDeployments: []domain.Deployment{
		{UUID: uuid, ApplicationName: "api", Status: status},
	}}
}

// onlyToast runs the commands and returns the single toast they produced.
func onlyToast(t *testing.T, cmds []tea.Cmd) toastMsg {
	t.Helper()
	if len(cmds) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(cmds))
	}
	toast, ok := cmds[0]().(toastMsg)
	if !ok {
		t.Fatalf("command did not produce a toast")
	}
	return toast
}

func TestDeploymentFailureNotifiesOnceOnTransition(t *testing.T) {
	model := newWatchModel()

	// History loaded at startup has settled long ago and must stay silent.
	if cmds := model.deploymentOutcomes(settledSnapshot("d1", domain.DeploymentFailed)); len(cmds) != 0 {
		t.Fatalf("first snapshot notified %d times, want 0", len(cmds))
	}

	model = newWatchModel()
	if cmds := model.deploymentOutcomes(activeSnapshot("d1")); len(cmds) != 0 {
		t.Fatalf("in-flight deployment notified %d times, want 0", len(cmds))
	}

	toast := onlyToast(t, model.deploymentOutcomes(settledSnapshot("d1", domain.DeploymentFailed)))
	if toast.Kind != int(components.ToastError) {
		t.Fatalf("toast kind = %d, want error", toast.Kind)
	}

	// The status does not change again, so neither should the notification.
	if cmds := model.deploymentOutcomes(settledSnapshot("d1", domain.DeploymentFailed)); len(cmds) != 0 {
		t.Fatalf("repeat snapshot notified %d times, want 0", len(cmds))
	}
}

func TestDeploymentSuccessNotifiesOnlyForOwnDeploys(t *testing.T) {
	model := newWatchModel()

	// Someone else's green build is noise.
	model.deploymentOutcomes(activeSnapshot("d1"))
	if cmds := model.deploymentOutcomes(settledSnapshot("d1", domain.DeploymentFinished)); len(cmds) != 0 {
		t.Fatalf("foreign success notified %d times, want 0", len(cmds))
	}

	// A deploy triggered here is claimed when Coolify accepts it.
	model.Update(actionCompletedMsg{
		Seq:    model.operationSeq,
		Result: app.OperationResult{Operation: "deploy", DeploymentUUID: "d2"},
	})
	if !model.ownDeploys["d2"] {
		t.Fatal("queued deployment was not claimed by the session")
	}

	model.deploymentOutcomes(activeSnapshot("d2"))
	toast := onlyToast(t, model.deploymentOutcomes(settledSnapshot("d2", domain.DeploymentFinished)))
	if toast.Kind != int(components.ToastSuccess) {
		t.Fatalf("toast kind = %d, want success", toast.Kind)
	}

	// Ownership ends with the deployment, so the map cannot grow all session.
	if model.ownDeploys["d2"] {
		t.Fatal("settled deployment is still claimed")
	}
}

func TestInstanceSwitchForgetsDeploymentStates(t *testing.T) {
	model := newWatchModel()
	model.deploymentOutcomes(activeSnapshot("d1"))
	model.ownDeploys["d1"] = true

	model.forgetDeploymentStates()

	// Without the reset, a deployment UUID reused by another fleet would be
	// reported as a transition it never made.
	if cmds := model.deploymentOutcomes(settledSnapshot("d1", domain.DeploymentFailed)); len(cmds) != 0 {
		t.Fatalf("stale state notified %d times, want 0", len(cmds))
	}
}
