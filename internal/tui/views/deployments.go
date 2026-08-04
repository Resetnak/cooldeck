package views

import (
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

func (v *Deployments) visible() []domain.Deployment {
	if !v.activeOnly {
		return v.items
	}
	out := make([]domain.Deployment, 0, len(v.items))
	for _, d := range v.items {
		if d.Status.IsActive() {
			out = append(out, d)
		}
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

	rows := make([]components.Row, 0, len(visible))
	for _, d := range visible {
		name := d.ApplicationName
		if name == "" {
			name = d.ApplicationUUID
		}
		rows = append(rows, components.Row{Cells: []string{
			th.DeploymentStatusText(d.Status),
			components.OrDash(name),
			components.OrDash(d.ShortCommit()),
			domain.HumanizeAge(d.CreatedAt, now),
			domain.HumanizeDuration(d.Duration(now)),
			components.OrDash(d.Trigger),
			components.OrDash(d.CommitMessage),
		}})
	}

	header := th.Title.Render("DEPLOYMENTS")
	mode := "recent history"
	if v.activeOnly {
		mode = "active only"
	}
	meta := th.Subtle.Render(strconv.Itoa(len(visible)) + " shown  ·  " +
		strconv.Itoa(v.ActiveCount()) + " active  ·  " + mode + "  ·  a toggle")
	table := components.Table{
		Columns: []components.Column{
			{Title: "Status", MinWidth: 12, Priority: 0},
			{Title: "Application", MinWidth: 14, Flex: 1, Priority: 0},
			{Title: "Commit", MinWidth: 8, Priority: 1},
			{Title: "Started", MinWidth: 10, Priority: 2},
			{Title: "Duration", MinWidth: 10, Priority: 2},
			{Title: "Trigger", MinWidth: 8, Priority: 3},
			{Title: "Message", MinWidth: 16, Flex: 1, Priority: 4},
		},
		Rows:     rows,
		Selected: v.selected,
		Focused:  focused,
	}
	body := lipgloss.JoinVertical(lipgloss.Left, header, meta, "", table.Render(th, width, max(height-3, 1)))
	return components.FitBlock(body, width, height)
}
