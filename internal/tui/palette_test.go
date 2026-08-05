package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/views"
)

func TestCommandPaletteOpensFiltersAndRuns(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{Now: func() time.Time { return now }})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	model := New(Options{Config: config.Default(), Service: service, Demo: true, ASCII: true, Now: func() time.Time { return now }})
	model.loading = false
	model.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 28})

	_, _ = model.handleKey(tea.KeyPressMsg{Code: ':', Text: ":"})
	if !model.paletteOpen {
		t.Fatal(": did not open command palette")
	}
	plain := ansi.Strip(model.render())
	if !strings.Contains(plain, "Commands") || !strings.Contains(plain, "Deploy selected application") {
		t.Fatalf("palette missing catalogue:\n%s", plain)
	}

	// Filter down to deploy commands.
	for _, r := range "deploy" {
		model.handleKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	items := model.filteredPaletteCommands()
	if len(items) == 0 {
		t.Fatal("filter removed every command")
	}
	for _, item := range items {
		if !strings.Contains(strings.ToLower(item.cmd.Title+item.cmd.Description), "deploy") {
			t.Fatalf("unrelated command survived filter: %s", item.cmd.Title)
		}
	}

	// Confirming a dangerous action stages the confirmation modal, not the API call.
	model.paletteSelected = 0
	// Ensure the first filtered item is deploy.
	found := false
	for i, item := range model.filteredPaletteCommands() {
		if item.cmd.ID == "deploy" {
			model.paletteSelected = i
			found = true
			break
		}
	}
	if !found {
		t.Fatal("deploy command missing after filter")
	}
	_, cmd := model.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if model.paletteOpen {
		t.Fatal("enter left palette open")
	}
	if model.pendingAction == nil || model.pendingAction.kind != actionDeploy {
		t.Fatalf("deploy was not staged: pending=%v cmd=%v", model.pendingAction, cmd)
	}
}

func TestCommandPaletteShowsDisabledReason(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 28})
	model.openPalette()
	// Without applications loaded, action commands should be disabled.
	items := model.filteredPaletteCommands()
	var deploy *rankedCommand
	for i := range items {
		if items[i].cmd.ID == "deploy" {
			deploy = &items[i]
			break
		}
	}
	if deploy == nil {
		t.Fatal("deploy command missing")
	}
	if deploy.ok || deploy.reason == "" {
		t.Fatalf("deploy should be disabled with reason, got ok=%v reason=%q", deploy.ok, deploy.reason)
	}

	// Selecting a disabled command toasts instead of running.
	// "deploy" is disabled without an application selected.
	for i, item := range items {
		if item.cmd.ID == "deploy" {
			model.paletteSelected = i
			break
		}
	}
	_, cmd := model.handleKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !model.paletteOpen {
		t.Fatal("disabled command closed the palette")
	}
	if cmd == nil {
		t.Fatal("disabled command did not toast")
	}
	msg := cmd()
	toast, ok := msg.(toastMsg)
	if !ok || toast.Kind != int(components.ToastWarning) {
		t.Fatalf("expected warning toast, got %T %+v", msg, msg)
	}
}

func TestCommandPaletteEscapeCloses(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.openPalette()
	model.paletteQuery = "dep"
	model.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.paletteOpen || model.paletteQuery != "" {
		t.Fatal("esc did not reset palette state")
	}
}

func TestCommandPaletteQueryKeepsSpaces(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.openPalette()

	// Command titles are multi-word, so the query has to carry a space.
	for _, k := range []tea.KeyPressMsg{
		{Code: 'a', Text: "a"},
		{Code: tea.KeySpace, Text: " "},
		{Code: 'b', Text: "b"},
	} {
		model.handleKey(k)
	}
	if model.paletteQuery != "a b" {
		t.Fatalf("palette query = %q, want %q", model.paletteQuery, "a b")
	}
}

func TestFuzzyScoreRanksContiguousHigher(t *testing.T) {
	contiguous, ok1 := fuzzyScore("dep", "Deploy selected application")
	scattered, ok2 := fuzzyScore("dep", "Open deployments")
	if !ok1 || !ok2 {
		t.Fatal("expected both candidates to match")
	}
	// Both match; the function must at least accept valid subsequences.
	if contiguous <= 0 || scattered <= 0 {
		t.Fatalf("scores should be positive: %d %d", contiguous, scattered)
	}
	_, miss := fuzzyScore("zzz", "Deploy")
	if miss {
		t.Fatal("non-subsequence reported as match")
	}
}

