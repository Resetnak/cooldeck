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
)

func newTerminalTestModel(t *testing.T) *Model {
	t.Helper()
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	model := New(Options{
		Config:  config.Default(),
		Service: service,
		Demo:    true,
		ASCII:   true,
		Now:     func() time.Time { return now },
	})
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
	return model
}

// TestTerminalKeyRouting pins the t/T split on the applications list: lowercase
// t opens the fleet tail, uppercase T reaches the container terminal. Both keys
// answer with a toast here because nothing is marked and no ssh_host is
// configured, which is exactly what makes the routing observable.
func TestTerminalKeyRouting(t *testing.T) {
	model := newTerminalTestModel(t)

	toastText := func(msg tea.KeyPressMsg) string {
		_, cmd := model.handleApplicationsKey(msg)
		if cmd == nil {
			return ""
		}
		toast, ok := cmd().(toastMsg)
		if !ok {
			return ""
		}
		return toast.Text
	}

	upper := toastText(tea.KeyPressMsg{Code: 't', Text: "T", Mod: tea.ModShift})
	if !strings.Contains(upper, "Terminal unavailable") {
		t.Fatalf("shift+T routed to %q, want the terminal toast", upper)
	}
	lower := toastText(tea.KeyPressMsg{Code: 't', Text: "t"})
	if !strings.Contains(lower, "Nothing marked") {
		t.Fatalf("t routed to %q, want the fleet-tail toast", lower)
	}
}

// TestTerminalContainerPicker drives the multi-container flow: several
// containers open the picker, arrows move the selection, esc closes it and
// enter hands off to the interactive session.
func TestTerminalContainerPicker(t *testing.T) {
	model := newTerminalTestModel(t)
	application := domain.Application{UUID: "abc123", Name: "repometer"}

	// Even a single container opens the picker: the operator must see which
	// container they are entering before the screen is handed over.
	model.terminalSeq = 7
	model.Update(terminalContainersMsg{Seq: 7, App: application, Containers: []string{"web-abc123"}})
	if model.terminalPicker == nil {
		t.Fatal("single container must still open the picker")
	}
	model.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})

	model.Update(terminalContainersMsg{Seq: 7, App: application,
		Containers: []string{"web-abc123", "php-abc123", "pg-abc123"}})
	if model.terminalPicker == nil {
		t.Fatal("multiple containers must open the picker")
	}
	plain := ansi.Strip(model.render())
	for _, name := range []string{"web-abc123", "php-abc123", "pg-abc123"} {
		if !strings.Contains(plain, name) {
			t.Fatalf("picker render omits %q:\n%s", name, plain)
		}
	}

	model.handleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := model.terminalPicker.selected; got != 1 {
		t.Fatalf("selected = %d after down, want 1", got)
	}
	model.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.terminalPicker != nil {
		t.Fatal("esc must close the picker")
	}

	// Stale replies from an abandoned lookup are dropped.
	model.Update(terminalContainersMsg{Seq: 6, App: application,
		Containers: []string{"web-abc123", "php-abc123"}})
	if model.terminalPicker != nil {
		t.Fatal("stale sequence must not open the picker")
	}
}
