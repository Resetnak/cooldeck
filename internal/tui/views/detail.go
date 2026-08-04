package views

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// DetailTab is a section of the application detail screen.
type DetailTab int

// Detail tabs.
const (
	TabOverview DetailTab = iota
	TabDeployments
	TabRuntimeLogs
	TabConfiguration
	detailTabCount
)

// Label names the tab.
func (t DetailTab) Label() string {
	switch t {
	case TabDeployments:
		return "Deployments"
	case TabRuntimeLogs:
		return "Runtime Logs"
	case TabConfiguration:
		return "Configuration"
	default:
		return "Overview"
	}
}

// Detail is the application detail screen.
type Detail struct {
	app      domain.Application
	loaded   bool
	loadedAt time.Time

	tab    DetailTab
	scroll int
}

// NewDetail returns an empty detail view.
func NewDetail() *Detail { return &Detail{} }

// SetApplication loads an application into the view. Switching to a different
// application resets the scroll position; refreshing the same one does not, so
// a background refresh cannot yank the reader back to the top.
func (v *Detail) SetApplication(a domain.Application, at time.Time) {
	if v.app.UUID != a.UUID {
		v.scroll = 0
		v.tab = TabOverview
	}
	v.app = a
	v.loaded = true
	v.loadedAt = at
}

// Reset clears the view when leaving the screen.
func (v *Detail) Reset() { *v = Detail{} }

// Application returns the displayed application.
func (v *Detail) Application() domain.Application { return v.app }

// Loaded reports whether an application has been set.
func (v *Detail) Loaded() bool { return v.loaded }

// Tab returns the active tab.
func (v *Detail) Tab() DetailTab { return v.tab }

// SetTab switches tabs.
func (v *Detail) SetTab(t DetailTab) {
	if t >= 0 && t < detailTabCount {
		v.tab = t
		v.scroll = 0
	}
}

// NextTab and PrevTab cycle through the sections.
func (v *Detail) NextTab() { v.SetTab((v.tab + 1) % detailTabCount) }

// PrevTab moves to the previous section.
func (v *Detail) PrevTab() { v.SetTab((v.tab - 1 + detailTabCount) % detailTabCount) }

// Scroll moves the overview body by delta lines.
func (v *Detail) Scroll(delta int) { v.scroll = max(v.scroll+delta, 0) }

// TabBar renders the section tabs for the detail screen.
func (v *Detail) TabBar(th *theme.Theme, width int, compact bool) string {
	items := make([]components.NavItem, 0, detailTabCount)
	for t := DetailTab(0); t < detailTabCount; t++ {
		label := t.Label()
		short := label
		if i := strings.IndexByte(label, ' '); i > 0 {
			short = label[:i]
		}
		items = append(items, components.NavItem{Label: label, Short: short, Count: -1, Enabled: true})
	}
	return components.Tabs(th, items, int(v.tab), width, compact)
}

// Render draws the active tab's body.
func (v *Detail) Render(th *theme.Theme, width, height int, now time.Time) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if !v.loaded {
		return components.Skeleton(th, width, height)
	}

	switch v.tab {
	case TabConfiguration:
		return v.renderScrollable(th, v.configurationRows(th), width, height)
	default:
		return v.renderScrollable(th, v.overviewRows(th, now), width, height)
	}
}

// field is one label/value pair in the detail body.
type field struct {
	label string
	value string
	// blank inserts a spacer instead of a row.
	blank bool
	// heading renders the value as a section title.
	heading bool
}

func (v *Detail) overviewRows(th *theme.Theme, now time.Time) []field {
	a := v.app

	rows := []field{
		{heading: true, value: "Overview"},
		{label: "Status", value: th.StatusText(a.Status)},
	}
	if a.Status.Health != domain.HealthNone {
		rows = append(rows, field{label: "Health", value: string(a.Status.Health)})
	}
	if a.Description != "" {
		rows = append(rows, field{label: "Description", value: a.Description})
	}

	rows = append(rows,
		field{label: "UUID", value: a.UUID},
		field{blank: true},
		field{heading: true, value: "Placement"},
		field{label: "Project", value: components.OrDash(a.Project.String())},
		field{label: "Environment", value: v.environmentValue(th)},
		field{label: "Server", value: components.OrDash(a.Server.String())},
		field{blank: true},
		field{heading: true, value: "Source"},
		field{label: "Repository", value: components.OrDash(a.RepositoryURL)},
		field{label: "Branch", value: components.OrDash(a.Branch)},
		field{label: "Commit", value: components.OrDash(a.ShortCommit())},
		field{label: "Build pack", value: components.OrDash(a.BuildPack)},
	)

	if len(a.FQDNs) > 0 {
		rows = append(rows, field{blank: true}, field{heading: true, value: "Domains"})
		for _, d := range a.FQDNs {
			rows = append(rows, field{label: "", value: th.Note.Render(d)})
		}
	}

	rows = append(rows, field{blank: true}, field{heading: true, value: "Last deployment"})
	if a.LastDeployment != nil {
		d := a.LastDeployment
		rows = append(rows,
			field{label: "Result", value: th.DeploymentStatusText(d.Status)},
			field{label: "Started", value: domain.HumanizeAge(d.StartedAt, now) +
				th.Subtle.Render("  ("+domain.HumanizeTime(d.StartedAt)+")")},
			field{label: "Duration", value: domain.HumanizeDuration(d.Duration)},
			field{label: "Commit", value: components.OrDash(domain.ShortSHA(d.CommitSHA))},
		)
		if d.Message != "" {
			rows = append(rows, field{label: "Message", value: d.Message})
		}
	} else {
		rows = append(rows, field{label: "", value: th.Muted.Render("No deployment recorded.")})
	}

	rows = append(rows,
		field{blank: true},
		field{heading: true, value: "Timestamps"},
		field{label: "Created", value: domain.HumanizeTime(a.CreatedAt)},
		field{label: "Updated", value: domain.HumanizeTime(a.UpdatedAt)},
	)
	return rows
}

