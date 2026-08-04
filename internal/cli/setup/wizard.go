// Package setup implements an interactive TUI wizard for first-time CoolDeck
// configuration. It walks the user through Coolify instance details, token
// storage, and display preferences, then writes a valid config.toml.
package setup

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/credentials"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// ---------------------------------------------------------------------------
// Step enum
// ---------------------------------------------------------------------------

type wizardStep int

const (
	stepWelcome wizardStep = iota
	stepInstanceID
	stepInstanceName
	stepURL
	stepTokenSource
	stepToken
	stepTheme
	stepConfirm
	stepSaving
	stepDone
)

const totalUserSteps = 7 // steps the user actively fills in (welcome & done excluded)

// contentMaxWidth caps the wizard column so lines stay readable on ultra-wide terminals.
const contentMaxWidth = 72

// ---------------------------------------------------------------------------
// Styles (derived from the shared theme package)
// ---------------------------------------------------------------------------

type wizardStyles struct {
	title       lipgloss.Style
	subtitle    lipgloss.Style
	accent      lipgloss.Style
	dim         lipgloss.Style
	success     lipgloss.Style
	errStyle    lipgloss.Style
	selected    lipgloss.Style
	unselected  lipgloss.Style
	key         lipgloss.Style
	stepCounter lipgloss.Style
	progressOn  lipgloss.Style
	progressOff lipgloss.Style
	box         lipgloss.Style
	logo        lipgloss.Style
	logoAccent  lipgloss.Style
	rule        lipgloss.Style
	app         lipgloss.Style
	warn        lipgloss.Style
}

// logoLines is a compact block wordmark for the welcome screen.
// Raw string keeps backslashes intact (no broken art).
var logoLines = []string{
	`   ____            _ ____            _   `,
	`  / ___|___   ___ | |  _ \  ___  ___| | __`,
	` | |   / _ \ / _ \| | | | |/ _ \/ __| |/ /`,
	` | |__| (_) | (_) | | |_| |  __/ (__|   < `,
	`  \____\___/ \___/|_|____/ \___|\___|_|\_\`,
}

// themeOpts maps a config theme name onto the TUI theme package options used
// for live preview. "auto" previews as dark so the wizard always has a concrete
// palette while browsing.
func themeOpts(t config.Theme) theme.Options {
	mode := theme.ModeDark
	paletteName := ""
	switch t {
	case config.ThemeLight:
		mode = theme.ModeLight
	case config.ThemeDark:
		mode = theme.ModeDark
	case config.ThemeDracula, config.ThemeCatppuccin, config.ThemeNord,
		config.ThemeGruvbox, config.ThemeTokyoNight:
		paletteName = string(t)
	default:
		mode = theme.ModeDark
	}
	return theme.Options{Mode: mode, PaletteName: paletteName}
}

func newStyles(t config.Theme) wizardStyles {
	th := theme.New(themeOpts(t))
	p := th.Palette
	// Every text style carries the app background. Foreground-only styles emit
	// a full SGR reset that drops the parent background mid-line and leaves
	// the terminal default (often white) behind the padded remainder.
	base := lipgloss.NewStyle().Background(p.Background)
	return wizardStyles{
		title:       base.Bold(true).Foreground(p.Primary),
		subtitle:    base.Foreground(p.Primary),
		accent:      base.Bold(true).Foreground(p.Secondary),
		dim:         base.Foreground(p.TextMuted),
		success:     base.Bold(true).Foreground(p.Success),
		errStyle:    base.Bold(true).Foreground(p.Error),
		selected:    base.Bold(true).Foreground(p.Primary),
		unselected:  base.Foreground(p.TextMuted),
		key:         base.Bold(true).Foreground(p.Warning),
		stepCounter: base.Foreground(p.TextSubtle),
		progressOn:  base.Foreground(p.Primary),
		progressOff: base.Foreground(p.Border),
		box: lipgloss.NewStyle().
			Border(th.Border).
			BorderForeground(p.Border).
			Background(p.Surface).
			Padding(1, 2),
		logo:       base.Bold(true).Foreground(p.Primary),
		logoAccent: base.Bold(true).Foreground(p.Secondary),
		rule:       base.Foreground(p.Border),
		app:        base.Foreground(p.Text),
		warn:       base.Bold(true).Foreground(p.Warning),
	}
}

func inputStyles(t config.Theme) textinput.Styles {
	th := theme.New(themeOpts(t))
	p := th.Palette
	base := lipgloss.NewStyle().Background(p.Background)
	return textinput.Styles{
		Focused: textinput.StyleState{
			Text:        base.Foreground(p.Text),
			Placeholder: base.Foreground(p.TextSubtle),
			Prompt:      base.Foreground(p.Primary).Bold(true),
		},
		Blurred: textinput.StyleState{
			Text:        base.Foreground(p.TextMuted),
			Placeholder: base.Foreground(p.TextSubtle),
			Prompt:      base.Foreground(p.TextSubtle),
		},
		Cursor: textinput.CursorStyle{
			Color: p.Primary,
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
}

// ---------------------------------------------------------------------------
// Token source / theme option helpers
// ---------------------------------------------------------------------------

var tokenSourceOptions = []struct {
	source config.TokenSource
	label  string
	desc   string
}{
	{config.TokenSourceKeyring, "OS Keyring", "Secure, recommended. Stored in macOS Keychain / GNOME Keyring / Windows Credential Manager."},
	{config.TokenSourceEnv, "Environment Variable", "Read token from a named env var at startup."},
	{config.TokenSourcePlaintext, "Plaintext in Config", "Stored directly in config.toml. Not recommended."},
}

var themeOptions = []struct {
	theme config.Theme
	label string
}{
	{config.ThemeAuto, "Auto"},
	{config.ThemeDark, "Dark"},
	{config.ThemeLight, "Light"},
	{config.ThemeDracula, "Dracula"},
	{config.ThemeCatppuccin, "Catppuccin Mocha"},
	{config.ThemeNord, "Nord"},
	{config.ThemeGruvbox, "Gruvbox Dark"},
	{config.ThemeTokyoNight, "Tokyo Night"},
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type wizard struct {
	step          wizardStep
	width, height int

	// text inputs
	instanceIDInput   textinput.Model
	instanceNameInput textinput.Model
	urlInput          textinput.Model
	tokenInput        textinput.Model // password (keyring / plaintext)
	tokenEnvInput     textinput.Model // env var name

	// selections
	tokenSourceIdx int
	themeIdx       int

	// collected values
	instanceID   string
	instanceName string
	coolifyURL   string
	tokenSource  config.TokenSource
	token        string
	tokenEnvVar  string
	theme        config.Theme

	// result / safety
	configPath   string
	configExists bool
	plaintextAck bool
	err          error
	validErr     string

	styles wizardStyles
}

// instanceIDRe matches valid instance IDs (same rule as config.validate).
var instanceIDRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.\-]{0,63}$`)

