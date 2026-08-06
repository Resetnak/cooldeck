package tui

import (
	"context"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/views"
)

// tailInterval is how often each tailed application is re-read. It is
// deliberately slower than the single-application log poll: the tail multiplies
// every tick by the number of sources, and config.MinRefreshInterval exists
// because Coolify's rate limit is shared with everything else the session does.
const tailInterval = 4 * time.Second

// tailStagger spreads the sources across the interval instead of firing them
// all at once, so a fleet tail looks like a steady trickle to the API rather
// than a burst every four seconds.
func tailStagger(index, count int) time.Duration {
	if count <= 1 {
		return 0
	}
	return time.Duration(index) * tailInterval / time.Duration(count)
}

// openTail switches to the fleet tail for the marked applications.
func (m *Model) openTail() (tea.Model, tea.Cmd) {
	marked := m.apps.Marked()
	if len(marked) == 0 {
		return m, m.pushToast(components.ToastInfo, "Nothing marked",
			"Press space on the applications list to mark what to tail.")
	}

	dropped := 0
	if len(marked) > views.MaxTailApps {
		dropped = len(marked) - views.MaxTailApps
		marked = marked[:views.MaxTailApps]
	}

	m.tail.SetApplications(marked)
	m.screen = screenTail
	m.tailSeq++

	cmds := make([]tea.Cmd, 0, len(marked)+1)
	for i := range marked {
		cmds = append(cmds, m.tailTick(marked[i].UUID, tailStagger(i, len(marked))))
	}
	if dropped > 0 {
		// Say what was left out rather than silently tailing a subset.
		cmds = append(cmds, m.pushToast(components.ToastWarning, "Tailing the first "+
			strconv.Itoa(views.MaxTailApps), strconv.Itoa(dropped)+" more stayed behind; Coolify serves logs as whole snapshots, so each source is its own request."))
	}
	return m, tea.Batch(cmds...)
}

// closeTail returns to the applications list and stops the polling.
func (m *Model) closeTail() {
	m.screen = screenList
	// Bumping the sequence orphans every reply still in flight, which is how a
	// set of concurrent requests is cancelled without tracking each one.
	m.tailSeq++
	m.tailSearching = false
	m.tailSearchText = ""
}

// tailTick schedules the next read of one source.
func (m *Model) tailTick(appUUID string, delay time.Duration) tea.Cmd {
	seq := m.tailSeq
	if delay <= 0 {
		delay = tailInterval
	}
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return tailTickMsg{Seq: seq, AppUUID: appUUID}
	})
}

// loadTailLogs reads one source. Unlike the single-application log request this
// keeps no cancel function: several are in flight at once by design, and a
// stale reply is dropped by its sequence number instead.
func (m *Model) loadTailLogs(appUUID string) tea.Cmd {
	seq := m.tailSeq
	service := m.service
	lines := m.opts.Config.LogLines
	if lines <= 0 {
		lines = 100
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		snapshot, err := service.RuntimeLogs(ctx, appUUID, lines)
		if err != nil {
			return tailFailedMsg{Seq: seq, AppUUID: appUUID, Err: domain.AsError(err)}
		}
		return tailLoadedMsg{Seq: seq, AppUUID: appUUID, Snapshot: snapshot}
	}
}

// applyTailLoaded stores one snapshot and schedules that source's next read.
func (m *Model) applyTailLoaded(msg tailLoadedMsg) tea.Cmd {
	if msg.Seq != m.tailSeq || m.screen != screenTail {
		return nil
	}
	m.tail.SetLines(msg.AppUUID, msg.Snapshot.Lines, msg.Snapshot.LoadedAt)
	if m.tail.Paused() {
		return nil
	}
	return m.tailTick(msg.AppUUID, 0)
}

// applyTailFailed degrades one source without emptying the view, and backs off
// rather than hammering an instance that is rate limiting or down.
func (m *Model) applyTailFailed(msg tailFailedMsg) tea.Cmd {
	if msg.Seq != m.tailSeq || m.screen != screenTail {
		return nil
	}
	if msg.Err.Kind == domain.ErrorCancelled {
		return nil
	}

	m.tail.SetFailure(msg.AppUUID, msg.Err.Title)
	if m.tail.Paused() {
		return nil
	}

	return m.tailTick(msg.AppUUID, tailRetryDelay(msg.Err))
}

// tailRetryDelay decides how long to wait before reading a source again after
// a failure. A rate limit is obeyed to the second Coolify asked for, because
// arguing with one is how it becomes a ban; anything else backs off to twice
// the normal interval so a source that is simply down costs half as much.
func tailRetryDelay(err *domain.Error) time.Duration {
	if err != nil && err.Kind == domain.ErrorRateLimited && err.RetryAfter > 0 {
		return err.RetryAfter
	}
	return tailInterval * 2
}
