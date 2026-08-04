package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

func TestDetailRendersDeploymentHistory(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	view := NewDetail()
	view.SetApplication(domain.Application{UUID: "app-1", Name: "API"}, now)
	view.SetDeployments([]domain.Deployment{{
		UUID:          "dep-1",
		Status:        domain.DeploymentFinished,
		CommitSHA:     "1234567890",
		CommitMessage: "Ship deployment history",
		Trigger:       "api",
		CreatedAt:     now.Add(-time.Minute),
		UpdatedAt:     now,
	}})
	view.SetTab(TabDeployments)

	plain := ansi.Strip(view.Render(testTheme(), 100, 20, now))
	for _, want := range []string{"DEPLOYMENTS", "Finished", "1234567", "Ship deployment history", "api"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("deployment view omitted %q:\n%s", want, plain)
		}
	}
}

func TestDetailRendersRuntimeLogs(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	view := NewDetail()
	view.SetApplication(domain.Application{UUID: "app-1", Name: "API"}, now)
	view.SetRuntimeLogs([]domain.LogLine{
		domain.ParseLogLine("2026-08-04T11:59:59Z INFO server ready"),
		domain.ParseLogLine("2026-08-04T12:00:00Z ERROR database unavailable"),
	}, now, false)
	view.SetTab(TabRuntimeLogs)

	plain := ansi.Strip(view.Render(testTheme(), 100, 20, now))
	for _, want := range []string{"RUNTIME LOGS", "INFO", "server ready", "ERROR", "database unavailable"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("runtime log view omitted %q:\n%s", want, plain)
		}
	}
}

func TestDetailMovesDeploymentSelection(t *testing.T) {
	view := NewDetail()
	view.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	view.SetDeployments([]domain.Deployment{{UUID: "dep-1"}, {UUID: "dep-2"}})
	view.SetTab(TabDeployments)

	selected, ok := view.SelectedDeployment()
	if !ok || selected.UUID != "dep-1" {
		t.Fatalf("initial selection = %#v, %v", selected, ok)
	}
	view.MoveDeployment(1)
	selected, _ = view.SelectedDeployment()
	if selected.UUID != "dep-2" {
		t.Fatalf("selection after move = %q", selected.UUID)
	}
	view.MoveDeployment(10)
	selected, _ = view.SelectedDeployment()
	if selected.UUID != "dep-2" {
		t.Fatalf("selection escaped end = %q", selected.UUID)
	}
}

func TestDetailRendersSelectedDeploymentLogs(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	view := NewDetail()
	view.SetApplication(domain.Application{UUID: "app-1"}, now)
	view.SetDeployments([]domain.Deployment{{UUID: "dep-1", CommitSHA: "1234567890"}})
	view.SetTab(TabDeployments)
	uuid, ok := view.OpenSelectedDeploymentLogs()
	if !ok || uuid != "dep-1" {
		t.Fatalf("opened deployment = %q, %v", uuid, ok)
	}
	view.SetDeploymentLogs(uuid, []domain.LogLine{
		domain.ParseLogLine("2026-08-04T12:00:00Z INFO image pushed"),
	}, now, false)

	plain := ansi.Strip(view.Render(testTheme(), 100, 20, now))
	for _, want := range []string{"DEPLOYMENT LOG", "dep-1", "image pushed"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("deployment log view omitted %q:\n%s", want, plain)
		}
	}
}

func TestDetailRuntimeLogControls(t *testing.T) {
	view := NewDetail()
	view.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	view.SetTab(TabRuntimeLogs)
	view.SetRuntimeLogs([]domain.LogLine{
		{Text: "first"},
		{Text: "second"},
		{Text: "third"},
		{Text: "fourth"},
	}, time.Now(), false)
	th := theme.New(theme.Options{ASCII: true})

	view.Render(th, 40, 4, time.Now())
	view.Scroll(-1)
	if view.RuntimeLogsFollowing() {
		t.Fatal("manual upward scroll kept follow enabled")
	}
	view.ToggleRuntimeFollow()
	if !view.RuntimeLogsFollowing() {
		t.Fatal("follow toggle did not resume following")
	}
	view.ToggleRuntimeWrap()
	if !view.RuntimeLogsWrapping() {
		t.Fatal("wrap toggle did not enable wrapping")
	}
	view.ToggleRuntimePause()
	if !view.RuntimeLogsPaused() {
		t.Fatal("pause toggle did not pause logs")
	}
}

func TestDetailSearchesLoadedRuntimeLogs(t *testing.T) {
	view := NewDetail()
	view.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	view.SetTab(TabRuntimeLogs)
	view.SetRuntimeLogs([]domain.LogLine{
		{Text: "server ready"},
		{Text: "ERROR first"},
		{Text: "still running"},
		{Text: "error second"},
	}, time.Now(), false)

	view.SetRuntimeSearch("error")
	current, total := view.RuntimeSearchStatus()
	if current != 1 || total != 2 || view.RuntimeLogsFollowing() {
		t.Fatalf("search status = %d/%d, following = %v", current, total, view.RuntimeLogsFollowing())
	}
	view.NextRuntimeMatch(1)
	current, total = view.RuntimeSearchStatus()
	if current != 2 || total != 2 {
		t.Fatalf("next search status = %d/%d", current, total)
	}
}

func testTheme() *theme.Theme {
	return theme.New(theme.Options{Mode: theme.ModeDark, ASCII: true})
}
