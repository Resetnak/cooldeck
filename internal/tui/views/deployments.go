package views

import (
	"slices"
	"strconv"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// Deployments is the top-level deployments overview: recent history across
// every application, with an optional active-only filter.
type Deployments struct {
	items      []domain.Deployment
	selected   int
	loaded     bool
	loadedAt   time.Time
	activeOnly bool
	// timeline switches from the status-grouped table to a strictly
	// chronological view with day separators - one merged story of what
	// happened across the fleet, newest first.
	timeline bool
}

// NewDeployments returns an empty deployments view.
func NewDeployments() *Deployments {
	return &Deployments{items: []domain.Deployment{}}
}

// SetItems replaces the list, preserving selection by UUID when possible.
func (v *Deployments) SetItems(items []domain.Deployment, at time.Time) {
	var keep string
	if d, ok := v.Selected(); ok {
		keep = d.UUID
	}
	v.items = append([]domain.Deployment{}, items...)
	// A queue leads with what is happening now. Coolify returns newest first,
	// which buries an in-flight build under yesterday's history the moment the
	// history is longer than the screen - and that build is the row the user
	// opened this section to look at. Sorting here rather than in visible()
	// means it costs one pass per refresh instead of one per frame.
	slices.SortStableFunc(v.items, activeFirst)
	v.loaded = true
	v.loadedAt = at
	v.selected = 0
	visible := v.visible()
	if keep != "" {
		for i, d := range visible {
			if d.UUID == keep {
				v.selected = i
				break
			}
		}
	}
	if v.selected >= len(visible) {
		v.selected = max(len(visible)-1, 0)
	}
}

// ToggleActiveOnly switches between full recent history and the live queue.
func (v *Deployments) ToggleActiveOnly() bool {
	v.activeOnly = !v.activeOnly
	v.selected = 0
	return v.activeOnly
}

// ActiveOnly reports the current filter.
func (v *Deployments) ActiveOnly() bool { return v.activeOnly }

// ToggleTimeline switches between the status-grouped table and the
// chronological fleet timeline.
func (v *Deployments) ToggleTimeline() bool {
	v.timeline = !v.timeline
	v.selected = 0
	return v.timeline
}

// Timeline reports whether the chronological view is active.
func (v *Deployments) Timeline() bool { return v.timeline }

// Items returns the full loaded history, for the fleet snapshot.
func (v *Deployments) Items() []domain.Deployment {
	return append([]domain.Deployment{}, v.items...)
}

func (v *Deployments) visible() []domain.Deployment {
	out := v.items
	if v.activeOnly {
		filtered := make([]domain.Deployment, 0, len(out))
		for _, d := range out {
			if d.Status.IsActive() {
				filtered = append(filtered, d)
			}
		}
		out = filtered
	}
	if v.timeline {
		// The timeline tells the story in the order it happened, so the
		// active-first grouping of the table gives way to pure chronology.
		chrono := append([]domain.Deployment{}, out...)
		slices.SortStableFunc(chrono, func(a, b domain.Deployment) int {
			return b.CreatedAt.Compare(a.CreatedAt)
		})
		return chrono
	}
	return out
}

// Loaded reports whether data has been set at least once.
func (v *Deployments) Loaded() bool { return v.loaded }

// Count returns the number of visible rows.
func (v *Deployments) Count() int { return len(v.visible()) }

// Total returns the unfiltered recent history size.
func (v *Deployments) Total() int { return len(v.items) }

// ActiveCount returns how many items are still in flight.
func (v *Deployments) ActiveCount() int {
	n := 0
	for _, d := range v.items {
		if d.Status.IsActive() {
			n++
		}
	}
	return n
}

// Selected returns the highlighted deployment.
func (v *Deployments) Selected() (domain.Deployment, bool) {
	visible := v.visible()
	if v.selected < 0 || v.selected >= len(visible) {
		return domain.Deployment{}, false
	}
	return visible[v.selected], true
}

// SelectedIndex is the zero-based cursor position within the visible list.
func (v *Deployments) SelectedIndex() int { return v.selected }

// Move shifts the selection by delta rows.
func (v *Deployments) Move(delta int) {
	visible := v.visible()
	if len(visible) == 0 {
		return
	}
	v.selected = min(max(v.selected+delta, 0), len(visible)-1)
}

// MoveTo jumps to an absolute index.
func (v *Deployments) MoveTo(idx int) {
	visible := v.visible()
	if len(visible) == 0 {
		return
	}
	v.selected = min(max(idx, 0), len(visible)-1)
}

// Render draws the deployments table.
func (v *Deployments) Render(th *theme.Theme, width, height int, focused bool, now time.Time) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if !v.loaded {
		return components.Skeleton(th, width, height)
	}
	visible := v.visible()
	if len(visible) == 0 {
		title := "No deployments yet"
		body := "When Coolify builds or deploys, recent work appears here."
		if v.activeOnly {
			title = "No active deployments"
			body = "Nothing is currently building. Press a to show recent history."
		}
		return components.EmptyState(
			th,
			width,
			height,
			title,
			body,
			[]components.KeyHint{
				{Key: "a", Desc: "toggle active"},
				{Key: "1", Desc: "applications"},
				{Key: "R", Desc: "refresh"},
			},
		)
	}

	if v.timeline {
		return v.renderTimeline(th, width, height, visible, now)
	}

	// The baseline is taken from the whole loaded history, not just the visible
	// rows, so filtering to active-only does not remove the very deployments the
	// expectation is computed from.
	showProgress := anyDeploymentActive(visible)

	rows := make([]components.Row, 0, len(visible))
	for _, d := range visible {
		name := d.ApplicationName
		if name == "" {
			name = d.ApplicationUUID
		}
		cells := []string{
			th.DeploymentStatusText(d.Status),
			components.OrDash(name),
			components.OrDash(d.ShortCommit()),
			domain.HumanizeAge(d.CreatedAt, now),
			domain.HumanizeDuration(d.Duration(now)),
		}
		if showProgress {
			cells = append(cells, deploymentProgressCell(th, d, v.items, now))
		}
		cells = append(cells,
			components.OrDash(d.Trigger),
			components.OrDash(d.CommitMessage),
		)
		rows = append(rows, components.Row{Cells: cells})
	}

	header := th.Title.Render("DEPLOYMENTS")
	mode := "recent history"
	if v.activeOnly {
		mode = "active only"
	}
	meta := th.Subtle.Render(strconv.Itoa(len(visible)) + " shown  ·  " +
		strconv.Itoa(v.ActiveCount()) + " active  ·  " + mode + "  ·  a toggle")
	columns := []components.Column{
		{Title: "Status", MinWidth: 12, Priority: 0},
		{Title: "Application", MinWidth: 14, Flex: 1, Priority: 0},
		{Title: "Commit", MinWidth: 8, Priority: 1},
		{Title: "Started", MinWidth: 10, Priority: 2},
		{Title: "Duration", MinWidth: 10, Priority: 2},
	}
	if showProgress {
		columns = append(columns, progressColumn)
	}
	columns = append(columns,
		components.Column{Title: "Trigger", MinWidth: 8, Priority: 3},
		components.Column{Title: "Message", MinWidth: 16, Flex: 1, Priority: 4},
	)

	table := components.Table{
		Columns:  columns,
		Rows:     rows,
		Selected: v.selected,
		Focused:  focused,
	}
	body := lipgloss.JoinVertical(lipgloss.Left, header, meta, "", table.Render(th, width, max(height-3, 1)))
	return components.FitBlock(body, width, height)
}

