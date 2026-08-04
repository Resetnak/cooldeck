package demo

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
)

var fixedNow = time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return New(Options{Now: func() time.Time { return fixedNow }})
}

// TestDeterministicForAGivenSeed is what makes the demo usable for golden
// tests: two services built with the same seed and clock must be identical.
func TestDeterministicForAGivenSeed(t *testing.T) {
	a, b := newTestService(t), newTestService(t)

	sa, err := a.Dashboard(t.Context())
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}
	sb, err := b.Dashboard(t.Context())
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}
	if len(sa.Applications) != len(sb.Applications) {
		t.Fatalf("app counts differ: %d vs %d", len(sa.Applications), len(sb.Applications))
	}
	for i := range sa.Applications {
		if !reflect.DeepEqual(sa.Applications[i], sb.Applications[i]) {
			t.Fatalf("application %d differs between runs:\n%+v\n%+v",
				i, sa.Applications[i], sb.Applications[i])
		}
	}

	// A different seed must produce different UUIDs, otherwise the seed does
	// not actually reach the generator.
	other := New(Options{Seed: 1, Now: func() time.Time { return fixedNow }})
	so, _ := other.Dashboard(t.Context())
	if so.Applications[0].UUID == sa.Applications[0].UUID {
		t.Error("a different seed should produce different data")
	}
}

// TestFixturesCoverEveryVisualState keeps the demo useful as a UI harness: if
// a state stops being represented, the screens for it stop being exercised.
func TestFixturesCoverEveryVisualState(t *testing.T) {
	snap, err := newTestService(t).Dashboard(t.Context())
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}

	seen := map[domain.ResourceStatus]bool{}
	var withoutDomain, longName, withoutDeployment bool
	for _, a := range snap.Applications {
		seen[a.Status.State] = true
		if a.PrimaryDomain() == "" {
			withoutDomain = true
		}
		if len(a.Name) > 30 {
			longName = true
		}
		if a.LastDeployment == nil {
			withoutDeployment = true
		}
	}

	for _, want := range []domain.ResourceStatus{
		domain.StatusRunning, domain.StatusDegraded, domain.StatusStopped,
		domain.StatusBuilding, domain.StatusDeploying, domain.StatusRestarting,
		domain.StatusUnknown,
	} {
		if !seen[want] {
			t.Errorf("no demo application is in state %q", want)
		}
	}
	if !withoutDomain {
		t.Error("no demo application is missing a domain")
	}
	if !longName {
		t.Error("no demo application has a name long enough to test truncation")
	}
	if !withoutDeployment {
		t.Error("no demo application is missing its deployment history")
	}
}

func TestOfflineSimulation(t *testing.T) {
	s := newTestService(t)
	s.SetOffline(true)

	if _, err := s.Dashboard(t.Context()); !domain.IsKind(err, domain.ErrorNetwork) {
		t.Fatalf("got %v, want a network error while offline", err)
	}
	if _, err := s.Connect(t.Context()); err == nil {
		t.Fatal("Connect should fail while offline")
	}

	s.SetOffline(false)
	if _, err := s.Dashboard(t.Context()); err != nil {
		t.Fatalf("Dashboard should recover: %v", err)
	}
}

func TestCancellationIsHonoured(t *testing.T) {
	s := New(Options{Latency: time.Second, Now: func() time.Time { return fixedNow }})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	start := time.Now()
	_, err := s.Dashboard(ctx)
	if !domain.IsKind(err, domain.ErrorCancelled) {
		t.Fatalf("got %v, want a cancelled error", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Errorf("cancellation took %s; the latency delay should be interrupted", elapsed)
	}
}

func TestLogsAreSanitized(t *testing.T) {
	s := newTestService(t)
	snap, err := s.Dashboard(t.Context())
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}

	target := snap.Applications[0]
	logs, err := s.RuntimeLogs(t.Context(), target.UUID, 100)
	if err != nil {
		t.Fatalf("RuntimeLogs: %v", err)
	}
	if len(logs.Lines) == 0 {
		t.Fatal("expected runtime log output")
	}
	for _, l := range logs.Lines {
		if strings.ContainsRune(l.Text, 0x1b) {
			t.Fatalf("escape sequence survived into the demo logs: %q", l.Text)
		}
	}

	// The build log deliberately contains ANSI colour codes, which is exactly
	// what the sanitiser has to strip.
	deps, err := s.Deployments(t.Context(), target.UUID, 10)
	if err != nil {
		t.Fatalf("Deployments: %v", err)
	}
	if len(deps) == 0 {
		t.Fatal("expected deployment history")
	}
	dl, err := s.DeploymentLogs(t.Context(), deps[0].UUID)
	if err != nil {
		t.Fatalf("DeploymentLogs: %v", err)
	}
	for _, l := range dl.Lines {
		if strings.ContainsRune(l.Text, 0x1b) {
			t.Fatalf("escape sequence survived into the build log: %q", l.Text)
		}
	}
}

