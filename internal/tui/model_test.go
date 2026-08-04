package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

func TestLoadedDashboardRendersAcrossBreakpoints(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	for _, size := range []struct{ width, height int }{{60, 18}, {100, 28}, {160, 40}} {
		model := New(Options{
			Config:       config.Default(),
			Service:      service,
			InstanceName: "Demo",
			Demo:         true,
			ASCII:        true,
			Now:          func() time.Time { return now },
		})
		model.Update(tea.WindowSizeMsg{Width: size.width, Height: size.height})
		model.loading = false
		model.connection = components.ConnectionOnline
		model.lastSuccess = now
		model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)

		plain := ansi.Strip(model.render())
		selected, ok := model.apps.Selected()
		if !ok || !strings.Contains(plain, selected.Name) {
			t.Fatalf("%dx%d render omitted application name:\n%s", size.width, size.height, plain)
		}
		for _, line := range strings.Split(plain, "\n") {
			if ansi.StringWidth(line) > size.width {
				t.Fatalf(
					"%dx%d render overflowed to %d cells: %q",
					size.width,
					size.height,
					ansi.StringWidth(line),
					line,
				)
			}
		}
	}
}

func TestDashboardErrorStateIsActionable(t *testing.T) {
	service := demo.New(demo.Options{})
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true})
	model.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	model.loading = false
	model.lastError = domain.NewError(domain.ErrorNetwork, nil)

	plain := ansi.Strip(model.render())
	if !strings.Contains(plain, "Cannot reach Coolify") || !strings.Contains(plain, "retry") {
		t.Fatalf("error state is not actionable:\n%s", plain)
	}
}
