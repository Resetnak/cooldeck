package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/views"
)

func tailModel(t *testing.T) *Model {
	t.Helper()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
	return model
}

func markFirst(t *testing.T, model *Model, n int) []string {
	t.Helper()
	uuids := make([]string, 0, n)
	for i := range n {
		a, ok := model.apps.At(i)
		if !ok {
			t.Fatalf("only %d applications available", i)
		}
		model.apps.SelectUUID(a.UUID)
		if _, ok := model.apps.ToggleMark(); !ok {
			t.Fatal("could not mark")
		}
		uuids = append(uuids, a.UUID)
	}
	return uuids
}

// The spacebar arrives as Code KeySpace and stringifies to "space"; a binding on
// a literal " " silently never matches and the whole tail becomes unreachable.
func TestSpacebarMarksTheSelectedApplication(t *testing.T) {
	model := tailModel(t)

	model.handleApplicationsKey(tea.KeyPressMsg{Code: ' ', Text: " "})

	if len(model.apps.Marked()) != 1 {
		t.Fatalf("space marked %d applications, want 1", len(model.apps.Marked()))
	}
}

func TestTailNeedsSomethingMarked(t *testing.T) {
	model := tailModel(t)

	_, cmd := model.openTail()
	if model.screen == screenTail {
		t.Fatal("opened an empty tail")
	}
	if cmd == nil {
		t.Fatal("the user was not told why nothing happened")
	}
}

func TestTailStartsOnePollPerMarkedApplication(t *testing.T) {
	model := tailModel(t)
	uuids := markFirst(t, model, 3)

	_, cmd := model.openTail()
	if model.screen != screenTail {
		t.Fatal("t did not open the tail")
	}
	if got := model.tail.Count(); got != 3 {
		t.Fatalf("tailing %d applications, want 3", got)
	}

	// Every source has to be armed, or one of them would simply never update.
	ticks := map[string]bool{}
	collectTailTicks(cmd, ticks)
	for _, uuid := range uuids {
		if !ticks[uuid] {
			t.Errorf("no poll scheduled for %s", uuid)
		}
	}
}

// collectTailTicks walks a command tree, running only the ticks, and records
// which applications they belong to.
func collectTailTicks(cmd tea.Cmd, into map[string]bool) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case tailTickMsg:
		into[msg.AppUUID] = true
	case tea.BatchMsg:
		for _, sub := range msg {
			collectTailTicks(sub, into)
		}
	}
}

func TestLeavingTheTailOrphansItsRepliesInFlight(t *testing.T) {
	model := tailModel(t)
	uuids := markFirst(t, model, 2)
	model.openTail()
	stale := model.tailSeq

	model.closeTail()
	if model.screen != screenList {
		t.Fatal("esc did not return to the list")
	}

	// A reply from the tail the user just left must not schedule more work:
	// bumping the sequence is what stands in for cancelling several requests.
	_, cmd := model.Update(tailLoadedMsg{
		Seq:      stale,
		AppUUID:  uuids[0],
		Snapshot: app.LogSnapshot{Lines: []domain.LogLine{{Text: "late"}}},
	})
	if cmd != nil {
		t.Fatal("a stale reply scheduled another poll")
	}
}

func TestRetryDelayObeysCoolifyOverTheDefault(t *testing.T) {
	rateLimited := domain.NewError(domain.ErrorRateLimited, nil)
	rateLimited.RetryAfter = 30 * time.Second

	if got := tailRetryDelay(rateLimited); got != 30*time.Second {
		t.Errorf("rate limited: waited %s, want the 30s Coolify asked for", got)
	}
	// A rate limit with no hint, and every other failure, falls back to backing
	// off rather than retrying on the normal interval.
	if got := tailRetryDelay(domain.NewError(domain.ErrorRateLimited, nil)); got != 2*tailInterval {
		t.Errorf("rate limited without a hint: waited %s, want %s", got, 2*tailInterval)
	}
	if got := tailRetryDelay(domain.NewError(domain.ErrorNetwork, nil)); got != 2*tailInterval {
		t.Errorf("network failure: waited %s, want %s", got, 2*tailInterval)
	}
}

func TestFailedSourceIsRetriedNotDropped(t *testing.T) {
	model := tailModel(t)
	uuids := markFirst(t, model, 1)
	model.openTail()

	cmd := model.applyTailFailed(tailFailedMsg{
		Seq: model.tailSeq, AppUUID: uuids[0],
		Err: domain.NewError(domain.ErrorNetwork, nil),
	})
	if cmd == nil {
		t.Fatal("a failing source was never retried")
	}
}

func TestTailStaggerSpreadsSourcesAcrossTheInterval(t *testing.T) {
	// All five firing at once would be a burst every tick, which is the thing
	// the interval exists to avoid.
	seen := map[time.Duration]bool{}
	for i := range views.MaxTailApps {
		d := tailStagger(i, views.MaxTailApps)
		if seen[d] {
			t.Fatalf("source %d shares its offset %s with an earlier one", i, d)
		}
		seen[d] = true
		if d >= tailInterval {
			t.Fatalf("offset %s is beyond the interval %s", d, tailInterval)
		}
	}
}
