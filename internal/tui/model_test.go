package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/views"
)

func TestLoadedDashboardRendersAcrossBreakpoints(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	for _, size := range []struct{ width, height int }{{60, 18}, {100, 28}, {160, 40}} {
		model := New(Options{
			Config:       config.Default(),
			Service:      service,
			InstanceName: "Demo",
			Demo:         true,
			ASCII:        true,
			Now:          func() time.Time { return now },
		})
		model.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
		model.loading = false
		model.connection = components.ConnectionOnline
		model.lastSuccess = now
		model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)

		plain := ansi.Strip(model.render())
		selected, ok := model.apps.Selected()
		if !ok || !strings.Contains(plain, selected.Name) {
			t.Fatalf("%dx%d render omitted application name:\n%s", size.width, size.height, plain)
		}
		for _, line := range strings.Split(plain, "\n") {
			if ansi.StringWidth(line) > size.width {
				t.Fatalf(
					"%dx%d render overflowed to %d cells: %q",
					size.width,
					size.height,
					ansi.StringWidth(line),
					line,
				)
			}
		}
	}
}

func TestDashboardErrorStateIsActionable(t *testing.T) {
	service := demo.New(demo.Options{})
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	model.loading = false
	model.lastError = domain.NewError(domain.ErrorNetwork, nil)

	plain := ansi.Strip(model.render())
	if !strings.Contains(plain, "Cannot reach Coolify") || !strings.Contains(plain, "retry") {
		t.Fatalf("error state is not actionable:\n%s", plain)
	}
}

func TestDetailMessageKeepsDeploymentHistory(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	dashboard, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	detail, err := service.ApplicationDetail(t.Context(), dashboard.Applications[0].UUID)
	if err != nil {
		t.Fatal(err)
	}
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.Update(detailLoadedMsg{Seq: model.detailSeq, Detail: detail})
	model.detail.SetTab(views.TabDeployments)

	plain := ansi.Strip(model.detail.Render(model.theme, 100, 20, now))
	if len(detail.Deployments) == 0 || !strings.Contains(plain, detail.Deployments[0].ShortCommit()) {
		t.Fatalf("detail message dropped deployment history:\n%s", plain)
	}
}

func TestRuntimeLogsCommandUpdatesDetail(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	dashboard, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	application := dashboard.Applications[0]
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.detail.SetApplication(application, now)
	model.detail.SetTab(views.TabRuntimeLogs)

	message := model.loadRuntimeLogs(application.UUID)()
	loaded, ok := message.(runtimeLogsLoadedMsg)
	if !ok {
		t.Fatalf("loadRuntimeLogs() message = %T", message)
	}
	model.Update(loaded)
	if !model.detail.RuntimeLogsLoaded() {
		t.Fatal("runtime log snapshot was not stored")
	}
	plain := ansi.Strip(model.detail.Render(model.theme, 100, 20, now))
	if !strings.Contains(plain, "RUNTIME LOGS") {
		t.Fatalf("runtime logs were not rendered:\n%s", plain)
	}
}

func TestRuntimeLogTickOnlyReloadsOpenLogTab(t *testing.T) {
	service := demo.New(demo.Options{})
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.screen = screenDetail
	model.detail.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	model.detail.SetTab(views.TabRuntimeLogs)

	_, command := model.Update(runtimeLogsTickMsg{AppUUID: "app-1"})
	if command == nil {
		t.Fatal("open runtime log tab did not schedule a refresh")
	}

	model.detail.SetTab(views.TabOverview)
	_, command = model.Update(runtimeLogsTickMsg{AppUUID: "app-1"})
	if command != nil {
		t.Fatal("closed runtime log tab kept polling")
	}
}

func TestLeavingDetailCancelsRuntimeLogRequest(t *testing.T) {
	service := demo.New(demo.Options{})
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.screen = screenDetail
	model.detail.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	cancelled := false
	model.cancelRuntimeLogs = func() { cancelled = true }

	model.handleDetailKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !cancelled || model.cancelRuntimeLogs != nil {
		t.Fatal("leaving detail did not cancel the runtime log request")
	}
}

func TestDeploymentLogsCommandUpdatesDetail(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	dashboard, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	detail, err := service.ApplicationDetail(t.Context(), dashboard.Applications[0].UUID)
	if err != nil {
		t.Fatal(err)
	}

	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.detail.SetApplication(detail.Application, now)
	model.detail.SetDeployments(detail.Deployments)
	model.detail.SetTab(views.TabDeployments)
	uuid, ok := model.detail.OpenSelectedDeploymentLogs()
	if !ok {
		t.Fatal("demo application has no deployment")
	}

	message := model.loadDeploymentLogs(uuid)()
	loaded, ok := message.(deploymentLogsLoadedMsg)
	if !ok {
		t.Fatalf("loadDeploymentLogs() message = %T", message)
	}
	model.Update(loaded)

	plain := ansi.Strip(model.detail.Render(model.theme, 100, 20, now))
	if !strings.Contains(plain, "DEPLOYMENT LOG") || !strings.Contains(plain, uuid) {
		t.Fatalf("deployment logs not rendered:\n%s", plain)
	}
}

