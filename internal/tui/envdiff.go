package tui

import (
	"context"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// envDiffResult is the loaded comparison shown in the env drift overlay.
// Only keys and verdicts are carried; values never reach the TUI layer at all
// (the Service hashes them at the API boundary).
type envDiffResult struct {
	NameA string
	NameB string
	Diff  domain.EnvDiff
}

// envDiffLoadedMsg delivers a finished comparison.
type envDiffLoadedMsg struct {
	Seq    uint64
	Result envDiffResult
}

// envDiffFailedMsg delivers a failed comparison.
type envDiffFailedMsg struct {
	Seq uint64
	Err *domain.Error
}

// handleCompareEnv implements the two-press flow on the applications list:
// the first x marks the baseline application, the second x on another row
// loads and compares both environments. Pressing x on the baseline unmarks it.
func (m *Model) handleCompareEnv() tea.Cmd {
	a, ok := m.apps.Selected()
	if !ok {
		return nil
	}
	if !m.capabilities.EnvVars {
		return m.pushToast(components.ToastWarning, "Env compare unavailable", "The token cannot read environment variables.")
	}
	switch m.compareBaseUUID {
	case "":
		m.compareBaseUUID = a.UUID
		m.compareBaseName = a.Name
		return m.pushToast(components.ToastInfo, "Comparing from "+a.Name, "Press x on another application to diff their environments.")
	case a.UUID:
		m.compareBaseUUID = ""
		m.compareBaseName = ""
		return m.pushToast(components.ToastInfo, "Env compare cancelled", "")
	default:
		return m.loadEnvDiff(m.compareBaseUUID, m.compareBaseName, a.UUID, a.Name)
	}
}

// loadEnvDiff fetches both environments in one command and diffs them off the
// event loop. Supersession follows the standard seq + cancel pattern.
func (m *Model) loadEnvDiff(uuidA, nameA, uuidB, nameB string) tea.Cmd {
	if m.cancelEnvDiff != nil {
		m.cancelEnvDiff()
	}
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	m.cancelEnvDiff = cancel
	m.envDiffSeq++
	seq := m.envDiffSeq
	service := m.service

	return func() tea.Msg {
		defer cancel()
		varsA, err := service.EnvVars(ctx, uuidA)
		if err != nil {
			return envDiffFailedMsg{Seq: seq, Err: domain.AsError(err)}
		}
		varsB, err := service.EnvVars(ctx, uuidB)
		if err != nil {
			return envDiffFailedMsg{Seq: seq, Err: domain.AsError(err)}
		}
		return envDiffLoadedMsg{Seq: seq, Result: envDiffResult{
			NameA: nameA,
			NameB: nameB,
			Diff:  domain.DiffEnv(varsA, varsB),
		}}
	}
}

// handleEnvDiffKey drives the open overlay; any dismissal key closes it.
func (m *Model) handleEnvDiffKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "x", "enter":
		m.envDiff = nil
	}
	return m, nil
}

// overlayEnvDiff places the comparison modal over the frame.
func (m *Model) overlayEnvDiff(frame string) string {
	if m.envDiff == nil || m.layout.Width < 28 || m.layout.Height < 10 {
		return frame
	}
	th := m.theme
	r := m.envDiff
	width := min(m.layout.Width-4, 64)
	inner := max(width-4, 20)

	nameA := domain.SanitizeLogText(r.NameA)
	nameB := domain.SanitizeLogText(r.NameB)

	lines := []string{
		th.ModalTitle.Render("Env drift"),
		th.ModalHint.Render(components.Truncate(nameA+" vs "+nameB, inner, th.Sym.Ellipsis)),
		th.ModalHint.Render("keys and value fingerprints only - values never leave the API client"),
		"",
	}
	section := func(title string, keys []string) {
		if len(keys) == 0 {
			return
		}
		lines = append(lines, th.PaletteGroup.Render(title))
		for _, k := range keys {
			lines = append(lines, th.ModalBody.Render("  "+components.Truncate(domain.SanitizeLogText(k), inner-2, th.Sym.Ellipsis)))
		}
		lines = append(lines, "")
	}
	section("Only in "+components.Truncate(nameA, 24, th.Sym.Ellipsis), r.Diff.OnlyInA)
	section("Only in "+components.Truncate(nameB, 24, th.Sym.Ellipsis), r.Diff.OnlyInB)
	section("Different values", r.Diff.Different)
	if r.Diff.InSync() {
		lines = append(lines, th.ModalBody.Render("The environments are in sync."), "")
	}
	lines = append(lines,
		th.ModalHint.Render(strconv.Itoa(r.Diff.Same)+" matching  ·  esc close"))

	maxBody := max(m.layout.Height-6, 6)
	if len(lines) > maxBody {
		lines = lines[:maxBody]
	}
	modal := theme.Fill(th.Modal.Width(width), lipgloss.JoinVertical(lipgloss.Left, lines...))
	dimmed := m.theme.Backdrop.Render(lipgloss.JoinVertical(lipgloss.Left, components.Lines(frame)...))
	x := max((m.layout.Width-lipgloss.Width(modal))/2, 0)
	y := max((m.layout.Height-lipgloss.Height(modal))/6, 1)
	return components.Overlay(dimmed, modal, x, y)
}
