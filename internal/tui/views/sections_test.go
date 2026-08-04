package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

func TestDeploymentsRendersActiveRows(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	view := NewDeployments()
	view.SetItems([]domain.Deployment{
		{
			UUID:            "dep-1",
			ApplicationName: "api",
			Status:          domain.DeploymentInProgress,
			CommitSHA:       "abcdef123",
			Trigger:         "api",
			CreatedAt:       now.Add(-time.Minute),
		},
		{
			UUID:            "dep-2",
			ApplicationName: "web",
			Status:          domain.DeploymentFinished,
			CommitSHA:       "deadbeef",
			Trigger:         "git",
			CreatedAt:       now.Add(-time.Hour),
		},
	}, now)

	plain := ansi.Strip(view.Render(testTheme(), 100, 16, true, now))
	for _, want := range []string{"DEPLOYMENTS", "api", "abcdef1", "web", "recent history"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("deployments omitted %q:\n%s", want, plain)
		}
	}
	view.ToggleActiveOnly()
	if view.Count() != 1 {
		t.Fatalf("active-only count = %d", view.Count())
	}
	plain = ansi.Strip(view.Render(testTheme(), 100, 16, true, now))
	if strings.Contains(plain, "web") || !strings.Contains(plain, "active only") {
		t.Fatalf("active filter failed:\n%s", plain)
	}
}

func TestInstancesRendersActiveRow(t *testing.T) {
	view := NewInstances()
	view.SetItems([]InstanceRow{{
		ID:          "prod",
		Name:        "Production",
		URL:         "https://coolify.example",
		Active:      true,
		Version:     "4.0.0",
		Connection:  components.ConnectionOnline,
		Latency:     12 * time.Millisecond,
		TokenSource: "keyring",
	}})
	plain := ansi.Strip(view.Render(testTheme(), 100, 20, true, time.Now()))
	for _, want := range []string{"INSTANCES", "Production", "coolify.example", "online", "4.0.0"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("instances omitted %q:\n%s", want, plain)
		}
	}
}

func TestDiagnosticsPlainTextAndRender(t *testing.T) {
	view := NewDiagnostics()
	view.SetFields([]DiagnosticField{
		{Section: "cooldeck", Label: "Version", Value: "1.0.0"},
		{Label: "Go", Value: "go1.26"},
		{Section: "instance", Label: "URL", Value: "https://example"},
	})
	text := view.PlainText()
	if !strings.Contains(text, "## cooldeck") || !strings.Contains(text, "Version: 1.0.0") {
		t.Fatalf("plain text unexpected:\n%s", text)
	}
	plain := ansi.Strip(view.Render(testTheme(), 80, 20))
	if !strings.Contains(plain, "DIAGNOSTICS") || !strings.Contains(plain, "Version") {
		t.Fatalf("render unexpected:\n%s", plain)
	}
}
