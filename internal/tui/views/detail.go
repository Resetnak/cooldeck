package views

import (
	"regexp"
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
	app                     domain.Application
	loaded                  bool
	loadedAt                time.Time
	deployments             []domain.Deployment
	deploymentsLoaded       bool
	deploymentSelected      int
	deploymentLogUUID       string
	deploymentLogs          []domain.LogLine
	deploymentLogsLoaded    bool
	deploymentLogsAt        time.Time
	deploymentLogsTruncated bool
	runtimeLogs             []domain.LogLine
	runtimeLogsLoaded       bool
	runtimeLogsAt           time.Time
	runtimeTruncated        bool
	runtimePaused           bool
	runtimeFollow           bool
	runtimeWrap             bool
	logMaxScroll            int
	runtimeSearch           string
	runtimeMatch            int

	tab    DetailTab
	scroll int
}

// NewDetail returns an empty detail view.
func NewDetail() *Detail {
	return &Detail{
		deployments:    []domain.Deployment{},
		deploymentLogs: []domain.LogLine{},
		runtimeLogs:    []domain.LogLine{},
		runtimeFollow:  true,
	}
}

// SetApplication loads an application into the view. Switching to a different
// application resets the scroll position; refreshing the same one does not, so
// a background refresh cannot yank the reader back to the top.
func (v *Detail) SetApplication(a domain.Application, at time.Time) {
	if v.app.UUID != a.UUID {
		v.scroll = 0
		v.tab = TabOverview
		v.deployments = []domain.Deployment{}
		v.deploymentsLoaded = false
		v.deploymentSelected = 0
		v.deploymentLogUUID = ""
		v.deploymentLogs = []domain.LogLine{}
		v.deploymentLogsLoaded = false
		v.deploymentLogsAt = time.Time{}
		v.deploymentLogsTruncated = false
		v.runtimeLogs = []domain.LogLine{}
		v.runtimeLogsLoaded = false
		v.runtimeLogsAt = time.Time{}
		v.runtimeTruncated = false
		v.runtimePaused = false
		v.runtimeFollow = true
		v.runtimeWrap = false
		v.logMaxScroll = 0
		v.runtimeSearch = ""
		v.runtimeMatch = 0
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

// SetDeployments replaces the application's deployment history.
func (v *Detail) SetDeployments(deployments []domain.Deployment) {
	v.deployments = append([]domain.Deployment{}, deployments...)
	v.deploymentsLoaded = true
	v.deploymentSelected = min(v.deploymentSelected, max(len(v.deployments)-1, 0))
}

// SelectedDeployment returns the highlighted deployment.
func (v *Detail) SelectedDeployment() (domain.Deployment, bool) {
	if v.deploymentSelected < 0 || v.deploymentSelected >= len(v.deployments) {
		return domain.Deployment{}, false
	}
	return v.deployments[v.deploymentSelected], true
}

// MoveDeployment moves the deployment table selection and clamps it to the list.
func (v *Detail) MoveDeployment(delta int) {
	if len(v.deployments) == 0 {
		return
	}
	v.deploymentSelected = min(max(v.deploymentSelected+delta, 0), len(v.deployments)-1)
}

// OpenSelectedDeploymentLogs enters the selected deployment's log view.
func (v *Detail) OpenSelectedDeploymentLogs() (string, bool) {
	deployment, ok := v.SelectedDeployment()
	if !ok {
		return "", false
	}
	v.deploymentLogUUID = deployment.UUID
	v.deploymentLogs = []domain.LogLine{}
	v.deploymentLogsLoaded = false
	v.deploymentLogsAt = time.Time{}
	v.deploymentLogsTruncated = false
	v.scroll = 0
	return deployment.UUID, true
}

// SetDeploymentLogs stores the selected deployment's build log snapshot.
func (v *Detail) SetDeploymentLogs(uuid string, lines []domain.LogLine, at time.Time, truncated bool) {
	if uuid != v.deploymentLogUUID {
		return
	}
	v.deploymentLogs = append([]domain.LogLine{}, lines...)
	v.deploymentLogsLoaded = true
	v.deploymentLogsAt = at
	v.deploymentLogsTruncated = truncated
}

// DeploymentLogsOpen reports whether the deployments tab is showing a build log.
func (v *Detail) DeploymentLogsOpen() bool { return v.deploymentLogUUID != "" }

// CloseDeploymentLogs returns to deployment history.
func (v *Detail) CloseDeploymentLogs() {
	v.deploymentLogUUID = ""
	v.scroll = 0
}

// SetRuntimeLogs replaces the current runtime log snapshot.
func (v *Detail) SetRuntimeLogs(lines []domain.LogLine, at time.Time, truncated bool) {
	v.runtimeLogs = append([]domain.LogLine{}, lines...)
	v.runtimeLogsLoaded = true
	v.runtimeLogsAt = at
	v.runtimeTruncated = truncated
}

// ClearRuntimeLogs drops the local runtime log buffer without fetching again.
// Polling can refill it on the next tick when follow is still active.
func (v *Detail) ClearRuntimeLogs() {
	v.runtimeLogs = nil
	v.runtimeLogsLoaded = true
	v.runtimeTruncated = false
	v.runtimeSearch = ""
	v.runtimeMatch = 0
	v.scroll = 0
	v.logMaxScroll = 0
}

// RuntimeLogText returns the loaded runtime log buffer as plain text for copy.
// When a search match is active, only that line is returned so "c" can copy
// the highlighted hit; otherwise the whole buffer is copied.
func (v *Detail) RuntimeLogText() string {
	if matches := v.runtimeMatches(); len(matches) > 0 {
		idx := min(v.runtimeMatch, len(matches)-1)
		return v.runtimeLogs[matches[idx]].String()
	}
	return joinLogLines(v.runtimeLogs)
}

// RuntimeLogCount returns how many runtime log lines are currently buffered.
func (v *Detail) RuntimeLogCount() int { return len(v.runtimeLogs) }

// ClearDeploymentLogs drops the local deployment log buffer.
func (v *Detail) ClearDeploymentLogs() {
	v.deploymentLogs = nil
	v.deploymentLogsLoaded = true
	v.deploymentLogsTruncated = false
	v.scroll = 0
}

// DeploymentLogText returns the loaded deployment log buffer as plain text.
func (v *Detail) DeploymentLogText() string {
	return joinLogLines(v.deploymentLogs)
}

// DeploymentLogCount returns how many deployment log lines are buffered.
func (v *Detail) DeploymentLogCount() int { return len(v.deploymentLogs) }

// RuntimeLogsLoaded reports whether at least one log request completed.
func (v *Detail) RuntimeLogsLoaded() bool { return v.runtimeLogsLoaded }

// RuntimeLogsPaused reports whether polling is paused.
func (v *Detail) RuntimeLogsPaused() bool { return v.runtimePaused }

// ToggleRuntimePause changes polling state and returns the new value.
func (v *Detail) ToggleRuntimePause() bool {
	v.runtimePaused = !v.runtimePaused
	return v.runtimePaused
}

// RuntimeLogsFollowing reports whether new output stays pinned to the tail.
func (v *Detail) RuntimeLogsFollowing() bool { return v.runtimeFollow }

// ToggleRuntimeFollow changes tail following and returns the new value.
func (v *Detail) ToggleRuntimeFollow() bool {
	v.runtimeFollow = !v.runtimeFollow
	return v.runtimeFollow
}

// RuntimeLogsWrapping reports whether long lines wrap.
func (v *Detail) RuntimeLogsWrapping() bool { return v.runtimeWrap }

// ToggleRuntimeWrap changes long-line wrapping and returns the new value.
func (v *Detail) ToggleRuntimeWrap() bool {
	v.runtimeWrap = !v.runtimeWrap
	return v.runtimeWrap
}

// RuntimeSearch returns the current local log query.
func (v *Detail) RuntimeSearch() string { return v.runtimeSearch }

// SetRuntimeSearch replaces the local query and selects its first match.
func (v *Detail) SetRuntimeSearch(query string) {
	v.runtimeSearch = query
	v.runtimeMatch = 0
	v.jumpRuntimeMatch()
}

// NextRuntimeMatch cycles through matching loaded log lines.
func (v *Detail) NextRuntimeMatch(delta int) {
	matches := v.runtimeMatches()
	if len(matches) == 0 {
		return
	}
	v.runtimeMatch = (v.runtimeMatch + delta%len(matches) + len(matches)) % len(matches)
	v.jumpRuntimeMatch()
}

// RuntimeSearchStatus returns the one-based selected match and total count.
func (v *Detail) RuntimeSearchStatus() (int, int) {
	matches := v.runtimeMatches()
	if len(matches) == 0 {
		return 0, 0
	}
	v.runtimeMatch = min(v.runtimeMatch, len(matches)-1)
	return v.runtimeMatch + 1, len(matches)
}

func (v *Detail) runtimeMatches() []int {
	query := strings.ToLower(v.runtimeSearch)
	if query == "" {
		return nil
	}
	matches := make([]int, 0)
	for i, line := range v.runtimeLogs {
		if strings.Contains(strings.ToLower(line.Text), query) {
			matches = append(matches, i)
		}
	}
	return matches
}

func (v *Detail) jumpRuntimeMatch() {
	matches := v.runtimeMatches()
	if len(matches) == 0 {
		return
	}
	v.runtimeMatch = min(v.runtimeMatch, len(matches)-1)
	v.scroll = matches[v.runtimeMatch] + 3
	v.runtimeFollow = false
}

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
func (v *Detail) Scroll(delta int) {
	v.scroll = max(v.scroll+delta, 0)
	if v.tab == TabRuntimeLogs {
		v.scroll = min(v.scroll, v.logMaxScroll)
		v.runtimeFollow = v.scroll >= v.logMaxScroll
	}
}

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
	case TabDeployments:
		return v.renderDeployments(th, width, height, now)
	case TabRuntimeLogs:
		return v.renderRuntimeLogs(th, width, height, now)
	case TabConfiguration:
		return v.renderScrollable(th, v.configurationRows(th), width, height)
	default:
		return v.renderScrollable(th, v.overviewRows(th, now), width, height)
	}
}

