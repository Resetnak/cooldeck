// Package views holds the screens. A view owns its own selection, scroll and
// filter state and renders itself; it never performs I/O. Data arrives through
// explicit setters called by the root model, which keeps the message flow
// one-directional and the views trivially testable.
package views

import (
	"sort"
	"strings"
	"time"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// SortMode orders the application table.
type SortMode int

// Available sort modes.
const (
	// SortByStatus surfaces the applications that need attention first, which
	// is what the dashboard is for.
	SortByStatus SortMode = iota
	SortByName
	SortByDeployed
	sortModeCount
)

// Label names the sort mode for the footer and command palette.
func (s SortMode) Label() string {
	switch s {
	case SortByName:
		return "name"
	case SortByDeployed:
		return "last deploy"
	default:
		return "status"
	}
}

// Next cycles to the following sort mode.
func (s SortMode) Next() SortMode { return (s + 1) % sortModeCount }

// Applications is the dashboard table of applications.
type Applications struct {
	// all holds every application from the last successful load.
	all []domain.Application
	// visible is the filtered and sorted projection, recomputed only when the
	// data, the filter or the sort mode changes, never during rendering.
	visible []domain.Application

	activeDeployments map[string]domain.Deployment

	filter domain.Filter
	sort   SortMode

	// selectedUUID anchors the selection to an application rather than to a
	// row index, so a refresh that reorders the table does not move the cursor
	// onto a different application.
	selectedUUID string
	selected     int
	offset       int

	loaded   bool
	loadedAt time.Time
}

// NewApplications returns an empty applications view.
func NewApplications() *Applications {
	return &Applications{activeDeployments: map[string]domain.Deployment{}}
}

// SetApplications replaces the data set, preserving the selected application.
func (v *Applications) SetApplications(apps []domain.Application, active []domain.Deployment, at time.Time) {
	v.all = apps
	v.loaded = true
	v.loadedAt = at

	v.activeDeployments = make(map[string]domain.Deployment, len(active))
	for _, d := range active {
		v.activeDeployments[d.ApplicationUUID] = d
	}
	v.reproject()
}

// Loaded reports whether data has arrived at least once. Until it has, the
// view renders skeleton rows rather than an empty state.
func (v *Applications) Loaded() bool { return v.loaded }

// LoadedAt returns when the current data was fetched.
func (v *Applications) LoadedAt() time.Time { return v.loadedAt }

// Count returns the number of applications after filtering.
func (v *Applications) Count() int { return len(v.visible) }

// Total returns the number of applications before filtering.
func (v *Applications) Total() int { return len(v.all) }

// Filter returns the active filter.
func (v *Applications) Filter() domain.Filter { return v.filter }

// SetFilter applies a new filter query and re-anchors the selection.
func (v *Applications) SetFilter(query string) {
	v.filter = domain.ParseFilter(query)
	v.reproject()
}

// SortMode returns the active sort mode.
func (v *Applications) SortMode() SortMode { return v.sort }

// SetSort changes the ordering.
func (v *Applications) SetSort(mode SortMode) {
	v.sort = mode
	v.reproject()
}

// Selected returns the highlighted application.
func (v *Applications) Selected() (domain.Application, bool) {
	if v.selected < 0 || v.selected >= len(v.visible) {
		return domain.Application{}, false
	}
	return v.visible[v.selected], true
}

// SelectedIndex returns the row index of the selection, or -1 when there is none.
func (v *Applications) SelectedIndex() int {
	if v.selected < 0 || v.selected >= len(v.visible) {
		return -1
	}
	return v.selected
}

// At returns the application at a visible row.
func (v *Applications) At(i int) (domain.Application, bool) {
	if i < 0 || i >= len(v.visible) {
		return domain.Application{}, false
	}
	return v.visible[i], true
}

// ActiveDeployment returns the in-flight deployment for an application.
func (v *Applications) ActiveDeployment(uuid string) (domain.Deployment, bool) {
	d, ok := v.activeDeployments[uuid]
	return d, ok
}

// Move shifts the selection by delta rows and returns whether it changed.
func (v *Applications) Move(delta int) bool {
	if len(v.visible) == 0 {
		return false
	}
	next := min(max(v.selected+delta, 0), len(v.visible)-1)
	if next == v.selected {
		return false
	}
	v.selected = next
	v.rememberSelection()
	return true
}

// MoveTo jumps the selection to an absolute row.
func (v *Applications) MoveTo(index int) {
	if len(v.visible) == 0 {
		v.selected = -1
		return
	}
	v.selected = min(max(index, 0), len(v.visible)-1)
	v.rememberSelection()
}

// SelectUUID moves the selection onto a specific application if it is visible.
func (v *Applications) SelectUUID(uuid string) bool {
	for i, a := range v.visible {
		if a.UUID == uuid {
			v.selected = i
			v.selectedUUID = uuid
			return true
		}
	}
	return false
}

func (v *Applications) rememberSelection() {
	if a, ok := v.Selected(); ok {
		v.selectedUUID = a.UUID
	}
}

// reproject rebuilds the filtered and sorted slice. It is the only place the
// visible set is computed, so rendering stays free of per-frame work.
func (v *Applications) reproject() {
	v.visible = v.visible[:0]
	for _, a := range v.all {
		if v.filter.MatchApplication(a) {
			v.visible = append(v.visible, a)
		}
	}

	less := map[SortMode]func(a, b domain.Application) bool{
		SortByStatus: func(a, b domain.Application) bool {
			if sa, sb := a.Status.Severity(), b.Status.Severity(); sa != sb {
				return sa < sb
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		},
		SortByName: func(a, b domain.Application) bool {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		},
		SortByDeployed: func(a, b domain.Application) bool {
			return deployedAt(a).After(deployedAt(b))
		},
	}[v.sort]
	sort.SliceStable(v.visible, func(i, j int) bool { return less(v.visible[i], v.visible[j]) })

	// Re-anchor onto the previously selected application; fall back to the
	// nearest valid row so the cursor never disappears.
	v.selected = -1
	for i, a := range v.visible {
		if a.UUID == v.selectedUUID {
			v.selected = i
			break
		}
	}
	if v.selected < 0 && len(v.visible) > 0 {
		v.selected = 0
		v.selectedUUID = v.visible[0].UUID
	}
}

func deployedAt(a domain.Application) time.Time {
	if a.LastDeployment != nil && !a.LastDeployment.StartedAt.IsZero() {
		return a.LastDeployment.StartedAt
	}
	return a.UpdatedAt
}

// applicationColumns defines the table shape once. Priority drives which
// columns survive on a narrow terminal: status and name never drop.
func applicationColumns() []components.Column {
	return []components.Column{
		{Title: "Status", MinWidth: 12, Priority: 0},
		{Title: "Application", MinWidth: 16, Flex: 3, MaxWidth: 40, Priority: 0},
		{Title: "Project / Env", ShortTitle: "Project", MinWidth: 16, Flex: 2, MaxWidth: 30, Priority: 3},
		{Title: "Branch", MinWidth: 8, Flex: 1, MaxWidth: 18, Priority: 2},
		{Title: "Deployed", MinWidth: 10, Priority: 1, Right: true},
		{Title: "Domain", MinWidth: 16, Flex: 3, MaxWidth: 44, Priority: 4},
	}
}

// Render draws the table, or the appropriate empty, loading or filtered-out
// state. It never mutates the view.
func (v *Applications) Render(th *theme.Theme, width, height int, focused bool, now time.Time) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if !v.loaded {
		return components.Skeleton(th, width, height)
	}
	if len(v.all) == 0 {
		return components.EmptyState(th, width, height,
			"No applications on this instance",
			"Check that the API token belongs to the right team, or create an application in the Coolify dashboard.",
			[]components.KeyHint{{Key: "R", Desc: "refresh"}, {Key: ":", Desc: "commands"}})
	}
	if len(v.visible) == 0 {
		return components.EmptyState(th, width, height,
			"No applications match this filter",
			"Filter: "+v.filter.Raw,
			[]components.KeyHint{{Key: "esc", Desc: "clear filter"}, {Key: "/", Desc: "edit filter"}})
	}

	rows := make([]components.Row, 0, len(v.visible))
	for _, a := range v.visible {
		rows = append(rows, components.Row{Key: a.UUID, Cells: v.cells(th, a, now)})
	}

	visible := components.VisibleRows(height)
	v.offset = components.ClampOffset(v.offset, v.selected, len(rows), visible)

	tbl := components.Table{
		Columns:  applicationColumns(),
		Rows:     rows,
		Selected: v.selected,
		Offset:   v.offset,
		Focused:  focused,
	}
	return tbl.Render(th, width, height)
}

func (v *Applications) cells(th *theme.Theme, a domain.Application, now time.Time) []string {
	status := th.StatusText(a.Status)
	// An in-flight deployment outranks the container status: it is the thing
	// the user is waiting on.
	if d, ok := v.activeDeployments[a.UUID]; ok {
		status = th.DeploymentStatusText(d.Status)
	}

	name := a.Name
	if a.IsProduction() {
		// Production is marked with a glyph rather than colour alone.
		name = th.Danger.Render(th.Sym.Bullet) + " " + name
	}

	projectEnv := a.Project.String()
	if env := a.Environment.String(); env != "" {
		if projectEnv == "" {
			projectEnv = env
		} else {
			projectEnv += " / " + env
		}
	}

	deployed := components.Dash
	if d, ok := v.activeDeployments[a.UUID]; ok {
		deployed = th.Attention.Render(domain.HumanizeDuration(d.Duration(now)))
	} else if a.LastDeployment != nil && !a.LastDeployment.StartedAt.IsZero() {
		deployed = th.TableCellMuted.Render(domain.HumanizeAge(a.LastDeployment.StartedAt, now))
	}

	return []string{
		status,
		name,
		th.TableCellMuted.Render(components.OrDash(projectEnv)),
		th.TableCellMuted.Render(components.OrDash(a.Branch)),
		deployed,
		th.TableCellMuted.Render(components.OrDash(a.PrimaryDomain())),
	}
}
