package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

func progressTheme(t *testing.T) *theme.Theme {
	t.Helper()
	// ASCII keeps the assertions readable and is the harder case: the bar has
	// to work for anyone whose terminal mangles block characters.
	return theme.New(theme.Options{Mode: theme.ModeDark, ASCII: true})
}

func history(app string, runs ...time.Duration) []domain.Deployment {
	out := make([]domain.Deployment, 0, len(runs))
	for i, run := range runs {
		start := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC).Add(-time.Duration(i+1) * time.Hour)
		end := start.Add(run)
		out = append(out, domain.Deployment{
			ApplicationUUID: app,
			Status:          domain.DeploymentFinished,
			StartedAt:       start,
			FinishedAt:      &end,
		})
	}
	return out
}

func TestProgressCellTracksElapsedAgainstTheBaseline(t *testing.T) {
	th := progressTheme(t)
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	past := history("app", 100*time.Second, 100*time.Second, 100*time.Second)

	cases := []struct {
		name    string
		elapsed time.Duration
		want    string
	}{
		{"just started", 1 * time.Second, "-------- ~1m 40s"},
		{"half way", 50 * time.Second, "####---- ~1m 40s"},
		// The bar must not read as full until the build genuinely passes the
		// expectation, or "full" stops meaning anything.
		{"nearly there", 99 * time.Second, "#######- ~1m 40s"},
		{"over budget", 130 * time.Second, "######## +30s"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			running := domain.Deployment{
				ApplicationUUID: "app",
				Status:          domain.DeploymentInProgress,
				StartedAt:       now.Add(-tc.elapsed),
			}

			got := ansi.Strip(deploymentProgressCell(th, running, past, now))
			if got != tc.want {
				t.Fatalf("cell = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProgressCellIsEmptyWhenThereIsNothingHonestToShow(t *testing.T) {
	th := progressTheme(t)
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	running := domain.Deployment{
		ApplicationUUID: "app",
		Status:          domain.DeploymentInProgress,
		StartedAt:       now.Add(-time.Minute),
	}
	finishedRun := domain.Deployment{
		ApplicationUUID: "app",
		Status:          domain.DeploymentFinished,
		StartedAt:       now.Add(-time.Minute),
	}

	// A first-ever deploy has no expectation to compare against. Inventing one
	// would be worse than showing nothing.
	if cell := deploymentProgressCell(th, running, nil, now); cell != "" {
		t.Errorf("cell without history = %q, want empty", cell)
	}
	// A finished deployment's duration is already in the Duration column.
	if cell := deploymentProgressCell(th, finishedRun, history("app", time.Minute), now); cell != "" {
		t.Errorf("cell for a finished deployment = %q, want empty", cell)
	}
}

func TestProgressColumnFitsItsWidestCell(t *testing.T) {
	th := progressTheme(t)
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)

	running := domain.Deployment{
		ApplicationUUID: "app",
		Status:          domain.DeploymentInProgress,
		// Far enough over that the "+" label is at its longest.
		StartedAt: now.Add(-90 * time.Minute),
	}
	cell := ansi.Strip(deploymentProgressCell(th, running, history("app", 30*time.Minute), now))

	// A cell wider than the column would be truncated with an ellipsis, which
	// would eat the very number the user is looking at.
	if len(cell) > progressColumn.MinWidth {
		t.Fatalf("cell %q is %d wide, column is %d", cell, len(cell), progressColumn.MinWidth)
	}
	if !strings.HasPrefix(cell, strings.Repeat(th.Sym.BarFull, 8)) {
		t.Fatalf("an overrun should render a full bar, got %q", cell)
	}
}

func TestActiveDetection(t *testing.T) {
	if anyDeploymentActive(history("app", time.Minute)) {
		t.Error("finished deployments must not add the Progress column")
	}
	if !anyDeploymentActive([]domain.Deployment{{Status: domain.DeploymentQueued}}) {
		t.Error("a queued deployment is in flight")
	}
}