func (v *Detail) renderDeployments(th *theme.Theme, width, height int, now time.Time) string {
	if v.DeploymentLogsOpen() {
		return v.renderDeploymentLogs(th, width, height, now)
	}
	if !v.deploymentsLoaded {
		return components.Skeleton(th, width, height)
	}
	if len(v.deployments) == 0 {
		return components.EmptyState(
			th,
			width,
			height,
			"No deployments yet",
			"Coolify has no deployment history for this application.",
			nil,
		)
	}

	showProgress := anyDeploymentActive(v.deployments)

	rows := make([]components.Row, 0, len(v.deployments))
	for _, deployment := range v.deployments {
		cells := []string{
			th.DeploymentStatusText(deployment.Status),
			components.OrDash(deployment.ShortCommit()),
			domain.HumanizeAge(deployment.CreatedAt, now),
			domain.HumanizeDuration(deployment.Duration(now)),
		}
		if showProgress {
			cells = append(cells, deploymentProgressCell(th, deployment, v.deployments, now))
		}
		cells = append(cells,
			components.OrDash(deployment.Trigger),
			components.OrDash(deployment.CommitMessage),
		)
		rows = append(rows, components.Row{Cells: cells})
	}

	columns := []components.Column{
		{Title: "Status", MinWidth: 12, Priority: 0},
		{Title: "Commit", MinWidth: 8, Priority: 0},
		{Title: "Started", MinWidth: 10, Priority: 1},
		{Title: "Duration", MinWidth: 10, Priority: 2},
	}
	if showProgress {
		columns = append(columns, progressColumn)
	}
	columns = append(columns,
		components.Column{Title: "Trigger", MinWidth: 8, Priority: 3},
		components.Column{Title: "Message", MinWidth: 18, Flex: 1, Priority: 4},
	)

	table := components.Table{
		Columns:  columns,
		Rows:     rows,
		Selected: v.deploymentSelected,
		Focused:  true,
	}
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		th.Title.Render("DEPLOYMENTS"),
		th.Subtle.Render(strconv.Itoa(len(v.deployments))+" recent"),
		"",
		table.Render(th, width, max(height-3, 1)),
	)
	return components.FitBlock(content, width, height)
}