func TestMutationsAffectOnlyLocalState(t *testing.T) {
	s := newTestService(t)
	snap, _ := s.Dashboard(t.Context())
	target := snap.Applications[0]

	res, err := s.Stop(t.Context(), target.UUID)
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if res.Operation != "stop" || res.ResourceUUID != target.UUID {
		t.Errorf("unexpected result %+v", res)
	}

	after, _ := s.Dashboard(t.Context())
	if after.Applications[0].Status.State != domain.StatusStopped {
		t.Errorf("status = %q, want stopped", after.Applications[0].Status.State)
	}

	// A deploy must queue a followable deployment rather than doing nothing.
	deployRes, err := s.Deploy(t.Context(), target.UUID, app.DeployOptions{Force: true})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if deployRes.DeploymentUUID == "" {
		t.Fatal("deploy should return a deployment UUID to follow")
	}
	deps, _ := s.Deployments(t.Context(), target.UUID, 5)
	if deps[0].UUID != deployRes.DeploymentUUID {
		t.Error("the queued deployment should be the newest in the history")
	}
	if !deps[0].ForceBuild {
		t.Error("force flag should be recorded on the deployment")
	}
}

func TestInFlightDeploymentProgressesOverTime(t *testing.T) {
	clock := fixedNow
	s := New(Options{Now: func() time.Time { return clock }})

	snap, _ := s.Dashboard(t.Context())
	target := snap.Applications[0]
	if _, err := s.Deploy(t.Context(), target.UUID, app.DeployOptions{}); err != nil {
		t.Fatalf("Deploy: %v", err)
	}

	deps, _ := s.Deployments(t.Context(), target.UUID, 1)
	if deps[0].Status != domain.DeploymentQueued {
		t.Fatalf("status = %q, want queued", deps[0].Status)
	}

	clock = clock.Add(30 * time.Second)
	deps, _ = s.Deployments(t.Context(), target.UUID, 1)
	if deps[0].Status != domain.DeploymentInProgress {
		t.Fatalf("status = %q, want in_progress after 30s", deps[0].Status)
	}

	clock = clock.Add(2 * time.Minute)
	deps, _ = s.Deployments(t.Context(), target.UUID, 1)
	if deps[0].Status != domain.DeploymentFinished {
		t.Fatalf("status = %q, want finished after 2m30s", deps[0].Status)
	}
	if deps[0].FinishedAt == nil {
		t.Error("a finished deployment must have a finish time")
	}
}

func TestUnknownApplicationYieldsNotFound(t *testing.T) {
	s := newTestService(t)
	if _, err := s.ApplicationDetail(t.Context(), "nope"); !domain.IsKind(err, domain.ErrorNotFound) {
		t.Errorf("got %v, want not found", err)
	}
	if _, err := s.Restart(t.Context(), "nope"); !domain.IsKind(err, domain.ErrorNotFound) {
		t.Errorf("got %v, want not found", err)
	}
}

func TestAppCountScalesTheFleet(t *testing.T) {
	s := New(Options{AppCount: 200, Now: func() time.Time { return fixedNow }})
	snap, err := s.Dashboard(t.Context())
	if err != nil {
		t.Fatalf("Dashboard: %v", err)
	}
	if len(snap.Applications) != 200 {
		t.Fatalf("got %d applications, want 200", len(snap.Applications))
	}

	seen := map[string]bool{}
	for _, a := range snap.Applications {
		if seen[a.UUID] {
			t.Fatalf("duplicate UUID %q", a.UUID)
		}
		seen[a.UUID] = true
	}
}
