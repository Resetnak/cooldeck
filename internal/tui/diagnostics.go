package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/logging"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
	"github.com/resetnak/cooldeck/internal/tui/views"
	"github.com/resetnak/cooldeck/internal/version"
)

// buildDiagnosticFields assembles the diagnostics screen. Values are purely
// observational and never include tokens or response bodies.
func (m *Model) buildDiagnosticFields() []views.DiagnosticField {
	info := version.Current()
	fields := []views.DiagnosticField{
		{Section: "cooldeck", Label: "Version", Value: info.Version},
		{Label: "Commit", Value: info.Commit},
		{Label: "Built", Value: info.Date},
		{Label: "Go", Value: info.GoVer},
		{Label: "Platform", Value: info.Platform},
		{Label: "GOOS/GOARCH", Value: runtime.GOOS + "/" + runtime.GOARCH},

		{Section: "paths", Label: "Config", Value: firstNonEmpty(m.opts.ConfigPath, m.opts.Config.Path(), "(unset)")},
		{Label: "Log file", Value: firstNonEmpty(m.opts.LogPath, "(unset)")},

		{Section: "instance", Label: "Name", Value: m.instanceLabel()},
		{Label: "ID", Value: m.service.InstanceID()},
		{Label: "URL", Value: firstNonEmpty(m.lastConnection.BaseURL, activeInstance(m).URL, "(unknown)")},
		{Label: "Coolify", Value: firstNonEmpty(m.coolifyVer, "(unknown)")},
		{Label: "Team", Value: firstNonEmpty(m.lastConnection.TeamName, "-")},
		{Label: "Connection", Value: connectionStateLabel(m.connection)},
		{Label: "Latency", Value: formatLatency(m.lastConnection.Latency)},
		{Label: "Demo mode", Value: strconv.FormatBool(m.opts.Demo)},
	}

	caps := m.capabilities
	fields = append(fields,
		views.DiagnosticField{Section: "capabilities", Label: "Applications", Value: boolMark(caps.Applications)},
		views.DiagnosticField{Label: "Logs", Value: boolMark(caps.ApplicationLogs)},
		views.DiagnosticField{Label: "Deployments", Value: boolMark(caps.Deployments)},
		views.DiagnosticField{Label: "Deploy", Value: boolMark(caps.Deploy)},
		views.DiagnosticField{Label: "Restart", Value: boolMark(caps.Restart)},
		views.DiagnosticField{Label: "Start/Stop", Value: boolMark(caps.StartStop)},
	)

	fields = append(fields,
		views.DiagnosticField{Section: "terminal", Label: "Size", Value: fmt.Sprintf("%dx%d", m.width, m.height)},
		views.DiagnosticField{Label: "Layout", Value: layoutName(m.layout.Break)},
		views.DiagnosticField{Label: "Theme", Value: string(m.themeMode)},
		views.DiagnosticField{Label: "Dark terminal", Value: strconv.FormatBool(m.darkTerm)},
		views.DiagnosticField{Label: "Mouse", Value: strconv.FormatBool(m.opts.Mouse)},
		views.DiagnosticField{Label: "ASCII glyphs", Value: strconv.FormatBool(m.opts.ASCII)},
		views.DiagnosticField{Label: "Nerd Font", Value: string(m.opts.Config.UI.NerdFont)},
		views.DiagnosticField{Label: "Compact", Value: strconv.FormatBool(m.forceCompact || m.layout.IsCompact())},
		views.DiagnosticField{Label: "Log lines", Value: strconv.Itoa(m.logLines)},
		views.DiagnosticField{Label: "Refresh", Value: time.Duration(m.opts.Config.RefreshInterval).String()},
	)

	errors := append([]string{}, m.recentErrors...)
	if m.opts.RecentErrors != nil {
		errors = append(errors, m.opts.RecentErrors()...)
	}
	if len(errors) == 0 {
		fields = append(fields, views.DiagnosticField{Section: "recent errors", Label: "Status", Value: "none"})
	} else {
		// Newest last in buffer; show newest first.
		for i := len(errors) - 1; i >= 0; i-- {
			label := fmt.Sprintf("#%d", len(errors)-i)
			if i == len(errors)-1 {
				fields = append(fields, views.DiagnosticField{Section: "recent errors", Label: label, Value: errors[i]})
			} else {
				fields = append(fields, views.DiagnosticField{Label: label, Value: errors[i]})
			}
		}
	}
	return fields
}

func connectionStateLabel(state components.ConnectionState) string {
	switch state {
	case components.ConnectionOnline:
		return "online"
	case components.ConnectionOffline:
		return "offline"
	case components.ConnectionUnauthorized:
		return "unauthorized"
	case components.ConnectionConnecting:
		return "connecting"
	default:
		return "unknown"
	}
}

func formatLatency(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	return d.Round(time.Millisecond).String()
}

func boolMark(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func layoutName(b theme.Breakpoint) string {
	switch b {
	case theme.BreakpointWide:
		return "wide"
	case theme.BreakpointStandard:
		return "standard"
	case theme.BreakpointCompact:
		return "compact"
	case theme.BreakpointTooSmall:
		return "too-small"
	default:
		return "unknown"
	}
}

// copyDiagnostics copies the plain-text diagnostics dump to the clipboard.
func (m *Model) copyDiagnostics() tea.Cmd {
	m.refreshDiagnostics()
	text := m.diagnostics.PlainText()
	if text == "" {
		return m.pushToast(components.ToastWarning, "Nothing to copy", "")
	}
	return m.copyText(text, "Copied diagnostics")
}

// exportDiagnostics writes diagnostics to a file under the state directory.
func (m *Model) exportDiagnostics() tea.Cmd {
	m.refreshDiagnostics()
	// Same belt-and-braces pass as the fleet snapshot: the fields are built to
	// be secret-free, but this file is the one users attach to bug reports.
	text := logging.Redact(m.diagnostics.PlainText())
	if text == "" {
		return m.pushToast(components.ToastWarning, "Nothing to export", "")
	}
	return func() tea.Msg {
		path, err := writeDiagnosticsFile(text)
		if err != nil {
			return toastMsg{Kind: int(components.ToastError), Text: "Export failed", Detail: err.Error()}
		}
		return toastMsg{Kind: int(components.ToastSuccess), Text: "Exported diagnostics", Detail: path}
	}
}

func writeDiagnosticsFile(text string) (string, error) {
	dir, err := config.StateDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	name := fmt.Sprintf("diagnostics-%s.txt", time.Now().UTC().Format("20060102-150405"))
	path := filepath.Join(dir, name)
	// 0600 so a multi-user machine cannot read another user's diagnostics.
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// testActiveConnection re-runs Connect for the active instance.
func (m *Model) testActiveConnection() tea.Cmd {
	return m.connect()
}

// describeInstanceSwitch explains how to switch without performing a mid-session
// service swap (which would require reloading credentials and the whole model).
func (m *Model) describeInstanceSwitch(row views.InstanceRow) tea.Cmd {
	if row.Active {
		return m.pushToast(components.ToastInfo, "Already active", row.Name+" is the current instance.")
	}
	if row.Demo {
		return m.pushToast(components.ToastInfo, "Demo instance", "Restart without --demo to use a real Coolify instance.")
	}
	msg := "Restart with: cooldeck --instance " + row.ID
	return m.pushToast(components.ToastInfo, "Switch instance", msg)
}