func (v *Detail) renderDeploymentLogs(th *theme.Theme, width, height int, now time.Time) string {
	if !v.deploymentLogsLoaded {
		return components.Skeleton(th, width, height)
	}
	if len(v.deploymentLogs) == 0 {
		return components.EmptyState(
			th,
			width,
			height,
			"No deployment logs",
			"Coolify returned no build output for this deployment.",
			[]components.KeyHint{{Key: "esc", Desc: "back to deployments"}},
		)
	}
	meta := v.deploymentLogUUID + "  " + domain.HumanizeAge(v.deploymentLogsAt, now)
	if v.deploymentLogsTruncated {
		meta += "  truncated"
	}
	return v.renderLogs(
		th,
		width,
		height,
		logContent{title: "DEPLOYMENT LOG", meta: meta, lines: v.deploymentLogs},
	)
}

func (v *Detail) renderRuntimeLogs(th *theme.Theme, width, height int, now time.Time) string {
	if !v.runtimeLogsLoaded {
		return components.Skeleton(th, width, height)
	}
	if len(v.runtimeLogs) == 0 {
		return components.EmptyState(
			th,
			width,
			height,
			"No runtime logs",
			"The application has not emitted any output yet.",
			nil,
		)
	}

	meta := strconv.Itoa(len(v.runtimeLogs)) + " lines"
	if !v.runtimeLogsAt.IsZero() {
		meta += "  " + domain.HumanizeAge(v.runtimeLogsAt, now)
	}
	if v.runtimeTruncated {
		meta += "  truncated"
	}
	if v.runtimePaused {
		meta += "  paused"
	}
	if v.runtimeFollow {
		meta += "  follow"
	}
	if v.runtimeWrap {
		meta += "  wrap"
	}
	if current, total := v.RuntimeSearchStatus(); total > 0 {
		meta += "  search " + strconv.Itoa(current) + "/" + strconv.Itoa(total)
	} else if v.runtimeSearch != "" {
		meta += "  no matches"
	}
	return v.renderLogs(
		th,
		width,
		height,
		logContent{title: "RUNTIME LOGS", meta: meta, lines: v.runtimeLogs, wrap: v.runtimeWrap, follow: v.runtimeFollow, trackScroll: true, search: v.runtimeSearch},
	)
}

