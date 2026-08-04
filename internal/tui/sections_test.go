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
)

func TestHelpOverlayOpensAndCloses(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 28})

	model.handleKey(tea.KeyPressMsg{Code: '?', Text: "?"})
	if !model.helpOpen {
		t.Fatal("? did not open help")
	}
	plain := ansi.Strip(model.render())
	if !strings.Contains(plain, "help") || !strings.Contains(plain, "command palette") {
		t.Fatalf("help overlay missing content:\n%s", plain)
	}
	model.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.helpOpen {
		t.Fatal("esc did not close help")
	}
}

func TestSectionNavigationKeys(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	model := New(Options{
		Config: config.Default(), Service: service, Demo: true, ASCII: true,
		Now: func() time.Time { return now },
	})
	model.loading = false
	model.connection = components.ConnectionOnline
	model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
	model.deployments.SetItems(snapshot.ActiveDeployments, now)
	model.refreshInstances()
	model.refreshDiagnostics()
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 32})

	model.handleKey(tea.KeyPressMsg{Code: '2', Text: "2"})
	if model.section != SectionDeployments {
		t.Fatalf("section = %v, want deployments", model.section)
	}
	plain := ansi.Strip(model.render())
	if !strings.Contains(plain, "DEPLOYMENTS") {
		t.Fatalf("deployments view not rendered:\n%s", plain)
	}

	model.handleKey(tea.KeyPressMsg{Code: '3', Text: "3"})
	if model.section != SectionInstances {
		t.Fatalf("section = %v, want instances", model.section)
	}
	plain = ansi.Strip(model.render())
	if !strings.Contains(plain, "INSTANCES") {
		t.Fatalf("instances view not rendered:\n%s", plain)
	}

	model.handleKey(tea.KeyPressMsg{Code: '4', Text: "4"})
	if model.section != SectionDiagnostics {
		t.Fatalf("section = %v, want diagnostics", model.section)
	}
	plain = ansi.Strip(model.render())
	if !strings.Contains(plain, "DIAGNOSTICS") {
		t.Fatalf("diagnostics view not rendered:\n%s", plain)
	}

	model.handleKey(tea.KeyPressMsg{Code: '1', Text: "1"})
	if model.section != SectionApplications {
		t.Fatalf("section = %v, want applications", model.section)
	}
}

func TestDeploymentsEnterOpensBuildLog(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.ActiveDeployments) == 0 {
		// Seed a synthetic active deployment so the test is deterministic.
		snapshot.ActiveDeployments = []domain.Deployment{{
			UUID:            "dep-active",
			ApplicationUUID: snapshot.Applications[0].UUID,
			ApplicationName: snapshot.Applications[0].Name,
			Status:          domain.DeploymentInProgress,
			CommitSHA:       "abcdef1",
			CreatedAt:       now,
		}}
	}
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true, Now: func() time.Time { return now }})
	model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
	model.deployments.SetItems(snapshot.ActiveDeployments, now)
	model.section = SectionDeployments

	_, cmd := model.handleDeploymentsKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if model.screen != screenDetail || model.section != SectionApplications {
		t.Fatalf("enter did not open detail: screen=%v section=%v", model.screen, model.section)
	}
	if !model.detail.DeploymentLogsOpen() {
		t.Fatal("enter did not open deployment logs")
	}
	if cmd == nil {
		t.Fatal("expected load commands")
	}
}

func TestDiagnosticsExportAndCopy(t *testing.T) {
	model := New(Options{
		Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true,
		ConfigPath: "/tmp/cooldeck-test.toml",
		LogPath:    "/tmp/cooldeck-test.log",
	})
	model.refreshDiagnostics()
	text := model.diagnostics.PlainText()
	if !strings.Contains(text, "Version:") || !strings.Contains(text, "cooldeck") {
		t.Fatalf("plain diagnostics missing fields:\n%s", text)
	}
	if cmd := model.copyDiagnostics(); cmd == nil {
		t.Fatal("copy diagnostics returned nil")
	}
	if cmd := model.exportDiagnostics(); cmd == nil {
		t.Fatal("export diagnostics returned nil")
	}
	// Execute export in a hermetic state dir.
	t.Setenv("COOLDECK_STATE_DIR", t.TempDir())
	msg := model.exportDiagnostics()()
	toast, ok := msg.(toastMsg)
	if !ok || toast.Kind != int(components.ToastSuccess) {
		t.Fatalf("export result = %T %+v", msg, msg)
	}
	if toast.Detail == "" {
		t.Fatal("export toast missing path")
	}
}

func TestSidebarListsAllSections(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	items := model.sections()
	if len(items) != 4 {
		t.Fatalf("sections = %d, want 4", len(items))
	}
	labels := []string{items[0].Label, items[1].Label, items[2].Label, items[3].Label}
	for i, want := range []string{"Applications", "Deployments", "Instances", "Diagnostics"} {
		if labels[i] != want {
			t.Fatalf("section %d = %q, want %q", i, labels[i], want)
		}
	}
}
