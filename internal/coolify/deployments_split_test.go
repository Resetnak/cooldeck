package coolify

import (
	"testing"

	"github.com/resetnak/cooldeck/internal/domain"
)

func TestSplitDeploymentsActiveAndRecent(t *testing.T) {
	all := []domain.Deployment{
		{UUID: "1", Status: domain.DeploymentFinished},
		{UUID: "2", Status: domain.DeploymentInProgress},
		{UUID: "3", Status: domain.DeploymentFailed},
		{UUID: "4", Status: domain.DeploymentQueued},
	}
	active, recent := splitDeployments(all, 10)
	if len(active) != 2 {
		t.Fatalf("active = %d", len(active))
	}
	if !active[0].Status.IsActive() || !active[1].Status.IsActive() {
		t.Fatal("active list contains finished items")
	}
	if len(recent) != 4 {
		t.Fatalf("recent = %d", len(recent))
	}
	// Active first in recent.
	if !recent[0].Status.IsActive() || !recent[1].Status.IsActive() {
		t.Fatal("recent should lead with active")
	}
	_, limited := splitDeployments(all, 2)
	if len(limited) != 2 {
		t.Fatalf("limit = %d", len(limited))
	}
}