func TestDeploymentHistoryOpensAndClosesSelectedLog(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.screen = screenDetail
	model.detail.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	model.detail.SetDeployments([]domain.Deployment{{UUID: "dep-1"}})
	model.detail.SetTab(views.TabDeployments)

	_, command := model.handleDetailKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if command == nil || !model.detail.DeploymentLogsOpen() {
		t.Fatal("enter did not open selected deployment log")
	}

	cancelled := false
	model.cancelDeploymentLogs = func() { cancelled = true }
	model.handleDetailKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.screen != screenDetail || model.detail.DeploymentLogsOpen() {
		t.Fatal("escape left detail instead of closing deployment log")
	}
	if !cancelled || model.cancelDeploymentLogs != nil {
		t.Fatal("closing deployment log did not cancel request")
	}
}

func TestRuntimeLogPauseStopsAndResumesPolling(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.screen = screenDetail
	model.detail.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	model.detail.SetTab(views.TabRuntimeLogs)
	cancelled := false
	model.cancelRuntimeLogs = func() { cancelled = true }

	space := tea.KeyPressMsg{Code: ' ', Text: " "}
	_, command := model.handleDetailKey(space)
	if command != nil || !cancelled || !model.detail.RuntimeLogsPaused() {
		t.Fatal("pause did not stop runtime log polling")
	}
	_, command = model.Update(runtimeLogsTickMsg{AppUUID: "app-1"})
	if command != nil {
		t.Fatal("paused runtime logs accepted a polling tick")
	}

	_, command = model.handleDetailKey(space)
	if command == nil || model.detail.RuntimeLogsPaused() {
		t.Fatal("resume did not restart runtime log polling")
	}
}

func TestDetailLogShortcutsOpenTheirViews(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.screen = screenDetail
	model.detail.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	model.detail.SetDeployments([]domain.Deployment{{UUID: "dep-1"}})

	_, command := model.handleDetailKey(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if command == nil || model.detail.Tab() != views.TabRuntimeLogs {
		t.Fatal("l did not open runtime logs")
	}

	_, command = model.handleDetailKey(tea.KeyPressMsg{Code: 'L', Text: "L"})
	if command == nil || model.detail.Tab() != views.TabDeployments || !model.detail.DeploymentLogsOpen() {
		t.Fatal("L did not open selected deployment log")
	}
}

func TestRuntimeLogSearchConsumesTextAndAppliesQuery(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.screen = screenDetail
	model.detail.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	model.detail.SetTab(views.TabRuntimeLogs)
	model.detail.SetRuntimeLogs([]domain.LogLine{{Text: "ERROR first"}, {Text: "error second"}}, time.Now(), false)

	model.handleDetailKey(tea.KeyPressMsg{Code: '/', Text: "/"})
	if !model.logSearching {
		t.Fatal("/ did not open runtime log search")
	}
	model.handleKey(tea.KeyPressMsg{Code: 'e', Text: "e"})
	model.handleKey(tea.KeyPressMsg{Code: 'r', Text: "r"})
	model.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if model.logSearching || model.detail.RuntimeSearch() != "er" {
		t.Fatalf("runtime search = %q, editing = %v", model.detail.RuntimeSearch(), model.logSearching)
	}
}

func TestMutationRequiresConfirmationBeforeRunning(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.loading = false
	model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)

	_, command := model.handleListKey(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if command != nil || model.pendingAction == nil {
		t.Fatal("deploy ran without opening confirmation")
	}
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	plain := ansi.Strip(model.render())
	selected, _ := model.apps.Selected()
	if !strings.Contains(plain, "Deploy application?") || !strings.Contains(plain, selected.Name) {
		t.Fatalf("confirmation modal missing action context:\n%s", plain)
	}
	model.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.pendingAction != nil {
		t.Fatal("escape did not cancel confirmation")
	}

	model.handleListKey(tea.KeyPressMsg{Code: 'd', Text: "d"})
	_, command = model.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if command == nil || model.pendingAction != nil {
		t.Fatal("confirm did not start deploy")
	}
	message := command()
	completed, ok := message.(actionCompletedMsg)
	if !ok || completed.Result.Operation != "deploy" {
		t.Fatalf("deploy command returned %T (%+v)", message, message)
	}
}

func TestStartStopActionFollowsApplicationState(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	running := domain.Application{UUID: "running", Name: "Running", Status: domain.ParseStatus("running:healthy")}
	model.apps.SetApplications([]domain.Application{running}, nil, time.Now())
	model.handleListKey(tea.KeyPressMsg{Code: 's', Text: "s"})
	if model.pendingAction == nil || model.pendingAction.kind != actionStop {
		t.Fatal("running application did not stage stop")
	}

	stopped := domain.Application{UUID: "stopped", Name: "Stopped", Status: domain.ParseStatus("exited")}
	model.apps.SetApplications([]domain.Application{stopped}, nil, time.Now())
	model.pendingAction = nil
	model.handleListKey(tea.KeyPressMsg{Code: 's', Text: "s"})
	if model.pendingAction == nil || model.pendingAction.kind != actionStart {
		t.Fatal("stopped application did not stage start")
	}
}
