package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
)

func TestFleetSnapshotMarkdown(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	conn := app.Connection{InstanceName: "prod", Version: "4.0.0"}
	apps := []domain.Application{
		{Name: "api", Status: domain.ParseStatus("running:healthy"), Branch: "main"},
		{Name: "worker | pipe", Status: domain.ParseStatus("exited"), Branch: "main"},
	}
	deployments := []domain.Deployment{
		{ApplicationName: "api", Status: domain.DeploymentFinished,
			CreatedAt: now.Add(-10 * time.Minute), CommitMessage: "fix: \x1b[31mred\x1b[0m things"},
	}

	got := fleetSnapshotMarkdown(conn, apps, deployments, []string{"Timeout: instance slow"}, now)

	for _, want := range []string{
		"# Fleet snapshot",
		"prod (Coolify 4.0.0)",
		"| api | running:healthy | main |",
		"worker \\| pipe",
		"## Recent deployments",
		"## Recent errors",
		"Timeout: instance slow",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("snapshot missing %q\n---\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b") {
		t.Error("snapshot carries a raw escape sequence")
	}
}