// New returns a fresh wizard model.
func New() *wizard {
	s := newStyles(config.ThemeDark)
	inStyles := inputStyles(config.ThemeDark)

	idInput := textinput.New()
	idInput.Placeholder = "production"
	idInput.CharLimit = 64
	idInput.Prompt = "▸ "
	idInput.SetStyles(inStyles)
	idInput.SetWidth(48)

	nameInput := textinput.New()
	nameInput.Placeholder = "Production"
	nameInput.CharLimit = 128
	nameInput.Prompt = "▸ "
	nameInput.SetStyles(inStyles)
	nameInput.SetWidth(48)

	urlIn := textinput.New()
	urlIn.Placeholder = "https://coolify.example.com"
	urlIn.CharLimit = 256
	urlIn.Prompt = "▸ "
	urlIn.SetStyles(inStyles)
	urlIn.SetWidth(48)

	tokIn := textinput.New()
	tokIn.Placeholder = "paste your Coolify API token"
	tokIn.EchoMode = textinput.EchoPassword
	tokIn.EchoCharacter = '•'
	tokIn.CharLimit = 512
	tokIn.Prompt = "▸ "
	tokIn.SetStyles(inStyles)
	tokIn.SetWidth(48)

	tokEnvIn := textinput.New()
	tokEnvIn.Placeholder = "COOLIFY_API_TOKEN"
	tokEnvIn.CharLimit = 128
	tokEnvIn.Prompt = "▸ "
	tokEnvIn.SetStyles(inStyles)
	tokEnvIn.SetWidth(48)

	return &wizard{
		step:              stepWelcome,
		instanceIDInput:   idInput,
		instanceNameInput: nameInput,
		urlInput:          urlIn,
		tokenInput:        tokIn,
		tokenEnvInput:     tokEnvIn,
		styles:            s,
	}
}

// applyThemePreview rebuilds wizard and input styles from the highlighted theme
// option so navigation through the list is a live colour preview.
func (w *wizard) applyThemePreview() {
	t := config.ThemeDark
	if w.themeIdx >= 0 && w.themeIdx < len(themeOptions) {
		t = themeOptions[w.themeIdx].theme
	}
	w.styles = newStyles(t)
	in := inputStyles(t)
	w.instanceIDInput.SetStyles(in)
	w.instanceNameInput.SetStyles(in)
	w.urlInput.SetStyles(in)
	w.tokenInput.SetStyles(in)
	w.tokenEnvInput.SetStyles(in)
}