type logContent struct {
	title       string
	meta        string
	lines       []domain.LogLine
	wrap        bool
	follow      bool
	trackScroll bool
	search      string
}

func (v *Detail) renderLogs(th *theme.Theme, width, height int, content logContent) string {
	lines := []string{th.Title.Render(content.title), th.Subtle.Render(content.meta), ""}
	var search *regexp.Regexp
	if content.search != "" {
		search = regexp.MustCompile("(?i)" + regexp.QuoteMeta(content.search))
	}
	for _, line := range content.lines {
		timestamp := line.TimestampText
		if timestamp == "" && !line.Timestamp.IsZero() {
			timestamp = line.Timestamp.Format("15:04:05")
		}
		prefix := ""
		if timestamp != "" {
			prefix = th.Subtle.Render(timestamp) + " "
		}
		if label := line.Level.Label(); label != "" {
			prefix += logLevelStyle(th, line.Level).Render(components.Pad(label, 5)) + " "
		}
		text := prefix + line.Text
		if search != nil {
			text = search.ReplaceAllStringFunc(text, func(match string) string {
				return th.FilterPrompt.Render(match)
			})
		}
		if content.wrap {
			lines = append(lines, strings.Split(components.Wrap(text, width), "\n")...)
		} else {
			lines = append(lines, components.Truncate(text, width, th.Sym.Ellipsis))
		}
	}

	maxScroll := max(len(lines)-height, 0)
	if content.trackScroll {
		v.logMaxScroll = maxScroll
	}
	if content.follow {
		v.scroll = maxScroll
	}
	v.scroll = min(v.scroll, maxScroll)
	return components.FitBlock(strings.Join(lines[v.scroll:], "\n"), width, height)
}

func logLevelStyle(th *theme.Theme, level domain.LogLevel) lipgloss.Style {
	switch level {
	case domain.LogLevelWarn:
		return th.Attention
	case domain.LogLevelError, domain.LogLevelFatal:
		return th.Danger
	case domain.LogLevelInfo:
		return th.Note
	default:
		return th.Subtle
	}
}

func joinLogLines(lines []domain.LogLine) string {
	if len(lines) == 0 {
		return ""
	}
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, line.String())
	}
	return strings.Join(parts, "\n")
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