// renderTimeline draws the chronological fleet timeline: every deployment
// across every application on one axis, separated by day, newest first.
func (v *Deployments) renderTimeline(th *theme.Theme, width, height int, visible []domain.Deployment, now time.Time) string {
	header := th.Title.Render("FLEET TIMELINE")
	meta := th.Subtle.Render(strconv.Itoa(len(visible)) + " deployments  ·  newest first  ·  t table view")

	// Build the full line list first, remembering which line the cursor is on,
	// then window it around the selection.
	lines := make([]string, 0, len(visible)*2)
	selectedLine := 0
	lastDay := ""
	for i, d := range visible {
		if day := timelineDay(d.CreatedAt, now); day != lastDay {
			lastDay = day
			lines = append(lines, th.Subtle.Render(components.Truncate("── "+day+" ", width, th.Sym.Ellipsis)))
		}
		row := th.DeploymentStatusText(d.Status) + "  " +
			components.OrDash(d.ApplicationName) + "  " +
			th.Subtle.Render(domain.HumanizeAge(d.CreatedAt, now)+"  "+
				domain.HumanizeDuration(d.Duration(now))+"  "+
				components.OrDash(d.ShortCommit())) + "  " +
			components.OrDash(d.CommitMessage)
		row = components.Truncate("  "+row, width-2, th.Sym.Ellipsis)
		if i == v.selected {
			selectedLine = len(lines)
			row = th.TableRowActive.Render(components.Pad(row, width))
		}
		lines = append(lines, row)
	}

	bodyHeight := max(height-3, 1)
	start := 0
	if selectedLine >= bodyHeight {
		start = selectedLine - bodyHeight + 1
	}
	end := min(start+bodyHeight, len(lines))
	body := lipgloss.JoinVertical(lipgloss.Left, lines[start:end]...)
	block := lipgloss.JoinVertical(lipgloss.Left, header, meta, "", body)
	return components.FitBlock(block, width, height)
}

// timelineDay buckets a timestamp into a relative day label. Relative labels
// keep golden snapshots host-independent where absolute dates would not.
func timelineDay(t, now time.Time) string {
	days := int(now.Sub(t).Hours() / 24)
	switch {
	case days <= 0:
		return "Today"
	case days == 1:
		return "Yesterday"
	default:
		return strconv.Itoa(days) + " days ago"
	}
}
