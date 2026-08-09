package views

import (
	"testing"
	"time"

	"github.com/resetnak/cooldeck/internal/domain"
)

func deployment(uuid string, status domain.DeploymentStatus) domain.Deployment {
	return domain.Deployment{UUID: uuid, ApplicationUUID: "app", Status: status}
}

func uuids(deployments []domain.Deployment) []string {
	out := make([]string, 0, len(deployments))
	for _, d := range deployments {
		out = append(out, d.UUID)
	}
	return out
}

func TestSetItemsLeadsWithWhatIsInFlight(t *testing.T) {
	view := NewDeployments()
	now := time.Now()

	// Coolify returns newest first, which puts a running build below a wall of
	// finished ones as soon as the history is longer than the screen.
	view.SetItems([]domain.Deployment{
		deployment("finished-1", domain.DeploymentFinished),
		deployment("finished-2", domain.DeploymentFailed),
		deployment("running", domain.DeploymentInProgress),
		deployment("finished-3", domain.DeploymentFinished),
		deployment("queued", domain.DeploymentQueued),
	}, now)

	got := uuids(view.items)
	want := []string{"running", "queued", "finished-1", "finished-2", "finished-3"}
	for i := range want {
		if got[i] != want[i] {
			// Settled deployments must keep the order they arrived in: the sort
			// promotes the live ones, it does not reshuffle the history.
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestSelectionFollowsADeploymentThatSettles(t *testing.T) {
	view := NewDeployments()
	now := time.Now()

	view.SetItems([]domain.Deployment{
		deployment("older", domain.DeploymentFinished),
		deployment("watched", domain.DeploymentInProgress),
	}, now)
	if selected, _ := view.Selected(); selected.UUID != "watched" {
		// It sorted to the top, and the cursor starts at the top.
		t.Fatalf("selected %q at the top of the queue, want watched", selected.UUID)
	}

	// The build finishes: it drops back into chronological order, and the
	// cursor has to follow the row rather than stay on the index.
	view.SetItems([]domain.Deployment{
		deployment("older", domain.DeploymentFinished),
		deployment("watched", domain.DeploymentFinished),
	}, now)

	selected, ok := view.Selected()
	if !ok || selected.UUID != "watched" {
		t.Fatalf("selection = %q (ok=%v), want watched", selected.UUID, ok)
	}
}

func TestTimelineOrdersChronologically(t *testing.T) {
	view := NewDeployments()
	now := time.Now()

	older := deployment("older-active", domain.DeploymentInProgress)
	older.CreatedAt = now.Add(-2 * time.Hour)
	newer := deployment("newer-finished", domain.DeploymentFinished)
	newer.CreatedAt = now.Add(-5 * time.Minute)
	view.SetItems([]domain.Deployment{older, newer}, now)

	// The table leads with the active build; the timeline must not.
	if got := uuids(view.visible()); got[0] != "older-active" {
		t.Fatalf("table order = %v, want active first", got)
	}
	if !view.ToggleTimeline() {
		t.Fatal("ToggleTimeline() = false on first toggle")
	}
	if got := uuids(view.visible()); got[0] != "newer-finished" {
		t.Fatalf("timeline order = %v, want newest first", got)
	}
}
