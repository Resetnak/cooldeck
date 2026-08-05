package components

import (
	"strings"
	"time"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// barCells is the width of the bar itself. Eight cells is enough to read at a
// glance and leaves room for the label inside a table column.
const barCells = 8

// DurationBar renders how long an in-flight deployment has been running against
// how long this application usually takes.
//
// It deliberately never shows a percentage. Coolify cannot say how far a build
// has actually got, so a number like "60%" would be a claim the UI has no way
// to back up - and the first slow build would expose it. What it shows is
// elapsed against an expectation, and once that expectation is passed it stops
// estimating and reports how far over it has gone, which is the point at which
// the user should go and look at the logs.
//
// An empty string means there is nothing worth drawing: no baseline to compare
// against, or a deployment that is not running.
func DurationBar(th *theme.Theme, elapsed, baseline time.Duration) string {
	if baseline <= 0 || elapsed < 0 {
		return ""
	}

	filled := barCells
	label := "+" + domain.HumanizeDuration(elapsed-baseline)
	style, labelStyle := th.Attention, th.Attention
	if elapsed < baseline {
		// Round down, so the bar only reads as full when it genuinely is.
		filled = int(int64(barCells) * int64(elapsed) / int64(baseline))
		label = "~" + domain.HumanizeDuration(baseline)
		style, labelStyle = th.Accent, th.Subtle
	}
	filled = max(min(filled, barCells), 0)

	bar := style.Render(strings.Repeat(th.Sym.BarFull, filled)) +
		th.Subtle.Render(strings.Repeat(th.Sym.BarEmpty, barCells-filled))
	return bar + " " + labelStyle.Render(label)
}

// DurationBarWidth is the widest a bar plus its label can be, so a table can
// size the column without measuring every row.
func DurationBarWidth() int {
	// The label is at most "+59m 59s": one sign, then the longest duration the
	// bar can meaningfully show before the build is a lost cause anyway.
	return barCells + 1 + len("+59m 59s")
}