// resetWizardChrome restores the default dark chrome used outside the theme step.
func (w *wizard) resetWizardChrome() {
	w.styles = newStyles(config.ThemeDark)
	in := inputStyles(config.ThemeDark)
	w.instanceIDInput.SetStyles(in)
	w.instanceNameInput.SetStyles(in)
	w.urlInput.SetStyles(in)
	w.tokenInput.SetStyles(in)
	w.tokenEnvInput.SetStyles(in)
}

// Run launches the setup wizard and blocks until it exits.
func Run() error {
	p := tea.NewProgram(New())
	_, err := p.Run()
	return err
}

// ---------------------------------------------------------------------------
// Config build + persist (testable, no UI)
// ---------------------------------------------------------------------------

// buildConfig assembles a Config from wizard fields. It does not touch the
// filesystem or keyring.
func (w *wizard) buildConfig() config.Config {
	cfg := config.Default()
	cfg.DefaultInstance = w.instanceID
	cfg.Theme = w.theme

	inst := config.Instance{
		ID:          w.instanceID,
		Name:        w.instanceName,
		URL:         w.coolifyURL,
		TokenSource: w.tokenSource,
	}

	switch w.tokenSource {
	case config.TokenSourceKeyring:
		inst.TokenKey = w.instanceID
	case config.TokenSourceEnv:
		inst.TokenEnv = w.tokenEnvVar
	case config.TokenSourcePlaintext:
		inst.Token = w.token
	}

	cfg.Instances = map[string]config.Instance{w.instanceID: inst}
	return cfg
}

// persistConfig stores the token (keyring first) and then writes config.toml.
// Order matters: a failed keyring write must not leave a config pointing at a
// missing secret.
func persistConfig(cfg config.Config, token, path string) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	inst, err := cfg.Instance(cfg.DefaultInstance)
	if err != nil {
		return err
	}

	if inst.TokenSource == config.TokenSourceKeyring {
		if strings.TrimSpace(token) == "" {
			return fmt.Errorf("API token is required for keyring storage")
		}
		if err := credentials.Store(credentials.KeyFor(inst), credentials.NewToken(token)); err != nil {
			return err
		}
	}

	if err := cfg.Save(path); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

// savedMsg is delivered after an async persist finishes.
type savedMsg struct {
	path string
	err  error
}

func saveCmd(cfg config.Config, token, path string) tea.Cmd {
	return func() tea.Msg {
		err := persistConfig(cfg, token, path)
		return savedMsg{path: path, err: err}
	}
}

// ---------------------------------------------------------------------------
// tea.Model
// ---------------------------------------------------------------------------

func (w *wizard) Init() tea.Cmd {
	return tea.ClearScreen
}

func (w *wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
		w.resizeInputs()
		return w, nil

	case savedMsg:
		w.configPath = msg.path
		w.err = msg.err
		if msg.err == nil {
			w.clearSecrets()
		}
		w.step = stepDone
		return w, nil

	case tea.KeyPressMsg:
		// Esc / Ctrl+C always abandon the wizard immediately. Letter keys
		// (including "q") stay free for text fields.
		if msg.Code == tea.KeyEscape || (msg.Mod == tea.ModCtrl && msg.Code == 'c') {
			w.clearSecrets()
			return w, tea.Quit
		}

		return w.handleKey(msg)
	}

	// Forward to active text input.
	return w.updateActiveInput(msg)
}

func (w *wizard) View() tea.View {
	var body string
	switch w.step {
	case stepWelcome:
		body = w.viewWelcome()
	case stepInstanceID:
		body = w.viewTextStep(1, "Instance ID",
			"A short identifier for this Coolify instance (e.g. production, staging).\nUsed as the config key and keyring entry name.",
			w.instanceIDInput.View())
	case stepInstanceName:
		body = w.viewTextStep(2, "Display Name",
			"A human-friendly label shown in the dashboard header.",
			w.instanceNameInput.View())
	case stepURL:
		body = w.viewTextStep(3, "Coolify URL",
			"The base URL of your Coolify dashboard.\nCoolDeck will auto-detect and strip /api/v1 suffixes.",
			w.urlInput.View())
	case stepTokenSource:
		body = w.viewSelectionStep(4, "Token Storage",
			"How should CoolDeck store your Coolify API token?")
	case stepToken:
		body = w.viewTokenStep()
	case stepTheme:
		body = w.viewThemeStep()
	case stepConfirm:
		body = w.viewConfirm()
	case stepSaving:
		body = w.viewSaving()
	case stepDone:
		body = w.viewDone()
	}

	frame := w.frame(body)
	v := tea.NewView(frame)
	v.AltScreen = true
	v.WindowTitle = "cooldeck setup"
	return v
}

// ---------------------------------------------------------------------------
// Key handling per step
// ---------------------------------------------------------------------------

