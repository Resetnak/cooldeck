// Package setup implements an interactive TUI wizard for first-time CoolDeck
// configuration. It walks the user through Coolify instance details, token
// storage, and display preferences, then writes a valid config.toml.
package setup

import (
	"fmt"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/credentials"
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
	stepDone
)

const totalUserSteps = 7 // steps the user actively fills in (welcome & done excluded)

// ---------------------------------------------------------------------------
// Styles
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
	box         lipgloss.Style
	logo        lipgloss.Style
}

func newStyles() wizardStyles {
	return wizardStyles{
		title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")),
		subtitle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")),
		accent:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#06B6D4")),
		dim:         lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		success:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10B981")),
		errStyle:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#EF4444")),
		selected:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")),
		unselected:  lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		key:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F59E0B")),
		stepCounter: lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")),
		box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4B5563")).
			Padding(1, 2),
		logo: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")),
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
	desc  string
}{
	{config.ThemeAuto, "Auto", "Detect from terminal background."},
	{config.ThemeDark, "Dark", "Always use dark theme."},
	{config.ThemeLight, "Light", "Always use light theme."},
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

	// result
	configPath string
	err        error
	validErr   string

	styles wizardStyles
}

// instanceIDRe matches valid instance IDs (same rule as config.validate).
var instanceIDRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.\-]{0,63}$`)

// New returns a fresh wizard model.
func New() *wizard {
	s := newStyles()

	idInput := textinput.New()
	idInput.Placeholder = "production"
	idInput.CharLimit = 64

	nameInput := textinput.New()
	nameInput.Placeholder = "Production"
	nameInput.CharLimit = 128

	urlIn := textinput.New()
	urlIn.Placeholder = "https://coolify.example.com"
	urlIn.CharLimit = 256

	tokIn := textinput.New()
	tokIn.Placeholder = "paste your Coolify API token"
	tokIn.EchoMode = textinput.EchoPassword
	tokIn.EchoCharacter = '•'
	tokIn.CharLimit = 512

	tokEnvIn := textinput.New()
	tokEnvIn.Placeholder = "COOLIFY_API_TOKEN"
	tokEnvIn.CharLimit = 128

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

// Run launches the setup wizard and blocks until it exits.
func Run() error {
	p := tea.NewProgram(New())
	_, err := p.Run()
	return err
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
		return w, nil

	case tea.KeyPressMsg:
		// Global quit.
		if msg.Code == tea.KeyEscape || (msg.Mod == tea.ModCtrl && msg.Code == 'c') {
			return w, tea.Quit
		}
		return w.handleKey(msg)
	}

	// Forward to active text input.
	return w.updateActiveInput(msg)
}

func (w *wizard) View() tea.View {
	var b strings.Builder

	switch w.step {
	case stepWelcome:
		b.WriteString(w.viewWelcome())
	case stepInstanceID:
		b.WriteString(w.viewTextStep(1, "Instance ID",
			"A short identifier for this Coolify instance (e.g. production, staging).\nUsed as the config key and keyring entry name.",
			w.instanceIDInput.View()))
	case stepInstanceName:
		b.WriteString(w.viewTextStep(2, "Display Name",
			"A human-friendly label shown in the dashboard header.",
			w.instanceNameInput.View()))
	case stepURL:
		b.WriteString(w.viewTextStep(3, "Coolify URL",
			"The base URL of your Coolify dashboard.\nCoolDeck will auto-detect and strip /api/v1 suffixes.",
			w.urlInput.View()))
	case stepTokenSource:
		b.WriteString(w.viewSelectionStep(4, "Token Storage",
			"How should CoolDeck store your Coolify API token?"))
	case stepToken:
		b.WriteString(w.viewTokenStep())
	case stepTheme:
		b.WriteString(w.viewThemeStep())
	case stepConfirm:
		b.WriteString(w.viewConfirm())
	case stepDone:
		b.WriteString(w.viewDone())
	}

	return tea.NewView(b.String())
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
				w.tokenEnvVar = v
			}
			w.step = stepTheme
			w.tokenInput.Blur()
			w.tokenEnvInput.Blur()
			w.validErr = ""
			return w, nil
		}
		return w.updateActiveInput(key)

	case stepTheme:
		switch {
		case key.Text == "b":
			w.step = stepToken
			w.validErr = ""
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
			}
		case key.Code == tea.KeyDown || key.Text == "j":
			if w.themeIdx < len(themeOptions)-1 {
				w.themeIdx++
			}
		case key.Code == tea.KeyEnter:
			w.theme = themeOptions[w.themeIdx].theme
			w.step = stepConfirm
			w.validErr = ""
		}
		return w, nil

	case stepConfirm:
		switch {
		case key.Text == "b":
			w.step = stepTheme
			w.validErr = ""
			return w, nil
		case key.Code == tea.KeyEnter || key.Text == "y":
			w.saveConfig()
			w.step = stepDone
			return w, nil
		}
		return w, nil

	case stepDone:
		if key.Code == tea.KeyEnter || key.Text == "q" {
			return w, tea.Quit
		}
		return w, nil
	}

	return w, nil
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

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

func (w *wizard) saveConfig() {
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

	path, err := config.DefaultPath()
	if err != nil {
		w.err = err
		return
	}
	w.configPath = path

	if err := cfg.Save(""); err != nil {
		w.err = fmt.Errorf("save config: %w", err)
		return
	}

	// Store token in keyring if that source was chosen.
	if w.tokenSource == config.TokenSourceKeyring && w.token != "" {
		if err := credentials.Store(w.instanceID, credentials.NewToken(w.token)); err != nil {
			w.err = fmt.Errorf("store token in keyring: %w", err)
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Views
// ---------------------------------------------------------------------------

const logo = `   ___            _ ___          _   
  / __|___  ___  | |   \ ___  __| |__
 | (__/ _ \/ _ \ | | |) / -_)/ _| / /
  \___\___/\___/ |_|___/\___|\____|_\_\`

func (w *wizard) viewWelcome() string {
	s := w.styles
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(s.logo.Render(logo))
	b.WriteString("\n\n")
	b.WriteString(s.title.Render("  ⚡ Setup Wizard"))
	b.WriteString("\n\n")
	b.WriteString(s.subtitle.Render("  Welcome to CoolDeck! This wizard will guide you through"))
	b.WriteString("\n")
	b.WriteString(s.subtitle.Render("  creating your first configuration file."))
	b.WriteString("\n\n")
	b.WriteString(s.dim.Render("  You'll configure:"))
	b.WriteString("\n")
	b.WriteString(s.dim.Render("   • Coolify instance connection details"))
	b.WriteString("\n")
	b.WriteString(s.dim.Render("   • API token storage method"))
	b.WriteString("\n")
	b.WriteString(s.dim.Render("   • Display theme preferences"))
	b.WriteString("\n\n")
	b.WriteString(w.hint("Press", "Enter", "to start", "Esc", "to quit"))
	return b.String()
}

func (w *wizard) viewTextStep(n int, title, desc, input string) string {
	s := w.styles
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(w.stepHeader(n, title))
	b.WriteString("\n\n")
	for _, line := range strings.Split(desc, "\n") {
		b.WriteString("  ")
		b.WriteString(s.dim.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString("  " + input)
	b.WriteString("\n")

	if w.validErr != "" {
		b.WriteString("\n")
		b.WriteString("  " + s.errStyle.Render("✗ "+w.validErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.hint("Press", "Enter", "to continue", "b", "back when empty"))
	return b.String()
}

func (w *wizard) viewSelectionStep(n int, title, desc string) string {
	s := w.styles
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(w.stepHeader(n, title))
	b.WriteString("\n\n")
	b.WriteString("  " + s.dim.Render(desc))
	b.WriteString("\n\n")

	for i, opt := range tokenSourceOptions {
		if i == w.tokenSourceIdx {
			b.WriteString("  " + s.selected.Render("▸ "+opt.label))
			b.WriteString("\n")
			b.WriteString("    " + s.dim.Render(opt.desc))
		} else {
			b.WriteString("  " + s.unselected.Render("  "+opt.label))
			b.WriteString("\n")
			b.WriteString("    " + s.dim.Render(opt.desc))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.hint("Use", "↑/↓", "to navigate", "Enter", "to select", "b", "back"))
	return b.String()
}

func (w *wizard) viewTokenStep() string {
	s := w.styles
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(w.stepHeader(5, "API Token"))
	b.WriteString("\n\n")

	switch w.tokenSource {
	case config.TokenSourceKeyring:
		b.WriteString("  " + s.dim.Render("Paste your Coolify API token. It will be stored securely"))
		b.WriteString("\n")
		b.WriteString("  " + s.dim.Render("in the OS keyring under the key \""+w.instanceID+"\"."))
		b.WriteString("\n\n")
		b.WriteString("  " + w.tokenInput.View())
	case config.TokenSourcePlaintext:
		b.WriteString("  " + s.dim.Render("Paste your Coolify API token. It will be stored in"))
		b.WriteString("\n")
		b.WriteString("  " + s.errStyle.Render("⚠ plaintext") + s.dim.Render(" inside config.toml."))
		b.WriteString("\n\n")
		b.WriteString("  " + w.tokenInput.View())
	case config.TokenSourceEnv:
		b.WriteString("  " + s.dim.Render("Enter the environment variable name CoolDeck should read"))
		b.WriteString("\n")
		b.WriteString("  " + s.dim.Render("the API token from at startup."))
		b.WriteString("\n\n")
		b.WriteString("  " + w.tokenEnvInput.View())
	}

	b.WriteString("\n")

	if w.validErr != "" {
		b.WriteString("\n")
		b.WriteString("  " + s.errStyle.Render("✗ "+w.validErr))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.hint("Press", "Enter", "to continue", "b", "back when empty"))
	return b.String()
}

func (w *wizard) viewThemeStep() string {
	s := w.styles
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(w.stepHeader(6, "Theme"))
	b.WriteString("\n\n")
	b.WriteString("  " + s.dim.Render("Choose a colour scheme for CoolDeck."))
	b.WriteString("\n\n")

	for i, opt := range themeOptions {
		if i == w.themeIdx {
			b.WriteString("  " + s.selected.Render("▸ "+opt.label))
			b.WriteString("  " + s.dim.Render(opt.desc))
		} else {
			b.WriteString("  " + s.unselected.Render("  "+opt.label))
			b.WriteString("  " + s.dim.Render(opt.desc))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(w.hint("Use", "↑/↓", "to navigate", "Enter", "to select", "b", "back"))
	return b.String()
}

func (w *wizard) viewConfirm() string {
	s := w.styles
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(w.stepHeader(7, "Confirm"))
	b.WriteString("\n\n")
	b.WriteString("  " + s.dim.Render("Review your configuration before saving:"))
	b.WriteString("\n\n")

	// Build the summary box content.
	var box strings.Builder
	box.WriteString(w.summaryRow("Instance ID", w.instanceID))
	box.WriteString(w.summaryRow("Display Name", w.instanceName))
	box.WriteString(w.summaryRow("Coolify URL", w.coolifyURL))
	box.WriteString(w.summaryRow("Token Source", string(w.tokenSource)))
	switch w.tokenSource {
	case config.TokenSourceKeyring:
		box.WriteString(w.summaryRow("Token Key", w.instanceID))
		box.WriteString(w.summaryRow("Token", "•••••••• (will be stored in keyring)"))
	case config.TokenSourceEnv:
		box.WriteString(w.summaryRow("Env Variable", "$"+w.tokenEnvVar))
	case config.TokenSourcePlaintext:
		box.WriteString(w.summaryRow("Token", "•••••••• (plaintext in config)"))
	}
	box.WriteString(w.summaryRow("Theme", string(w.theme)))

	b.WriteString(s.box.Render(box.String()))
	b.WriteString("\n\n")
	b.WriteString(w.hint("Press", "Enter/y", "to save", "b", "back"))
	return b.String()
}

func (w *wizard) viewDone() string {
	s := w.styles
	var b strings.Builder

	b.WriteString("\n")
	if w.err != nil {
		b.WriteString("  " + s.errStyle.Render("✗ Error: "+w.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(w.hint("Press", "Enter", "or", "q", "to exit"))
		return b.String()
	}

	b.WriteString("  " + s.success.Render("✓ Configuration saved successfully!"))
	b.WriteString("\n\n")
	b.WriteString("  " + s.dim.Render("Config file: ") + s.accent.Render(w.configPath))
	b.WriteString("\n\n")

	switch w.tokenSource {
	case config.TokenSourceKeyring:
		b.WriteString("  " + s.success.Render("✓") + s.dim.Render(" Token stored in OS keyring."))
		b.WriteString("\n\n")
	case config.TokenSourceEnv:
		b.WriteString("  " + s.dim.Render("Make sure ") + s.accent.Render("$"+w.tokenEnvVar) + s.dim.Render(" is set in your shell."))
		b.WriteString("\n\n")
	case config.TokenSourcePlaintext:
		b.WriteString("  " + s.errStyle.Render("⚠") + s.dim.Render(" Token stored in plaintext. Consider switching to keyring."))
		b.WriteString("\n\n")
	}

	b.WriteString("  " + s.dim.Render("Next steps:"))
	b.WriteString("\n")
	b.WriteString("  " + s.dim.Render("  • Run ") + s.accent.Render("cooldeck") + s.dim.Render(" to launch the dashboard"))
	b.WriteString("\n")
	b.WriteString("  " + s.dim.Render("  • Run ") + s.accent.Render("cooldeck --demo") + s.dim.Render(" to try the demo mode"))
	b.WriteString("\n")
	b.WriteString("  " + s.dim.Render("  • Run ") + s.accent.Render("cooldeck config validate") + s.dim.Render(" to verify your config"))
	b.WriteString("\n\n")
	b.WriteString(w.hint("Press", "Enter", "or", "q", "to exit"))
	return b.String()
}

// ---------------------------------------------------------------------------
// Rendering helpers
// ---------------------------------------------------------------------------

func (w *wizard) stepHeader(n int, title string) string {
	s := w.styles
	counter := s.stepCounter.Render(fmt.Sprintf("Step %d/%d", n, totalUserSteps))
	return "  " + s.title.Render("▎ "+title) + "  " + counter
}

func (w *wizard) summaryRow(label, value string) string {
	s := w.styles
	return fmt.Sprintf("  %s  %s\n", s.key.Render(fmt.Sprintf("%-14s", label+":")), s.accent.Render(value))
}

// hint builds a bottom navigation bar. Arguments alternate between descriptive
// text and key names: ("Press", "Enter", "to continue", "b", "back").
func (w *wizard) hint(parts ...string) string {
	s := w.styles
	var b strings.Builder
	b.WriteString("  ")
	for i, p := range parts {
		if i%2 == 0 {
			b.WriteString(s.dim.Render(p + " "))
		} else {
			b.WriteString(s.key.Render("["+p+"]") + " ")
		}
	}
	return b.String()
}

// titleCase is a simple replacement for strings.Title which is deprecated.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
