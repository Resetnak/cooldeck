package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/theme"
	"github.com/resetnak/cooldeck/internal/version"
)

// ConnectionState is what the header's status dot reports.
type ConnectionState int

// Connection states.
const (
	ConnectionConnecting ConnectionState = iota
	ConnectionOnline
	ConnectionOffline
	ConnectionUnauthorized
)

// HeaderData is everything the header line shows. It is assembled by the root
// model so the header itself stays a pure renderer.
type HeaderData struct {
	InstanceName string
	Connection   ConnectionState
	// Demo marks the run as using generated data, which must never be
	// mistaken for a real instance.
	Demo bool

	ResourceCount int
	ResourceNoun  string

	LastRefresh time.Time
	Now         time.Time
	// Refreshing draws a small spinner instead of clearing the screen, so a
	// background refresh never blanks the data the user is reading.
	Refreshing bool
	Spinner    string

	// Filter is the active query, shown as a chip so it is never a mystery
	// why rows are missing.
	Filter string
}

// Header renders the top bar plus its hairline rule.
func Header(th *theme.Theme, l theme.Layout, d HeaderData) string {
	if !l.ShowHeader {
		return ""
	}

	left := th.HeaderLogo.Render(strings.ToUpper(version.AppName))
	if !l.IsCompact() {
		left += "  " + th.HeaderInstance.Render(d.InstanceName)
	} else {
		left += " " + th.HeaderInstance.Render(d.InstanceName)
	}
	left += "  " + connectionChip(th, d.Connection)
	if d.Demo {
		left += "  " + th.Badge(theme.BadgeInfo, "DEMO")
	}

	right := headerMeta(th, l, d)

	inner := l.Width - 2*theme.SpaceXS
	gap := inner - Width(left) - Width(right)
	if gap < 1 {
		// Drop the metadata rather than wrapping the bar onto a second line.
		right = ""
		gap = max(inner-Width(left), 0)
	}

	bar := th.HeaderBar.Render(left + strings.Repeat(" ", gap) + right)
	rule := th.HeaderRule.Render(strings.Repeat("─", l.Width))
	return lipgloss.JoinVertical(lipgloss.Left, Pad(bar, l.Width), rule)
}

func connectionChip(th *theme.Theme, s ConnectionState) string {
	switch s {
	case ConnectionOnline:
		return th.Badge(theme.BadgeSuccess, th.Sym.StatusRunning+" connected")
	case ConnectionOffline:
		return th.Badge(theme.BadgeError, th.Sym.StatusFailed+" offline")
	case ConnectionUnauthorized:
		return th.Badge(theme.BadgeError, th.Sym.Lock+" unauthorized")
	default:
		return th.Badge(theme.BadgeWarning, th.Sym.StatusDeploying+" connecting")
	}
}

func headerMeta(th *theme.Theme, l theme.Layout, d HeaderData) string {
	var parts []string

	if d.Filter != "" {
		label := d.Filter
		if l.IsCompact() {
			label = Truncate(label, 12, th.Sym.Ellipsis)
		}
		parts = append(parts, th.Accent.Render(th.Sym.Filter+" "+label))
	}

	if d.ResourceCount > 0 || d.ResourceNoun != "" {
		noun := d.ResourceNoun
		if noun == "" {
			noun = "items"
		}
		parts = append(parts, th.HeaderMeta.Render(fmt.Sprintf("%d %s", d.ResourceCount, noun)))
	}

	switch {
	case d.Refreshing:
		parts = append(parts, th.Spinner.Render(d.Spinner)+th.HeaderMeta.Render(" refreshing"))
	case !d.LastRefresh.IsZero():
		age := domain.HumanizeAge(d.LastRefresh, d.Now)
		if l.IsCompact() {
			parts = append(parts, th.HeaderMeta.Render(age))
		} else {
			parts = append(parts, th.HeaderMeta.Render("refreshed "+age))
		}
	}

	sep := th.HeaderRule.Render("  " + th.Sym.Separator + "  ")
	if l.IsCompact() {
		sep = " "
	}
	return strings.Join(parts, sep)
}

// StaleBanner renders the offline notice shown above cached data, so the user
// is never misled into thinking they are looking at the live state.
func StaleBanner(th *theme.Theme, width string, since time.Duration) string {
	_ = width
	return th.Attention.Render(fmt.Sprintf(
		"%s OFFLINE %s showing data from %s ago",
		th.Sym.Warning, th.Sym.Separator, domain.HumanizeDuration(since)))
}