func (w *wizard) handleKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch w.step {
	case stepWelcome:
		if key.Code == tea.KeyEnter || key.Text == " " {
			w.step = stepInstanceID
			w.validErr = ""
			return w, w.instanceIDInput.Focus()
		}

	case stepInstanceID:
		if key.Text == "b" && w.instanceIDInput.Value() == "" {
			w.step = stepWelcome
			w.instanceIDInput.Blur()
			w.validErr = ""
			return w, nil
		}
		if key.Code == tea.KeyEnter {
			v := strings.TrimSpace(w.instanceIDInput.Value())
			if v == "" {
				v = "production"
				w.instanceIDInput.SetValue(v)
			}
			if !instanceIDRe.MatchString(v) {
				w.validErr = "Must start with a letter or digit and contain only [a-zA-Z0-9_.-]"
				return w, nil
			}
			w.instanceID = v
			w.step = stepInstanceName
			w.instanceIDInput.Blur()
			w.validErr = ""
			return w, w.instanceNameInput.Focus()
		}
		return w.updateActiveInput(key)

	case stepInstanceName:
		if key.Text == "b" && w.instanceNameInput.Value() == "" {
			w.step = stepInstanceID
			w.instanceNameInput.Blur()
			w.validErr = ""
			return w, w.instanceIDInput.Focus()
		}
		if key.Code == tea.KeyEnter {
			v := strings.TrimSpace(w.instanceNameInput.Value())
			if v == "" {
				v = titleCase(w.instanceID)
			}
			w.instanceName = v
			w.step = stepURL
			w.instanceNameInput.Blur()
			w.validErr = ""
			return w, w.urlInput.Focus()
		}
		return w.updateActiveInput(key)

	case stepURL:
		if key.Text == "b" && w.urlInput.Value() == "" {
			w.step = stepInstanceName
			w.urlInput.Blur()
			w.validErr = ""
			return w, w.instanceNameInput.Focus()
		}
		if key.Code == tea.KeyEnter {
			v := strings.TrimSpace(w.urlInput.Value())
			if v == "" {
				w.validErr = "URL is required"
				return w, nil
			}
			normalized, err := config.NormalizeBaseURL(v)
			if err != nil {
				w.validErr = err.Error()
				return w, nil
			}
			w.coolifyURL = normalized
			w.step = stepTokenSource
			w.urlInput.Blur()
			w.validErr = ""
			return w, nil
		}
		return w.updateActiveInput(key)

	case stepTokenSource:
		switch {
		case key.Text == "b":
			w.step = stepURL
			w.validErr = ""
			return w, w.urlInput.Focus()
		case key.Code == tea.KeyUp || key.Text == "k":
			if w.tokenSourceIdx > 0 {
				w.tokenSourceIdx--
			}
		case key.Code == tea.KeyDown || key.Text == "j":
			if w.tokenSourceIdx < len(tokenSourceOptions)-1 {
				w.tokenSourceIdx++
			}
		case key.Code == tea.KeyEnter:
			w.tokenSource = tokenSourceOptions[w.tokenSourceIdx].source
			w.step = stepToken
			w.validErr = ""
			w.plaintextAck = false
			switch w.tokenSource {
			case config.TokenSourceKeyring, config.TokenSourcePlaintext:
				return w, w.tokenInput.Focus()
			case config.TokenSourceEnv:
				return w, w.tokenEnvInput.Focus()
			}
		}
		return w, nil

	case stepToken:
		activeInput := w.activeTokenInput()
		if key.Text == "b" && activeInput.Value() == "" {
			w.step = stepTokenSource
			w.tokenInput.Blur()
			w.tokenEnvInput.Blur()
			w.validErr = ""
			return w, nil
		}
		if key.Code == tea.KeyEnter {
			switch w.tokenSource {
			case config.TokenSourceKeyring, config.TokenSourcePlaintext:
				v := strings.TrimSpace(w.tokenInput.Value())
				if v == "" {
					w.validErr = "API token is required"
					return w, nil
				}
				w.token = v
			case config.TokenSourceEnv:
				v := strings.TrimSpace(w.tokenEnvInput.Value())
				if v == "" {
					v = "COOLIFY_API_TOKEN"
					w.tokenEnvInput.SetValue(v)
				}
				if !envVarNameOK(v) {
					w.validErr = "Env var name must match [A-Za-z_][A-Za-z0-9_]*"
					return w, nil
				}
				w.tokenEnvVar = v
			}
			w.step = stepTheme
			w.tokenInput.Blur()
			w.tokenEnvInput.Blur()
			w.validErr = ""
			w.applyThemePreview()
			return w, nil
		}
		return w.updateActiveInput(key)

	case stepTheme:
		switch {
		case key.Text == "b":
			w.step = stepToken
			w.validErr = ""
			w.resetWizardChrome()
			switch w.tokenSource {
			case config.TokenSourceKeyring, config.TokenSourcePlaintext:
				return w, w.tokenInput.Focus()
			case config.TokenSourceEnv:
				return w, w.tokenEnvInput.Focus()
			}
			return w, nil
		case key.Code == tea.KeyUp || key.Text == "k":
			if w.themeIdx > 0 {
				w.themeIdx--
				w.applyThemePreview()
			}
		case key.Code == tea.KeyDown || key.Text == "j":
			if w.themeIdx < len(themeOptions)-1 {
				w.themeIdx++
				w.applyThemePreview()
			}
		case key.Code == tea.KeyEnter:
			w.theme = themeOptions[w.themeIdx].theme
			w.prepareConfirm()
			w.step = stepConfirm
			w.validErr = ""
			// Keep the selected theme applied through confirm/done.
		}
		return w, nil

	case stepConfirm:
		switch {
		case key.Text == "b":
			w.step = stepTheme
			w.validErr = ""
			w.plaintextAck = false
			w.applyThemePreview()
			return w, nil
		case key.Code == tea.KeyEnter || key.Text == "y":
			// Plaintext requires an explicit second acknowledgement.
			if w.tokenSource == config.TokenSourcePlaintext && !w.plaintextAck {
				w.plaintextAck = true
				return w, nil
			}
			return w, w.beginSave()
		}
		return w, nil

	case stepSaving:
		return w, nil

	case stepDone:
		switch {
		case w.err != nil && (key.Text == "r" || key.Text == "R"):
			return w, w.beginSave()
		case w.err != nil && key.Text == "b":
			w.step = stepConfirm
			w.err = nil
			return w, nil
		case key.Code == tea.KeyEnter:
			return w, tea.Quit
		}
		return w, nil
	}

	return w, nil
}

