package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

func tailTheme(t *testing.T) *theme.Theme {
	t.Helper()
	return theme.New(theme.Options{Mode: theme.ModeDark, ASCII: true})
}

func at(clock time.Time, offset time.Duration, text string) domain.LogLine {
	return domain.LogLine{Timestamp: clock.Add(offset), Text: text}
}

func TestTailInterleavesSourcesByTimestamp(t *testing.T) {
	clock := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	v := NewTail()
	v.SetApplications([]domain.Application{{UUID: "a", Name: "api"}, {UUID: "b", Name: "web"}})

	v.SetLines("a", []domain.LogLine{at(clock, 0, "api first"), at(clock, 2*time.Second, "api third")}, clock)
	v.SetLines("b", []domain.LogLine{at(clock, time.Second, "web second")}, clock)

	var got []string
	for _, e := range v.merged(clock) {
		got = append(got, e.line.Text)
	}
	want := []string{"api first", "web second", "api third"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestTailKeepsContinuationLinesWithTheirParent(t *testing.T) {
	clock := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	v := NewTail()
	v.SetApplications([]domain.Application{{UUID: "a", Name: "api"}, {UUID: "b", Name: "web"}})

	// A stack trace: only the first line carries a timestamp. Sorting the
	// untimed lines to the front would tear the trace apart.
	v.SetLines("a", []domain.LogLine{
		at(clock, time.Second, "panic: boom"),
		{Text: "  goroutine 1"},
		{Text: "  main.go:42"},
	}, clock)
	v.SetLines("b", []domain.LogLine{at(clock, 2*time.Second, "web after")}, clock)

	var got []string
	for _, e := range v.merged(clock) {
		got = append(got, e.line.Text)
	}
	want := []string{"panic: boom", "  goroutine 1", "  main.go:42", "web after"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestTailCapsTheNumberOfSources(t *testing.T) {
	v := NewTail()
	apps := make([]domain.Application, 0, MaxTailApps+3)
	for i := range MaxTailApps + 3 {
		apps = append(apps, domain.Application{UUID: string(rune('a' + i)), Name: "app"})
	}
	v.SetApplications(apps)

	// Every source is a separate poll on every tick, so the cap is a rate-limit
	// guard, not a display preference.
	if v.Count() != MaxTailApps {
		t.Fatalf("tailing %d applications, want at most %d", v.Count(), MaxTailApps)
	}
}

func TestTailKeepsSnapshotsAcrossAReselect(t *testing.T) {
	clock := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	v := NewTail()
	v.SetApplications([]domain.Application{{UUID: "a", Name: "api"}})
	v.SetLines("a", []domain.LogLine{at(clock, 0, "keep me")}, clock)

	// Marking another application must not blank the buffer of one already
	// being tailed.
	v.SetApplications([]domain.Application{{UUID: "a", Name: "api"}, {UUID: "b", Name: "web"}})
	if len(v.merged(clock)) != 1 {
		t.Fatalf("re-selecting dropped the existing snapshot")
	}
}

func TestTailFailureDegradesOneSourceOnly(t *testing.T) {
	clock := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	th := tailTheme(t)
	v := NewTail()
	v.SetApplications([]domain.Application{{UUID: "a", Name: "api"}, {UUID: "b", Name: "web"}})
	v.SetLines("a", []domain.LogLine{at(clock, 0, "still here")}, clock)
	v.SetFailure("b", "Permission denied")

	out := ansi.Strip(v.Render(th, 120, 20, clock))
	if !strings.Contains(out, "still here") {
		t.Error("a failing source emptied the view")
	}
	if !strings.Contains(out, "Permission denied") {
		t.Error("the failure is not reported in the header")
	}
}

func TestTailScrollingUpLeavesFollowMode(t *testing.T) {
	v := NewTail()
	if !v.Following() {
		t.Fatal("a new tail should follow")
	}
	// Scrolling back while pinned to the newest line would fight the user on
	// every poll.
	v.Scroll(-1)
	if v.Following() {
		t.Fatal("scrolling up did not leave follow mode")
	}
}
