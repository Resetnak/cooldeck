package views

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// InstanceRow is one configured Coolify instance as shown in the instances list.
// Secrets are never present: only non-sensitive metadata and connection state.
type InstanceRow struct {
	ID          string
	Name        string
	URL         string
	TokenSource string
	Active      bool
	Demo        bool
	Version     string
	Team        string
	Latency     time.Duration
	Connection  components.ConnectionState
	LastError   string
	ConnectedAt time.Time
}

// Instances is the instances management list.
type Instances struct {
	items    []InstanceRow
	selected int
}

// NewInstances returns an empty instances view.
func NewInstances() *Instances {
	return &Instances{items: []InstanceRow{}}
}

// SetItems replaces the list, preserving selection by ID when possible.
func (v *Instances) SetItems(items []InstanceRow) {
	var keep string
	if r, ok := v.Selected(); ok {
		keep = r.ID
	}
	v.items = append([]InstanceRow{}, items...)
	v.selected = 0
	if keep != "" {
		for i, item := range v.items {
			if item.ID == keep {
				v.selected = i
				break
			}
		}
	}
	if v.selected >= len(v.items) {
		v.selected = max(len(v.items)-1, 0)
	}
}

// Count returns how many instances are listed.
func (v *Instances) Count() int { return len(v.items) }

// Selected returns the highlighted row.
func (v *Instances) Selected() (InstanceRow, bool) {
	if v.selected < 0 || v.selected >= len(v.items) {
		return InstanceRow{}, false
	}
	return v.items[v.selected], true
}

// Move shifts the selection.
func (v *Instances) Move(delta int) {
	if len(v.items) == 0 {
		return
	}
	v.selected = min(max(v.selected+delta, 0), len(v.items)-1)
}

// MoveTo jumps to an absolute index.
func (v *Instances) MoveTo(idx int) {
	if len(v.items) == 0 {
		return
	}
	v.selected = min(max(idx, 0), len(v.items)-1)
}

// Render draws the instances list.
func (v *Instances) Render(th *theme.Theme, width, height int, focused bool, now time.Time) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(v.items) == 0 {
		return components.EmptyState(
			th,
			width,
			height,
			"No instances configured",
			"Run cooldeck setup to add a Coolify instance, or start with --demo.",
			[]components.KeyHint{
				{Key: "q", Desc: "quit"},
			},
		)
	}

	rows := make([]components.Row, 0, len(v.items))
	for _, item := range v.items {
		marker := ""
		if item.Active {
			marker = th.Sym.Selected + " "
		}
		name := marker + item.Name
		if item.Demo {
			name += " (demo)"
		}
		state := connectionLabel(th, item.Connection)
		latency := "—"
		if item.Latency > 0 {
			latency = item.Latency.Round(time.Millisecond).String()
		}
		auth := item.TokenSource
		if auth == "" {
			auth = "—"
		}
		rows = append(rows, components.Row{Cells: []string{
			name,
			components.OrDash(item.URL),
			state,
			components.OrDash(item.Version),
			latency,
			auth,
		}})
	}

	header := th.Title.Render("INSTANCES")
	meta := th.Subtle.Render(strconv.Itoa(len(v.items)) + " configured")
	table := components.Table{
		Columns: []components.Column{
			{Title: "Name", MinWidth: 12, Flex: 1, Priority: 0},
			{Title: "URL", MinWidth: 18, Flex: 1, Priority: 0},
			{Title: "State", MinWidth: 10, Priority: 1},
			{Title: "Version", MinWidth: 10, Priority: 2},
			{Title: "Latency", MinWidth: 8, Priority: 3},
			{Title: "Auth", MinWidth: 8, Priority: 3},
		},
		Rows:     rows,
		Selected: v.selected,
		Focused:  focused,
	}

	parts := []string{header, meta, "", table.Render(th, width, max(height-6, 1))}
	if sel, ok := v.Selected(); ok {
		parts = append(parts, "", v.detailBlock(th, sel, width, now))
	}
	return components.FitBlock(strings.Join(parts, "\n"), width, height)
}

func (v *Instances) detailBlock(th *theme.Theme, item InstanceRow, width int, now time.Time) string {
	lines := []string{
		th.Subtle.Render("ID " + item.ID),
	}
	if item.Team != "" {
		lines = append(lines, th.Subtle.Render("Team "+item.Team))
	}
	if !item.ConnectedAt.IsZero() {
		lines = append(lines, th.Subtle.Render("Connected "+domainAge(item.ConnectedAt, now)))
	}
	if item.LastError != "" {
		lines = append(lines, th.Danger.Render(components.Truncate(item.LastError, max(width-2, 8), th.Sym.Ellipsis)))
	}
	if item.Active {
		lines = append(lines, th.Note.Render("Active instance for this session"))
	} else {
		lines = append(lines, th.Subtle.Render("Enter switches to this instance for the current session"))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func connectionLabel(th *theme.Theme, state components.ConnectionState) string {
	switch state {
	case components.ConnectionOnline:
		return th.Positive.Render("online")
	case components.ConnectionOffline:
		return th.Danger.Render("offline")
	case components.ConnectionUnauthorized:
		return th.Attention.Render("unauthorized")
	case components.ConnectionConnecting:
		return th.Muted.Render("connecting")
	default:
		return th.Subtle.Render("unknown")
	}
}

// domainAge avoids importing domain in the connection helpers above for one call;
// keep formatting consistent with the rest of the UI.
func domainAge(t, now time.Time) string {
	d := now.Sub(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return strconv.Itoa(int(d.Minutes())) + "m ago"
	}
	if d < 24*time.Hour {
		return strconv.Itoa(int(d.Hours())) + "h ago"
	}
	return strconv.Itoa(int(d.Hours()/24)) + "d ago"
}
