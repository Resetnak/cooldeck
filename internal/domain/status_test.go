package domain

import (
	"sort"
	"testing"
)

func TestParseStatus(t *testing.T) {
	tests := []struct {
		raw        string
		wantState  ResourceStatus
		wantHealth Health
	}{
		{"running:healthy", StatusRunning, HealthHealthy},
		// A running-but-unhealthy container is a real production problem and
		// must not be flattened into a plain "Running".
		{"running:unhealthy", StatusDegraded, HealthUnhealthy},
		{"running:starting", StatusRunning, HealthStarting},
		{"exited:unhealthy", StatusStopped, HealthUnhealthy},
		{"restarting:starting", StatusRestarting, HealthStarting},
		{"degraded:unhealthy", StatusDegraded, HealthUnhealthy},
		{"running", StatusRunning, HealthNone},
		{"exited", StatusStopped, HealthNone},
		{"stopped", StatusStopped, HealthNone},
		{"RUNNING:HEALTHY", StatusRunning, HealthHealthy},
		{"  running:healthy  ", StatusRunning, HealthHealthy},
		{"in_progress", StatusBuilding, HealthNone},
		{"queued", StatusDeploying, HealthNone},
		// Anything unrecognised must degrade, never panic or leak through.
		{"warp-drive:plaid", StatusUnknown, HealthNone},
		{"", StatusUnknown, HealthNone},
	}
	for _, tt := range tests {
		got := ParseStatus(tt.raw)
		if got.State != tt.wantState {
			t.Errorf("ParseStatus(%q).State = %q, want %q", tt.raw, got.State, tt.wantState)
		}
		if got.Health != tt.wantHealth {
			t.Errorf("ParseStatus(%q).Health = %q, want %q", tt.raw, got.Health, tt.wantHealth)
		}
		if got.Label() == "" {
			t.Errorf("ParseStatus(%q) produced an empty label", tt.raw)
		}
	}
}

func TestStatusPredicates(t *testing.T) {
	if !ParseStatus("running:unhealthy").NeedsAttention() {
		t.Error("an unhealthy container needs attention")
	}
	if ParseStatus("running:healthy").NeedsAttention() {
		t.Error("a healthy container does not need attention")
	}
	if !ParseStatus("restarting").IsTransitional() {
		t.Error("restarting is transitional")
	}
	if ParseStatus("exited").IsTransitional() {
		t.Error("exited is not transitional")
	}
	if !ParseStatus("running:unhealthy").IsRunning() {
		t.Error("a degraded container is still running")
	}
}

// TestStatusSeverityOrdersWorstFirst pins the sort key that drives the
// "sort by status" mode: problems must float to the top.
func TestStatusSeverityOrdersWorstFirst(t *testing.T) {
	raws := []string{"running:healthy", "exited", "running:unhealthy", "in_progress", "failed", "banana"}
	sort.Slice(raws, func(i, j int) bool {
		return ParseStatus(raws[i]).Severity() < ParseStatus(raws[j]).Severity()
	})
	want := []string{"failed", "running:unhealthy", "in_progress", "exited", "running:healthy", "banana"}
	for i := range want {
		if raws[i] != want[i] {
			t.Fatalf("severity order = %v, want %v", raws, want)
		}
	}
}

func TestParseDeploymentStatus(t *testing.T) {
	tests := map[string]DeploymentStatus{
		"queued":              DeploymentQueued,
		"in_progress":         DeploymentInProgress,
		"finished":            DeploymentFinished,
		"failed":              DeploymentFailed,
		"cancelled-by-user":   DeploymentCancelled,
		"canceled-by-user":    DeploymentCancelled,
		"CANCELLED-BY-SYSTEM": DeploymentCancelled,
		"":                    DeploymentUnknown,
		"something-new":       DeploymentUnknown,
	}
	for raw, want := range tests {
		if got := ParseDeploymentStatus(raw); got != want {
			t.Errorf("ParseDeploymentStatus(%q) = %q, want %q", raw, got, want)
		}
	}

	if !DeploymentQueued.IsActive() || !DeploymentInProgress.IsActive() {
		t.Error("queued and in-progress deployments are active")
	}
	if DeploymentFinished.IsActive() {
		t.Error("a finished deployment is not active")
	}
	if DeploymentFailed.AsResourceStatus() != StatusFailed {
		t.Error("a failed deployment projects onto StatusFailed")
	}
}