func (w *wizard) prepareConfirm() {
	path, err := config.DefaultPath()
	if err == nil {
		w.configPath = path
		if _, statErr := os.Stat(path); statErr == nil {
			w.configExists = true
		} else {
			w.configExists = false
		}
	}
	w.plaintextAck = false
}

func (w *wizard) beginSave() tea.Cmd {
	if w.configPath == "" {
		path, err := config.DefaultPath()
		if err != nil {
			w.err = err
			w.step = stepDone
			return nil
		}
		w.configPath = path
	}
	cfg := w.buildConfig()
	if err := cfg.Validate(); err != nil {
		w.err = err
		w.step = stepDone
		return nil
	}
	w.step = stepSaving
	w.err = nil
	// Copy token for the async cmd; the model still holds it until success
	// so a failed save can be retried without re-entry.
	return saveCmd(cfg, w.token, w.configPath)
}

func (w *wizard) clearSecrets() {
	w.token = ""
	w.tokenInput.SetValue("")
}

func (w *wizard) activeTokenInput() *textinput.Model {
	switch w.tokenSource {
	case config.TokenSourceEnv:
		return &w.tokenEnvInput
	default:
		return &w.tokenInput
	}
}

func (w *wizard) updateActiveInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch w.step {
	case stepInstanceID:
		w.instanceIDInput, cmd = w.instanceIDInput.Update(msg)
	case stepInstanceName:
		w.instanceNameInput, cmd = w.instanceNameInput.Update(msg)
	case stepURL:
		w.urlInput, cmd = w.urlInput.Update(msg)
	case stepToken:
		switch w.tokenSource {
		case config.TokenSourceKeyring, config.TokenSourcePlaintext:
			w.tokenInput, cmd = w.tokenInput.Update(msg)
		case config.TokenSourceEnv:
			w.tokenEnvInput, cmd = w.tokenEnvInput.Update(msg)
		}
	}
	return w, cmd
}

func (w *wizard) resizeInputs() {
	width := min(w.contentWidth()-4, 56)
	if width < 20 {
		width = 20
	}
	w.instanceIDInput.SetWidth(width)
	w.instanceNameInput.SetWidth(width)
	w.urlInput.SetWidth(width)
	w.tokenInput.SetWidth(width)
	w.tokenEnvInput.SetWidth(width)
}

// ---------------------------------------------------------------------------
// Views
// ---------------------------------------------------------------------------

func (w *wizard) viewWelcome() string {
	s := w.styles
	var b strings.Builder

	b.WriteString(w.renderLogoArt())
	b.WriteString("\n")
	b.WriteString(s.dim.Render("  ────────────────────────────────────────"))
	b.WriteString("\n")
	b.WriteString(s.accent.Render("  terminal dashboard for Coolify"))
	b.WriteString("\n")
	b.WriteString(s.dim.Render("  ────────────────────────────────────────"))
	b.WriteString("\n\n")
	b.WriteString(s.title.Render("  ✦  Setup Wizard"))
	b.WriteString("\n\n")
	b.WriteString(s.dim.Render("  Welcome! This wizard creates your first configuration file."))
	b.WriteString("\n\n")
	b.WriteString(s.dim.Render("  You'll configure:"))
	b.WriteString("\n")
	b.WriteString(s.selected.Render("    ▸ ") + s.dim.Render("Coolify instance connection details"))
	b.WriteString("\n")
	b.WriteString(s.selected.Render("    ▸ ") + s.dim.Render("API token storage method"))
	b.WriteString("\n")
	b.WriteString(s.selected.Render("    ▸ ") + s.dim.Render("Display theme preferences"))
	b.WriteString("\n\n")
	b.WriteString(w.hint("Press", "Enter", "to start", "Esc", "to quit"))
	return b.String()
}

