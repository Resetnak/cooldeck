package tui

import "charm.land/bubbles/v2/key"

// KeyMap is the single definition of every keybinding. Nothing in the UI
// compares key strings directly; a binding change here changes the behaviour,
// the footer hints and the help screen at once.
type KeyMap struct {
	// Navigation.
	Up       key.Binding
	Down     key.Binding
	Top      key.Binding
	Bottom   key.Binding
	PageDown key.Binding
	PageUp   key.Binding
	Left     key.Binding
	Right    key.Binding

	// Structure.
	Enter     key.Binding
	Back      key.Binding
	NextPane  key.Binding
	PrevPane  key.Binding
	Quit      key.Binding
	ForceQuit key.Binding

	// Global.
	Filter  key.Binding
	Palette key.Binding
	Help    key.Binding
	Refresh key.Binding
	Theme   key.Binding
	Compact key.Binding

	// Fleet tail.
	Mark key.Binding
	Tail key.Binding

	// Sections.
	SectionApplications key.Binding
	SectionDeployments  key.Binding
	SectionInstances    key.Binding
	SectionDiagnostics  key.Binding

	// Application actions.
	Deploy      key.Binding
	ForceDeploy key.Binding
	Restart     key.Binding
	StartStop   key.Binding
	RuntimeLogs key.Binding
	BuildLogs   key.Binding
	OpenBrowser key.Binding
	OpenRepo    key.Binding
	CopyUUID    key.Binding
	Sort        key.Binding

	// Deployments section.
	DeploymentsActive key.Binding

	// Instances section.
	TestConnection key.Binding
	AddInstance    key.Binding
	EditInstance   key.Binding
	DeleteInstance key.Binding

	// Diagnostics section.
	ExportDiagnostics key.Binding

	// Log view.
	LogFollow key.Binding
	LogPause  key.Binding
	LogWrap   key.Binding
	LogSearch key.Binding
	LogNext   key.Binding
	LogPrev   key.Binding
	LogClear  key.Binding
	LogMore   key.Binding
	LogLess   key.Binding

	// Confirmation.
	Confirm key.Binding
	Cancel  key.Binding

	// Demo-only.
	DemoToggleOffline key.Binding
}

// DefaultKeyMap returns the bindings cooldeck ships with. They follow the
// vi-flavoured conventions of lazygit and k9s, with arrow-key equivalents for
// everyone else.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("↑/k", "up")),
		Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("↓/j", "down")),
		Top:      key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "first")),
		Bottom:   key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "last")),
		PageDown: key.NewBinding(key.WithKeys("ctrl+d", "pgdown"), key.WithHelp("ctrl+d", "page down")),
		PageUp:   key.NewBinding(key.WithKeys("ctrl+u", "pgup"), key.WithHelp("ctrl+u", "page up")),
		Left:     key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("←/h", "left")),
		Right:    key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("→/l", "right")),

		Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		NextPane:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
		PrevPane:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev pane")),
		Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "back/quit")),
		ForceQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),

		Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Palette: key.NewBinding(key.WithKeys(":", "ctrl+k"), key.WithHelp(":", "commands")),
		Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Refresh: key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
		Theme:   key.NewBinding(key.WithKeys("ctrl+t"), key.WithHelp("ctrl+t", "toggle theme")),
		Compact: key.NewBinding(key.WithKeys("ctrl+w"), key.WithHelp("ctrl+w", "compact mode")),

		// Bubble Tea v2 renders the spacebar as "space", never as a literal " ".
		Mark: key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "mark for tail")),
		Tail: key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tail marked")),

		SectionApplications: key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "applications")),
		SectionDeployments:  key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "deployments")),
		SectionInstances:    key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "instances")),
		SectionDiagnostics:  key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "diagnostics")),

		Deploy:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "deploy")),
		ForceDeploy: key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "force deploy")),
		Restart:     key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
		StartStop:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start/stop")),
		RuntimeLogs: key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs")),
		BuildLogs:   key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "build log")),
		OpenBrowser: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "open domain")),
		OpenRepo:    key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open repo")),
		CopyUUID:    key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy UUID")),
		Sort:        key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "sort")),

		DeploymentsActive: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "active filter")),

		TestConnection: key.NewBinding(key.WithKeys("T"), key.WithHelp("T", "test connection")),
		AddInstance:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add instance")),
		EditInstance:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit instance")),
		DeleteInstance: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete instance")),

		ExportDiagnostics: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),

		LogFollow: key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "follow")),
		LogPause:  key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "pause")),
		LogWrap:   key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "wrap")),
		LogSearch: key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		LogNext:   key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next match")),
		LogPrev:   key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "prev match")),
		LogClear:  key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "clear buffer")),
		LogMore:   key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "more lines")),
		LogLess:   key.NewBinding(key.WithKeys("-"), key.WithHelp("-", "fewer lines")),

		Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),

		DemoToggleOffline: key.NewBinding(key.WithKeys("f2"), key.WithHelp("F2", "toggle demo outage")),
	}
}
