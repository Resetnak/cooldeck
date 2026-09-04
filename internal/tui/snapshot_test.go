package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/logging"
)

func TestFleetSnapshotMarkdown(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	conn := app.Connection{InstanceName: "prod", Version: "4.0.0"}
	apps := []domain.Application{
		{Name: "api", Status: domain.ParseStatus("running:healthy"), Branch: "main"},
		{Name: "worker | pipe", Status: domain.ParseStatus("exited"), Branch: "main"},
	}
	deployments := []domain.Deployment{
		{ApplicationName: "api", Status: domain.DeploymentFinished, UUID: "3fa85f64-5717-4562-b3fc-2c963f66afa6",
			CreatedAt: now.Add(-10 * time.Minute), CommitMessage: "fix: \x1b[31mred\x1b[0m things"},
	}

	got := fleetSnapshotMarkdown(conn, apps, deployments, []string{"Timeout: instance slow"}, now)

	for _, want := range []string{
		"# Fleet snapshot",
		"prod (Coolify 4.0.0)",
		"| api | running:healthy | main |",
		"worker \\| pipe",
		"## Recent deployments",
		// The UUID is the handle that makes a pasted deployment line actionable.
		"(3fa85f64-5717-4562-b3fc-2c963f66afa6)",
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

// TestFleetSnapshotRedactsSeededSecrets seeds a credential into every field
// that reaches the snapshot and asserts none of them survive the copy. The
// snapshot is meant to be pasted into a chat window or at an AI assistant, so
// "the fields happen to be safe today" is not a strong enough guarantee.
func TestFleetSnapshotRedactsSeededSecrets(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	const sanctum = "7|qA3xZk9LmPb2NrTvWy8Hc4Ju6Ef1Sd0Gh5Ki2Lo"
	const jwt = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dBjftJeZ4CVPmB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	seeded := map[string]string{
		"sanctum token":  sanctum,
		"jwt":            jwt,
		"env value":      "s3cr3t-database-password",
		"basic auth url": "hunter2",
		"bearer header":  "AbCdEfGh12345678",
	}

	conn := app.Connection{InstanceName: "prod token=" + sanctum, Version: "4.0.0"}
	apps := []domain.Application{{
		Name:   "api",
		Status: domain.ParseStatus("running:healthy"),
		Branch: "feat/rotate-DB_PASSWORD=s3cr3t-database-password",
	}}
	deployments := []domain.Deployment{{
		ApplicationName: "api",
		Status:          domain.DeploymentFailed,
		CreatedAt:       now.Add(-time.Minute),
		CommitMessage:   "chore: point at https://ci:hunter2@git.example.com and set JWT=" + jwt,
	}}
	recentErrors := []string{
		"Unauthorized: Authorization: Bearer AbCdEfGh12345678 was rejected",
		"Decode failed: {\"api_key\":\"" + sanctum + "\"}",
	}

	got := fleetSnapshotMarkdown(conn, apps, deployments, recentErrors, now)

	for name, value := range seeded {
		if strings.Contains(got, value) {
			t.Errorf("%s survived the snapshot:\n---\n%s", name, got)
		}
	}
	if !strings.Contains(got, logging.Mask) {
		t.Error("snapshot redacted nothing; the seeded secrets were not recognised")
	}
	if !strings.Contains(got, "replaced with "+logging.Mask) {
		t.Error("snapshot does not declare that it redacted anything")
	}
	// Redaction must not eat the context that makes the paste useful.
	for _, want := range []string{"api", "git.example.com", "running:healthy"} {
		if !strings.Contains(got, want) {
			t.Errorf("redaction removed useful context %q:\n---\n%s", want, got)
		}
	}
}

func TestFleetSnapshotDeclaresTruncation(t *testing.T) {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	deployments := make([]domain.Deployment, snapshotDeploymentLimit+5)
	for i := range deployments {
		deployments[i] = domain.Deployment{
			ApplicationName: "api", Status: domain.DeploymentFinished, CreatedAt: now,
		}
	}

	got := fleetSnapshotMarkdown(app.Connection{}, nil, deployments, nil, now)

	if !strings.Contains(got, "Newest 20 of 25") {
		t.Errorf("snapshot truncated silently:\n---\n%s", got)
	}
}