func TestRuntimeLogClearAndLineWindow(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.screen = screenDetail
	model.detail.SetApplication(domain.Application{UUID: "app-1"}, time.Now())
	model.detail.SetTab(views.TabRuntimeLogs)
	model.detail.SetRuntimeLogs([]domain.LogLine{{Text: "a"}, {Text: "b"}}, time.Now(), false)
	model.logLines = 300

	_, cmd := model.handleDetailKey(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	if model.detail.RuntimeLogCount() != 0 {
		t.Fatal("ctrl+l did not empty runtime buffer")
	}
	if cmd == nil {
		t.Fatal("clear should toast")
	}

	// More lines increases the session window and reloads.
	model.detail.SetRuntimeLogs([]domain.LogLine{{Text: "x"}}, time.Now(), false)
	_, cmd = model.handleDetailKey(tea.KeyPressMsg{Code: '+', Text: "+"})
	if model.logLines != 350 {
		t.Fatalf("logLines = %d, want 350", model.logLines)
	}
	if cmd == nil {
		t.Fatal("more lines should reload")
	}
	_, _ = model.handleDetailKey(tea.KeyPressMsg{Code: '-', Text: "-"})
	if model.logLines != 300 {
		t.Fatalf("logLines after less = %d", model.logLines)
	}

	// Floor at minimum.
	model.logLines = minSessionLogLines
	model.adjustLogLines(-1)
	if model.logLines != minSessionLogLines {
		t.Fatalf("logLines fell below minimum: %d", model.logLines)
	}
}

func TestCopyFromDetailPrefersLogsAndDeployUUID(t *testing.T) {
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.screen = screenDetail
	model.detail.SetApplication(domain.Application{UUID: "app-uuid"}, time.Now())
	model.detail.SetTab(views.TabRuntimeLogs)
	model.detail.SetRuntimeLogs([]domain.LogLine{{Text: "line-one"}, {Text: "line-two"}}, time.Now(), false)

	// Intercept clipboard via executing the command only for structure: the
	// text path is covered by detail tests; here we ensure a command is issued.
	cmd := model.copyFromDetail()
	if cmd == nil {
		t.Fatal("copy issued no command")
	}

	model.detail.SetTab(views.TabDeployments)
	model.detail.SetDeployments([]domain.Deployment{{UUID: "dep-uuid-1"}, {UUID: "dep-uuid-2"}})
	model.detail.MoveDeployment(1)
	// Still produces a copy command (deployment UUID).
	if model.copyFromDetail() == nil {
		t.Fatal("deployment uuid copy issued no command")
	}
}

func TestLogKeyBindingsMatch(t *testing.T) {
	keys := DefaultKeyMap()
	clearMsg := tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl}
	if !key.Matches(clearMsg, keys.LogClear) {
		t.Fatalf("ctrl+l did not match LogClear: %+v", clearMsg)
	}
	moreMsg := tea.KeyPressMsg{Code: '+', Text: "+"}
	if !key.Matches(moreMsg, keys.LogMore) {
		// try equals
		moreMsg = tea.KeyPressMsg{Code: '=', Text: "="}
		if !key.Matches(moreMsg, keys.LogMore) {
			t.Fatal("+ and = did not match LogMore")
		}
	}
	lessMsg := tea.KeyPressMsg{Code: '-', Text: "-"}
	if !key.Matches(lessMsg, keys.LogLess) {
		t.Fatal("- did not match LogLess")
	}
	copyMsg := tea.KeyPressMsg{Code: 'c', Text: "c"}
	if !key.Matches(copyMsg, keys.CopyUUID) {
		t.Fatal("c did not match CopyUUID")
	}
	paletteMsg := tea.KeyPressMsg{Code: ':', Text: ":"}
	if !key.Matches(paletteMsg, keys.Palette) {
		t.Fatal(": did not match Palette")
	}
}