// renderLogoArt paints the CoolDeck wordmark with a two-tone gradient feel
// so the brand reacts when the user live-previews themes.
func (w *wizard) renderLogoArt() string {
	s := w.styles
	var b strings.Builder
	for i, line := range logoLines {
		// Alternate primary / secondary for a subtle multi-colour mark.
		if i%2 == 0 {
			b.WriteString(s.logo.Render(line))
		} else {
			b.WriteString(s.logoAccent.Render(line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// brandChip is a compact one-line mark used above multi-step screens.
func (w *wizard) brandChip() string {
	s := w.styles
	return s.logo.Render("  ▌ cooldeck") + s.dim.Render("  ·  setup") + "\n" +
		s.rule.Render("  ────────────────────────────────────────") + "\n"
}

func (w *wizard) viewTextStep(n int, title, desc, input string) string {
	s := w.styles
	var b strings.Builder

	b.WriteString(w.brandChip())
	b.WriteString(w.stepHeader(n, title))
	b.WriteString("\n\n")
	for _, line := range strings.Split(desc, "\n") {
		b.WriteString(s.dim.Render("  " + line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(s.dim.Render("  ") + input)
	b.WriteString("\n")

	if w.validErr != "" {
		b.WriteString("\n")
		b.WriteString(s.errStyle.Render("  ✕ " + w.validErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.hint("Press", "Enter", "to continue", "b", "back when empty", "Esc", "quit"))
	return b.String()
}

func (w *wizard) viewSelectionStep(n int, title, desc string) string {
	s := w.styles
	var b strings.Builder

	b.WriteString(w.brandChip())
	b.WriteString(w.stepHeader(n, title))
	b.WriteString("\n\n")
	b.WriteString(s.dim.Render("  " + desc))
	b.WriteString("\n\n")

	for i, opt := range tokenSourceOptions {
		if i == w.tokenSourceIdx {
			b.WriteString(s.selected.Render("  ▸ " + opt.label))
			b.WriteString("\n")
			b.WriteString(s.dim.Render("      " + opt.desc))
		} else {
			b.WriteString(s.unselected.Render("    " + opt.label))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.hint("Use", "↑/↓", "to navigate", "Enter", "to select", "b", "back", "Esc", "quit"))
	return b.String()
}

func (w *wizard) viewTokenStep() string {
	s := w.styles
	var b strings.Builder

	b.WriteString(w.brandChip())
	b.WriteString(w.stepHeader(5, "API Token"))
	b.WriteString("\n\n")

	switch w.tokenSource {
	case config.TokenSourceKeyring:
		b.WriteString(s.dim.Render("  Paste your Coolify API token. It will be stored securely"))
		b.WriteString("\n")
		b.WriteString(s.dim.Render("  in the OS keyring under the key \"" + w.instanceID + "\"."))
		b.WriteString("\n\n")
		b.WriteString(s.dim.Render("  ") + w.tokenInput.View())
	case config.TokenSourcePlaintext:
		b.WriteString(s.dim.Render("  Paste your Coolify API token. It will be stored in"))
		b.WriteString("\n")
		b.WriteString(s.dim.Render("  ") + s.errStyle.Render("⚠ plaintext") + s.dim.Render(" inside config.toml."))
		b.WriteString("\n\n")
		b.WriteString(s.dim.Render("  ") + w.tokenInput.View())
	case config.TokenSourceEnv:
		b.WriteString(s.dim.Render("  Enter the environment variable name CoolDeck should read"))
		b.WriteString("\n")
		b.WriteString(s.dim.Render("  the API token from at startup."))
		b.WriteString("\n\n")
		b.WriteString(s.dim.Render("  ") + w.tokenEnvInput.View())
	}

	b.WriteString("\n")

	if w.validErr != "" {
		b.WriteString("\n")
		b.WriteString(s.errStyle.Render("  ✕ " + w.validErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.hint("Press", "Enter", "to continue", "b", "back when empty", "Esc", "quit"))
	return b.String()
}

func (w *wizard) viewThemeStep() string {
	s := w.styles
	var b strings.Builder

	b.WriteString(w.brandChip())
	b.WriteString(w.stepHeader(6, "Theme"))
	b.WriteString("\n\n")
	b.WriteString(s.dim.Render("  Choose a colour scheme (live preview)."))
	b.WriteString("\n\n")

	// Scrollable window so short terminals do not overflow.
	maxVisible := w.themeListVisible()
	start, end := windowAround(w.themeIdx, len(themeOptions), maxVisible)

	if start > 0 {
		b.WriteString(s.dim.Render("    ⋯"))
		b.WriteString("\n")
	}
	for i := start; i < end; i++ {
		opt := themeOptions[i]
		if i == w.themeIdx {
			b.WriteString(s.selected.Render("  ▸ " + opt.label))
		} else {
			b.WriteString(s.unselected.Render("    " + opt.label))
		}
		b.WriteString("\n")
	}
	if end < len(themeOptions) {
		b.WriteString(s.dim.Render("    ⋯"))
		b.WriteString("\n")
	}

	// Mini swatch strip so the active palette is obvious even on short lists.
	b.WriteString("\n")
	b.WriteString(s.dim.Render("  "))
	b.WriteString(s.accent.Render("██"))
	b.WriteString(s.success.Render("██"))
	b.WriteString(s.warn.Render("██"))
	b.WriteString(s.errStyle.Render("██"))
	b.WriteString(s.title.Render("██"))
	b.WriteString("\n\n")
	b.WriteString(w.hint("Use", "↑/↓", "to navigate", "Enter", "to select", "b", "back", "Esc", "quit"))
	return b.String()
}

func (w *wizard) viewConfirm() string {
	s := w.styles
	var b strings.Builder

	b.WriteString(w.brandChip())
	b.WriteString(w.stepHeader(7, "Confirm"))
	b.WriteString("\n\n")
	b.WriteString(s.dim.Render("  Review your configuration before saving:"))
	b.WriteString("\n\n")

	var box strings.Builder
	box.WriteString(w.summaryRow("Instance ID", w.instanceID))
	box.WriteString(w.summaryRow("Display Name", w.instanceName))
	box.WriteString(w.summaryRow("Coolify URL", w.coolifyURL))
	box.WriteString(w.summaryRow("Token Source", string(w.tokenSource)))
	switch w.tokenSource {
	case config.TokenSourceKeyring:
		box.WriteString(w.summaryRow("Token Key", w.instanceID))
		box.WriteString(w.summaryRow("Token", "•••••••• (OS keyring)"))
	case config.TokenSourceEnv:
		box.WriteString(w.summaryRow("Env Variable", "$"+w.tokenEnvVar))
	case config.TokenSourcePlaintext:
		box.WriteString(w.summaryRow("Token", "•••••••• (plaintext in config)"))
	}
	box.WriteString(w.summaryRow("Theme", string(w.theme)))

	b.WriteString(s.box.Render(box.String()))
	b.WriteString("\n")

	if w.configExists {
		b.WriteString("\n")
		b.WriteString("  " + s.warn.Render("⚠ Config already exists and will be overwritten:"))
		b.WriteString("\n")
		b.WriteString("  " + s.dim.Render(w.configPath))
		b.WriteString("\n")
	}

	if w.tokenSource == config.TokenSourcePlaintext {
		b.WriteString("\n")
		if !w.plaintextAck {
			b.WriteString("  " + s.errStyle.Render("⚠ Token will be stored in plaintext."))
			b.WriteString("\n")
			b.WriteString("  " + s.dim.Render("Press Enter once more to acknowledge, then save."))
		} else {
			b.WriteString("  " + s.warn.Render("Plaintext storage acknowledged. Press Enter to save."))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.hint("Press", "Enter/y", "to save", "b", "back", "Esc", "quit"))
	return b.String()
}

func (w *wizard) viewSaving() string {
	s := w.styles
	var b strings.Builder
	b.WriteString(w.brandChip())
	b.WriteString("\n")
	b.WriteString(s.dim.Render("  ") + s.accent.Render("⋯") + s.dim.Render(" Saving configuration…"))
	b.WriteString("\n")
	return b.String()
}

func (w *wizard) viewDone() string {
	s := w.styles
	var b strings.Builder

	b.WriteString(w.brandChip())
	b.WriteString("\n")
	if w.err != nil {
		b.WriteString(s.errStyle.Render("  ✕ Error: " + w.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(w.hint("Press", "r", "to retry", "b", "back", "Esc", "to exit"))
		return b.String()
	}

	b.WriteString(s.success.Render("  ✓  Configuration saved successfully!"))
	b.WriteString("\n")
	b.WriteString(s.rule.Render("  ────────────────────────────────────────"))
	b.WriteString("\n\n")
	b.WriteString(s.dim.Render("  Config file: ") + s.accent.Render(w.configPath))
	b.WriteString("\n\n")

	switch w.tokenSource {
	case config.TokenSourceKeyring:
		b.WriteString(s.dim.Render("  ") + s.success.Render("✓") + s.dim.Render(" Token stored in OS keyring."))
		b.WriteString("\n\n")
	case config.TokenSourceEnv:
		b.WriteString(s.dim.Render("  Make sure ") + s.accent.Render("$"+w.tokenEnvVar) + s.dim.Render(" is set in your shell."))
		b.WriteString("\n\n")
	case config.TokenSourcePlaintext:
		b.WriteString(s.dim.Render("  ") + s.errStyle.Render("⚠") + s.dim.Render(" Token stored in plaintext. Consider switching to keyring."))
		b.WriteString("\n\n")
	}

	b.WriteString(s.title.Render("  Next steps"))
	b.WriteString("\n")
	b.WriteString(s.selected.Render("    ▸ ") + s.dim.Render("Run ") + s.accent.Render("cooldeck") + s.dim.Render(" to launch the dashboard"))
	b.WriteString("\n")
	b.WriteString(s.selected.Render("    ▸ ") + s.dim.Render("Run ") + s.accent.Render("cooldeck --demo") + s.dim.Render(" to try the demo mode"))
	b.WriteString("\n")
	b.WriteString(s.selected.Render("    ▸ ") + s.dim.Render("Run ") + s.accent.Render("cooldeck config validate") + s.dim.Render(" to verify your config"))
	b.WriteString("\n\n")
	b.WriteString(w.hint("Press", "Enter", "or", "Esc", "to exit"))
	return b.String()
}

// ---------------------------------------------------------------------------
// Rendering helpers
// ---------------------------------------------------------------------------

func (w *wizard) frame(body string) string {
	if w.width <= 0 || w.height <= 0 {
		return body
	}
	// Single layer only: Width+Height+Align already centers and paints the
	// full cell. Place() + Width() together double-pads shorter lines and
	// leaves unstyled (often white) trails after SGR resets.
	//
	// If Place is ever needed again, always pass:
	//   lipgloss.WithWhitespaceStyle(w.styles.app)
	return w.styles.app.
		Width(max(w.width, 1)).
		Height(max(w.height, 1)).
		Align(lipgloss.Center, lipgloss.Center).
		Render(body)
}

func (w *wizard) contentWidth() int {
	if w.width <= 0 {
		return contentMaxWidth
	}
	return min(w.width-4, contentMaxWidth)
}

func (w *wizard) themeListVisible() int {
	// Reserve room for header, hints, and chrome.
	h := w.height
	if h <= 0 {
		return 8
	}
	visible := h - 12
	if visible < 3 {
		return 3
	}
	if visible > len(themeOptions) {
		return len(themeOptions)
	}
	return visible
}

func (w *wizard) stepHeader(n int, title string) string {
	s := w.styles
	filled := n
	total := totalUserSteps
	var bar strings.Builder
	for i := 1; i <= total; i++ {
		if i <= filled {
			bar.WriteString(s.progressOn.Render("█"))
		} else {
			bar.WriteString(s.progressOff.Render("░"))
		}
	}
	counter := s.stepCounter.Render(fmt.Sprintf("  Step %d/%d", n, total))
	return s.dim.Render("  ") + s.title.Render("▎ "+title) + "\n" +
		s.dim.Render("  ") + bar.String() + counter
}

func (w *wizard) summaryRow(label, value string) string {
	s := w.styles
	return s.dim.Render("  ") +
		s.key.Render(fmt.Sprintf("%-14s", label+":")) +
		s.dim.Render("  ") +
		s.accent.Render(value) +
		s.dim.Render("\n")
}

// hint builds a bottom navigation bar. Arguments alternate between descriptive
// text and key names: ("Press", "Enter", "to continue", "b", "back").
//
// Every character including spaces must go through a Background-styled Render;
// a bare " " after a key leaves the terminal default (white) behind.
func (w *wizard) hint(parts ...string) string {
	s := w.styles
	var b strings.Builder
	b.WriteString(s.dim.Render("  "))
	for i, p := range parts {
		if i%2 == 0 {
			b.WriteString(s.dim.Render(p + " "))
		} else {
			b.WriteString(s.key.Render("[" + p + "] "))
		}
	}
	return b.String()
}

// windowAround returns a [start,end) window of size maxVisible centred on idx.
func windowAround(idx, total, maxVisible int) (int, int) {
	if total <= maxVisible {
		return 0, total
	}
	start := idx - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
}

// envVarNameOK validates a POSIX-ish environment variable name.
func envVarNameOK(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r == '_':
			// ok
		case i > 0 && r >= '0' && r <= '9':
			// ok
		default:
			return false
		}
	}
	return true
}

// titleCase is a simple replacement for strings.Title which is deprecated.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
