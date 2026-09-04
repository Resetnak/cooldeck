package tui

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/credentials"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// StoreCredentialsFunc writes a token for an instance (typically keyring).
type StoreCredentialsFunc func(inst config.Instance, token string) error

// instanceFormMode selects add vs edit behaviour.
type instanceFormMode int

const (
	instanceFormAdd instanceFormMode = iota
	instanceFormEdit
)

// instanceFormField is the focused input on the form.
type instanceFormField int

const (
	formFieldID instanceFormField = iota
	formFieldName
	formFieldURL
	formFieldToken
	formFieldCount
)

// instanceForm is a lightweight modal editor for local instance config.
// Add mode collects id/name/url/token (token → keyring). Edit mode only
// changes non-secret name/url for an existing entry.
type instanceForm struct {
	mode   instanceFormMode
	field  instanceFormField
	id     string
	name   string
	url    string
	token  string
	err    string
	saving bool
}

var instanceIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

func newAddInstanceForm() *instanceForm {
	return &instanceForm{mode: instanceFormAdd}
}

func newEditInstanceForm(inst config.Instance) *instanceForm {
	return &instanceForm{
		mode: instanceFormEdit,
		id:   inst.ID,
		name: inst.Name,
		url:  inst.URL,
	}
}

func (f *instanceForm) title() string {
	if f.mode == instanceFormEdit {
		return "Edit instance"
	}
	return "Add instance"
}

func (f *instanceForm) fieldCount() instanceFormField {
	if f.mode == instanceFormEdit {
		return formFieldURL + 1 // id locked; name + url only → fields name,url mapped via navigation
	}
	return formFieldCount
}

func (f *instanceForm) currentValue() string {
	switch f.field {
	case formFieldID:
		return f.id
	case formFieldName:
		return f.name
	case formFieldURL:
		return f.url
	case formFieldToken:
		return f.token
	default:
		return ""
	}
}

func (f *instanceForm) setCurrent(v string) {
	switch f.field {
	case formFieldID:
		f.id = v
	case formFieldName:
		f.name = v
	case formFieldURL:
		f.url = v
	case formFieldToken:
		f.token = v
	}
}

func (f *instanceForm) nextField(delta int) {
	n := int(f.fieldCount())
	if n == 0 {
		return
	}
	// Edit mode: only name and url (fields 1 and 2).
	if f.mode == instanceFormEdit {
		if f.field != formFieldName && f.field != formFieldURL {
			f.field = formFieldName
		}
		if delta > 0 {
			if f.field == formFieldName {
				f.field = formFieldURL
			} else {
				f.field = formFieldName
			}
			return
		}
		if f.field == formFieldURL {
			f.field = formFieldName
		} else {
			f.field = formFieldURL
		}
		return
	}
	f.field = instanceFormField((int(f.field) + delta%n + n) % n)
}

func (f *instanceForm) validate() error {
	id := strings.TrimSpace(f.id)
	name := strings.TrimSpace(f.name)
	rawURL := strings.TrimSpace(f.url)
	if f.mode == instanceFormAdd {
		if !instanceIDPattern.MatchString(id) {
			return fmt.Errorf("id must be lowercase, start with a letter, and use a-z 0-9 _ - (max 32)")
		}
	}
	if name == "" {
		return fmt.Errorf("name is required")
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("url must be http(s) with a host")
	}
	if f.mode == instanceFormAdd && strings.TrimSpace(f.token) == "" {
		return fmt.Errorf("API token is required")
	}
	return nil
}

func (f *instanceForm) toInstance() config.Instance {
	id := strings.TrimSpace(f.id)
	return config.Instance{
		ID:          id,
		Name:        strings.TrimSpace(f.name),
		URL:         strings.TrimRight(strings.TrimSpace(f.url), "/"),
		TokenSource: config.TokenSourceKeyring,
		TokenKey:    id,
	}
}

// openAddInstanceForm starts the add-instance modal.
func (m *Model) openAddInstanceForm() tea.Cmd {
	if m.opts.Demo {
		return m.pushToast(components.ToastWarning, "Cannot add in demo mode", "Restart without --demo.")
	}
	if m.opts.SaveConfig == nil {
		return m.pushToast(components.ToastWarning, "Cannot add instance", "Config writing is unavailable.")
	}
	m.instanceForm = newAddInstanceForm()
	return nil
}

// openEditInstanceForm starts the edit modal for the selected instance.
func (m *Model) openEditInstanceForm() tea.Cmd {
	if m.opts.Demo {
		return m.pushToast(components.ToastWarning, "Cannot edit in demo mode", "")
	}
	if m.opts.SaveConfig == nil {
		return m.pushToast(components.ToastWarning, "Cannot edit instance", "Config writing is unavailable.")
	}
	row, ok := m.instances.Selected()
	if !ok {
		return nil
	}
	inst, err := m.opts.Config.Instance(row.ID)
	if err != nil {
		return m.pushToast(components.ToastError, "Edit failed", err.Error())
	}
	m.instanceForm = newEditInstanceForm(inst)
	m.instanceForm.field = formFieldName
	return nil
}

func (m *Model) closeInstanceForm() {
	m.instanceForm = nil
}

