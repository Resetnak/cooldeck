package tui

import (
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/logging"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

// snapshotDeploymentLimit bounds the history section so the snapshot stays
// pasteable into a chat window.
const snapshotDeploymentLimit = 20

// copyFleetSnapshot copies a markdown digest of the whole fleet - statuses,
// recent deployments, recent errors - to the clipboard. It is the panic
// button: one keypress produces something ready to paste at an AI assistant
// or a colleague. Secret-free by construction, like the diagnostics export.
func (m *Model) copyFleetSnapshot() tea.Cmd {
	if !m.apps.Loaded() {
		return m.pushToast(components.ToastWarning, "Nothing to snapshot", "The fleet has not loaded yet.")
	}
	text := fleetSnapshotMarkdown(m.lastConnection, m.apps.All(), m.deployments.Items(), m.recentErrors, m.now())
	return m.copyText(text, "Fleet snapshot copied - paste it anywhere")
}

// fleetSnapshotMarkdown renders the fleet state as markdown. Everything that
// originated outside cooldeck goes through SanitizeLogText, so a hostile
// commit message cannot smuggle escape sequences into wherever this is pasted.
func fleetSnapshotMarkdown(
	conn app.Connection,
	applications []domain.Application,
	deployments []domain.Deployment,
	recentErrors []string,
	now time.Time,
) string {
	var b strings.Builder
	b.WriteString("# Fleet snapshot\n\n")
	if conn.InstanceName != "" {
		b.WriteString("- Instance: " + domain.SanitizeLogText(conn.InstanceName))
		if conn.Version != "" {
			b.WriteString(" (Coolify " + domain.SanitizeLogText(conn.Version) + ")")
		}
		b.WriteString("\n")
	}
	b.WriteString("- Taken: " + now.UTC().Format(time.RFC3339) + "\n\n")

	b.WriteString("## Applications\n\n")
	b.WriteString("| Application | Status | Branch | Last deployment |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, a := range applications {
		last := "unknown"
		if a.LastDeployment != nil {
			last = string(a.LastDeployment.Status) + " " + domain.HumanizeAge(a.LastDeployment.StartedAt, now)
		}
		b.WriteString("| " + snapshotCell(a.Name) +
			" | " + snapshotCell(a.Status.Raw) +
			" | " + snapshotCell(a.Branch) +
			" | " + snapshotCell(last) + " |\n")
	}

	if len(deployments) > 0 {
		b.WriteString("\n## Recent deployments\n\n")
		shown := deployments
		if len(shown) > snapshotDeploymentLimit {
			shown = shown[:snapshotDeploymentLimit]
			b.WriteString("_Newest " + strconv.Itoa(snapshotDeploymentLimit) + " of " +
				strconv.Itoa(len(deployments)) + "; the rest were truncated._\n\n")
		}
		for _, d := range shown {
			line := "- " + snapshotCell(d.ApplicationName) +
				": " + string(d.Status) +
				" " + domain.HumanizeAge(d.CreatedAt, now)
			// The UUID is what turns a pasted line back into something
			// actionable: it is the handle for get_deployment_logs and for
			// Coolify's own API.
			if d.UUID != "" {
				line += " (" + snapshotCell(d.UUID) + ")"
			}
			if msg := strings.TrimSpace(d.CommitMessage); msg != "" {
				line += " - " + snapshotCell(msg)
			}
			b.WriteString(line + "\n")
		}
	}

	if len(recentErrors) > 0 {
		b.WriteString("\n## Recent errors\n\n")
		for _, e := range recentErrors {
			b.WriteString("- " + domain.SanitizeLogText(e) + "\n")
		}
	}

	// Redaction runs once over the finished document rather than per field, so
	// a field added later cannot be the one that forgot to call it. Every value
	// here crossed a trust boundary - commit messages, application names and
	// Coolify's own error text are all attacker- or accident-controlled.
	out := b.String()
	if safe := logging.Redact(out); safe != out {
		out = safe + "\n_Credential-shaped values were replaced with " + logging.Mask + " before copying._\n"
	}
	return out
}

// snapshotCell sanitises a value and keeps it from breaking the markdown table.
func snapshotCell(s string) string {
	s = domain.SanitizeLogText(strings.TrimSpace(s))
	if s == "" {
		return "-"
	}
	return strings.ReplaceAll(s, "|", "\\|")
}
