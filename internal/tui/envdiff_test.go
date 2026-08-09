package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

// TestCompareEnvTwoPressFlow drives the whole feature: mark, pick, load, and
// render the drift overlay, then dismiss it.
func TestCompareEnvTwoPressFlow(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	model := New(Options{
		Config:       config.Default(),
		Service:      service,
		InstanceName: "Demo",
		Demo:         true,
		ASCII:        true,
		Now:          func() time.Time { return now },
	})
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 32})
	model.loading = false
	model.connection = components.ConnectionOnline
	model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)

	// First x marks the baseline.
	if cmd := model.handleCompareEnv(); cmd == nil {
		t.Fatal("first press produced no feedback toast")
	}
	if model.compareBaseUUID == "" {
		t.Fatal("first press did not mark a baseline")
	}

	// Second x on another application loads the comparison.
	model.apps.Move(1)
	cmd := model.handleCompareEnv()
	if cmd == nil {
		t.Fatal("second press produced no load command")
	}
	msg, ok := cmd().(envDiffLoadedMsg)
	if !ok {
		t.Fatalf("second press produced %T, want envDiffLoadedMsg", cmd())
	}
	model.Update(msg)
	if model.envDiff == nil {
		t.Fatal("loaded comparison did not open the overlay")
	}

	plain := ansi.Strip(model.render())
	if !strings.Contains(plain, "Env drift") {
		t.Fatalf("overlay missing from render:\n%s", plain)
	}
	// The demo fleet drifts SENTRY_DSN between neighbouring applications.
	if !strings.Contains(plain, "SENTRY_DSN") {
		t.Fatalf("expected drifted key in overlay:\n%s", plain)
	}

	model.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.envDiff != nil {
		t.Fatal("esc did not close the overlay")
	}
}