func (v *Detail) configurationRows(th *theme.Theme) []field {
	a := v.app

	rows := []field{{heading: true, value: "Health check"}}
	if a.HealthCheck.Enabled {
		rows = append(rows,
			field{label: "Endpoint", value: a.HealthCheck.Summary()},
			field{label: "Interval", value: domain.HumanizeDuration(a.HealthCheck.Interval)},
			field{label: "Timeout", value: domain.HumanizeDuration(a.HealthCheck.Timeout)},
			field{label: "Retries", value: strconv.Itoa(a.HealthCheck.Retries)},
		)
	} else {
		rows = append(rows, field{label: "", value: th.Muted.Render("Health checks are disabled.")})
	}

	rows = append(rows,
		field{blank: true},
		field{heading: true, value: "Build"},
		field{label: "Build pack", value: components.OrDash(a.BuildPack)},
		field{label: "Repository", value: components.OrDash(a.RepositoryURL)},
		field{label: "Branch", value: components.OrDash(a.Branch)},
		field{blank: true},
		// Environment variables, secrets and raw compose files are deliberately
		// absent: cooldeck never displays values that can carry credentials.
		field{label: "", value: th.Subtle.Render(
			"Environment variables and secrets are not shown by design.")},
		field{label: "", value: th.Subtle.Render(
			"Manage them in the Coolify dashboard.")},
	)
	return rows
}

func (v *Detail) environmentValue(th *theme.Theme) string {
	env := components.OrDash(v.app.Environment.String())
	if v.app.IsProduction() {
		return env + "  " + th.Badge(theme.BadgeProduction, "PRODUCTION")
	}
	return env
}

// renderScrollable lays out label/value rows and applies the scroll offset.
func (v *Detail) renderScrollable(th *theme.Theme, rows []field, width, height int) string {
	const labelWidth = 14
	valueWidth := max(width-labelWidth-3, 8)

	var lines []string
	for _, f := range rows {
		switch {
		case f.blank:
			lines = append(lines, "")
		case f.heading:
			lines = append(lines, th.Title.Render(strings.ToUpper(f.value)))
		case f.label == "":
			lines = append(lines, strings.Repeat(" ", labelWidth+1)+
				components.Truncate(f.value, valueWidth, th.Sym.Ellipsis))
		default:
			label := th.Label.Render(components.Pad(f.label, labelWidth))
			lines = append(lines, label+" "+components.Truncate(f.value, valueWidth, th.Sym.Ellipsis))
		}
	}

	// Clamp the scroll offset so the body cannot be scrolled past its end.
	maxScroll := max(len(lines)-height, 0)
	v.scroll = min(v.scroll, maxScroll)
	lines = lines[v.scroll:]

	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return components.FitBlock(body, width, height)
}

// Preview renders the compact side panel shown next to the table on wide
// terminals. It is a summary, not the full detail screen.
func Preview(th *theme.Theme, a domain.Application, width, height int, now time.Time) string {
	if width <= 2 || height <= 2 {
		return ""
	}
	inner := width - 2

	lines := []string{
		th.Title.Render(components.Truncate(a.Name, inner, th.Sym.Ellipsis)),
		th.StatusText(a.Status),
		"",
	}
	if a.IsProduction() {
		lines = append(lines, th.Badge(theme.BadgeProduction, "PRODUCTION"), "")
	}

	add := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		lines = append(lines,
			th.Label.Render(components.Pad(label, 10))+" "+
				components.Truncate(value, inner-11, th.Sym.Ellipsis))
	}
	add("Project", a.Project.String())
	add("Env", a.Environment.String())
	add("Server", a.Server.String())
	add("Branch", a.Branch)
	add("Commit", a.ShortCommit())
	add("Domain", a.PrimaryDomain())

	if a.LastDeployment != nil {
		lines = append(lines, "", th.Label.Render("LAST DEPLOYMENT"),
			th.DeploymentStatusText(a.LastDeployment.Status),
			th.Muted.Render(domain.HumanizeAge(a.LastDeployment.StartedAt, now)+
				"  "+domain.HumanizeDuration(a.LastDeployment.Duration)))
		if a.LastDeployment.Message != "" {
			lines = append(lines, th.Subtle.Render(
				components.Truncate(a.LastDeployment.Message, inner, th.Sym.Ellipsis)))
		}
	}

	body := components.FitBlock(strings.Join(lines, "\n"), inner, height-2)
	return th.Panel.Width(inner).Render(body)
}