func (m *Model) handleInstanceFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	f := m.instanceForm
	if f == nil || f.saving {
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.closeInstanceForm()
		return m, nil
	case "tab", "down", "ctrl+n":
		f.nextField(1)
		f.err = ""
		return m, nil
	case "shift+tab", "up", "ctrl+p":
		f.nextField(-1)
		f.err = ""
		return m, nil
	case "enter":
		return m, m.submitInstanceForm()
	case "backspace":
		cur := f.currentValue()
		if len(cur) > 0 {
			r := []rune(cur)
			f.setCurrent(string(r[:len(r)-1]))
		}
		f.err = ""
		return m, nil
	case "ctrl+u":
		f.setCurrent("")
		f.err = ""
		return m, nil
	}
	// String() renders space as "space"; Text carries the literal input,
	// and is empty for keys that produce none.
	if text := msg.Key().Text; text != "" {
		f.setCurrent(f.currentValue() + text)
		f.err = ""
	}
	return m, nil
}

func (m *Model) submitInstanceForm() tea.Cmd {
	f := m.instanceForm
	if f == nil {
		return nil
	}
	if err := f.validate(); err != nil {
		f.err = err.Error()
		return nil
	}
	inst := f.toInstance()

	// Build the new config on a copied map so nothing changes on the model
	// until the write succeeds.
	cfg := m.opts.Config
	cfg.Instances = cloneInstances(cfg.Instances)
	if f.mode == instanceFormAdd {
		if _, exists := cfg.Instances[inst.ID]; exists {
			f.err = "an instance with this id already exists"
			return nil
		}
		cfg.Instances[inst.ID] = inst
		if cfg.DefaultInstance == "" {
			cfg.DefaultInstance = inst.ID
		}
	} else {
		existing, err := cfg.Instance(inst.ID)
		if err != nil {
			f.err = err.Error()
			return nil
		}
		existing.Name = inst.Name
		existing.URL = inst.URL
		cfg.Instances[inst.ID] = existing
	}

	// Keyring and disk writes run inside the command, never on the event
	// loop: a macOS Keychain prompt can block for seconds, and the UI -
	// including ctrl+c - would freeze with it. f.saving keeps the form inert
	// until the result message lands.
	f.saving = true
	store := m.opts.StoreCredentials
	save := m.opts.SaveConfig
	token := strings.TrimSpace(f.token)
	mode := f.mode
	return func() tea.Msg {
		if mode == instanceFormAdd {
			// Persist token before config so a failed keyring write does not
			// leave a config pointing at a missing secret.
			var err error
			if store != nil {
				err = store(inst, token)
			} else {
				// Fallback: store directly when wired without a callback.
				err = credentials.Store(credentials.KeyFor(inst), credentials.NewToken(token))
			}
			if err != nil {
				return instanceFormSavedMsg{Err: fmt.Errorf("store token: %w", err)}
			}
		}
		if save != nil {
			if err := save(cfg); err != nil {
				return instanceFormSavedMsg{Err: fmt.Errorf("save config: %w", err)}
			}
		}
		return instanceFormSavedMsg{Config: cfg, Instance: inst, Mode: mode}
	}
}

// applyInstanceFormSaved commits or reports the result of the asynchronous
// form submission.
func (m *Model) applyInstanceFormSaved(msg instanceFormSavedMsg) tea.Cmd {
	f := m.instanceForm
	if msg.Err != nil {
		if f != nil {
			f.saving = false
			f.err = msg.Err.Error()
		}
		return nil
	}

	m.opts.Config = msg.Config
	name := msg.Instance.Name
	id := msg.Instance.ID
	m.closeInstanceForm()
	m.refreshInstances()
	m.refreshDiagnostics()

	if msg.Mode == instanceFormAdd && m.opts.OpenService != nil {
		return tea.Batch(
			m.pushToast(components.ToastSuccess, "Instance added", name),
			m.switchInstance(id),
		)
	}
	text := "Instance updated"
	if msg.Mode == instanceFormAdd {
		text = "Instance added"
	}
	return m.pushToast(components.ToastSuccess, text, name)
}

func (m *Model) renderInstanceForm() string {
	f := m.instanceForm
	if f == nil || m.layout.Width < 28 || m.layout.Height < 12 {
		return ""
	}
	th := m.theme
	width := min(m.layout.Width-6, 56)

	row := func(label string, value string, field instanceFormField, secret bool) string {
		display := value
		if secret && display != "" {
			display = strings.Repeat("•", min(len([]rune(display)), 24))
		}
		if display == "" {
			display = " "
		}
		cursor := ""
		style := th.ModalBody
		if f.field == field {
			cursor = "▏"
			style = th.PaletteActive
		}
		line := components.Pad(label, 10) + " " + style.Render(display+cursor)
		return line
	}

	lines := []string{
		th.ModalTitle.Render(f.title()),
		th.ModalHint.Render("tab next field  ·  enter save  ·  esc cancel"),
		"",
	}
	if f.mode == instanceFormAdd {
		lines = append(lines, row("ID", f.id, formFieldID, false))
	} else {
		lines = append(lines, th.ModalBody.Render(components.Pad("ID", 10)+" "+f.id+"  (locked)"))
	}
	lines = append(lines,
		row("Name", f.name, formFieldName, false),
		row("URL", f.url, formFieldURL, false),
	)
	if f.mode == instanceFormAdd {
		lines = append(lines, row("Token", f.token, formFieldToken, true))
		lines = append(lines, th.ModalHint.Render("Token is stored in the OS keyring, not the config file."))
	}
	if f.err != "" {
		lines = append(lines, "", th.Danger.Render(f.err))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return theme.Fill(th.Modal.Width(width), content)
}

func (m *Model) overlayInstanceForm(frame string) string {
	modal := m.renderInstanceForm()
	if modal == "" {
		return frame
	}
	x := max((m.layout.Width-lipgloss.Width(modal))/2, 0)
	y := max((m.layout.Height-lipgloss.Height(modal))/4, 1)
	return components.Overlay(frame, modal, x, y)
}
