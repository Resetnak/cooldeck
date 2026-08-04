// Package themepicker implements a live interactive TUI theme selection wizard.
// As the user navigates between themes, the screen live-previews the full design system.
package themepicker

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

type themeOption struct {
	id    config.Theme
	label string
}

var options = []themeOption{
	{config.ThemeAuto, "Auto"},
	{config.ThemeDark, "Dark"},
	{config.ThemeLight, "Light"},
	{config.ThemeDracula, "Dracula"},
	{config.ThemeCatppuccin, "Catppuccin Mocha"},
	{config.ThemeNord, "Nord"},
	{config.ThemeGruvbox, "Gruvbox Dark"},
	{config.ThemeTokyoNight, "Tokyo Night"},
}

type model struct {
	configPath string
	cfg        config.Config
	cursor     int
	saved      bool
	err        error
	width      int
	height     int
}

// New creates a new theme picker model.
func New(configPath string) (*model, error) {
	cfg, err := config.Load(configPath)
	if err != nil && !errors.Is(err, config.ErrNotFound) && !errors.Is(err, config.ErrUnknownKeys) {
		return nil, err
	}
	if errors.Is(err, config.ErrNotFound) {
		cfg = config.Default()
	}

	initialCursor := 0
	for i, opt := range options {
		if opt.id == cfg.Theme {
			initialCursor = i
			break
		}
	}

	return &model{
		configPath: configPath,
		cfg:        cfg,
		cursor:     initialCursor,
	}, nil
}

// Run launches the interactive theme picker TUI.
func Run(configPath string) error {
	m, err := New(configPath)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	if fm, ok := finalModel.(*model); ok {
		if fm.err != nil {
			return fm.err
		}
		if fm.saved {
			savedPath := fm.cfg.Path()
			if savedPath == "" {
				savedPath, _ = config.DefaultPath()
			}
			fmt.Printf("✓ Theme %q saved to %s\n", fm.options()[fm.cursor].label, savedPath)
		}
	}
	return nil
}

func (m *model) options() []themeOption {
	return options
}

func (m *model) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyEscape:
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case tea.KeyDown:
			if m.cursor < len(options)-1 {
				m.cursor++
			}
			return m, nil
		case tea.KeyEnter:
			selectedTheme := options[m.cursor].id
			m.cfg.Theme = selectedTheme
			if err := m.cfg.Save(m.configPath); err != nil {
				m.err = fmt.Errorf("save config: %w", err)
			} else {
				m.saved = true
			}
			return m, tea.Quit
		}

		switch msg.Text {
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j":
			if m.cursor < len(options)-1 {
				m.cursor++
			}
		case "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *model) activeTheme() *theme.Theme {
	cur := options[m.cursor].id
	mode := theme.ModeDark
	paletteName := ""
	switch cur {
	case config.ThemeLight:
		mode = theme.ModeLight
	case config.ThemeDark:
		mode = theme.ModeDark
	case config.ThemeDracula, config.ThemeCatppuccin, config.ThemeNord, config.ThemeGruvbox, config.ThemeTokyoNight:
		paletteName = string(cur)
	default:
		mode = theme.ModeDark
	}

	return theme.New(theme.Options{
		Mode:        mode,
		PaletteName: paletteName,
	})
}

func (m *model) View() tea.View {
	t := m.activeTheme()

	var b strings.Builder

	// Top header bar styled with theme Background & Text colors
	header := t.HeaderBar.Render(
		t.HeaderLogo.Render("COOLDECK") + "  " +
			t.HeaderInstance.Render("Theme Selector") + "  " +
			t.HeaderMeta.Render("• Live Interactive Preview"),
	)
	b.WriteString(header + "\n")
	b.WriteString(t.HeaderRule.Render(strings.Repeat("─", max(50, m.width))) + "\n\n")

	// Left column: Theme list box with explicit Surface background
	listBoxStyle := lipgloss.NewStyle().
		Background(t.Palette.Surface).
		Border(t.Border).
		BorderForeground(t.Palette.Border).
		Padding(1, 1).
		Width(24)

	var leftBuf strings.Builder
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Palette.Text).
		Background(t.Palette.Surface)

	leftBuf.WriteString(titleStyle.Render(" Select Theme") + "\n\n")

	itemStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextMuted).
		Background(t.Palette.Surface)

	activeItemStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Palette.SelectionText).
		Background(t.Palette.Selection)

	for i, opt := range options {
		if i == m.cursor {
			leftBuf.WriteString(activeItemStyle.Render("▸ "+opt.label) + "\n")
		} else {
			leftBuf.WriteString(itemStyle.Render("  "+opt.label) + "\n")
		}
	}

	leftBox := listBoxStyle.Render(leftBuf.String())

	// Right column: Rich Preview Box with solid background derived from theme Surface/Overlay
	previewBoxStyle := lipgloss.NewStyle().
		Background(t.Palette.Surface).
		Border(t.Border).
		BorderForeground(t.Palette.BorderFocused).
		Padding(1, 2).
		Width(54)

	var rightBuf strings.Builder

	rightTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Palette.Text).
		Background(t.Palette.Surface)

	rightBuf.WriteString(rightTitleStyle.Render("Theme: "+options[m.cursor].label) + "\n")

	// Color Swatches Bar
	labelStyle := lipgloss.NewStyle().
		Foreground(t.Palette.TextMuted).
		Background(t.Palette.Surface)

	swatchBlock := "██"
	swatches := lipgloss.NewStyle().Foreground(t.Palette.Primary).Background(t.Palette.Surface).Render(swatchBlock) + " " +
		lipgloss.NewStyle().Foreground(t.Palette.Secondary).Background(t.Palette.Surface).Render(swatchBlock) + " " +
		lipgloss.NewStyle().Foreground(t.Palette.Success).Background(t.Palette.Surface).Render(swatchBlock) + " " +
		lipgloss.NewStyle().Foreground(t.Palette.Warning).Background(t.Palette.Surface).Render(swatchBlock) + " " +
		lipgloss.NewStyle().Foreground(t.Palette.Error).Background(t.Palette.Surface).Render(swatchBlock) + " " +
		lipgloss.NewStyle().Foreground(t.Palette.BorderFocused).Background(t.Palette.Surface).Render(swatchBlock)

	rightBuf.WriteString(labelStyle.Render("Palette: ") + swatches + "\n\n")

	// Status Badges Sample
	posStyle := lipgloss.NewStyle().Foreground(t.Palette.Success).Background(t.Palette.Surface)
	attStyle := lipgloss.NewStyle().Foreground(t.Palette.Warning).Background(t.Palette.Surface)
	danStyle := lipgloss.NewStyle().Foreground(t.Palette.Error).Background(t.Palette.Surface)
	mutStyle := lipgloss.NewStyle().Foreground(t.Palette.TextMuted).Background(t.Palette.Surface)

	rightBuf.WriteString(labelStyle.Render("Status Badges:") + "\n")
	rightBuf.WriteString(
		posStyle.Render("● Running") + "  " +
			attStyle.Render("▲ Deploying") + "  " +
			danStyle.Render("✖ Failed") + "  " +
			mutStyle.Render("○ Stopped") + "\n\n",
	)

	// Sample Table Card
	rightBuf.WriteString(labelStyle.Render("Dashboard Table:") + "\n")

	thStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Palette.TextSubtle).Background(t.Palette.Overlay)
	tr1Style := lipgloss.NewStyle().Bold(true).Foreground(t.Palette.SelectionText).Background(t.Palette.Selection)
	tr2Style := lipgloss.NewStyle().Foreground(t.Palette.Text).Background(t.Palette.Overlay)
	tr3Style := lipgloss.NewStyle().Foreground(t.Palette.Text).Background(t.Palette.SurfaceRaised)

	tblHeader := thStyle.Render(fmt.Sprintf("%-16s %-10s %-8s", "APPLICATION", "STATUS", "HEALTH"))
	row1 := tr1Style.Render(fmt.Sprintf("%-16s %-10s %-8s", "api-gateway", "Running", "100%"))
	row2 := tr2Style.Render(fmt.Sprintf("%-16s %-10s %-8s", "web-frontend", "Deploying", "98%"))
	row3 := tr3Style.Render(fmt.Sprintf("%-16s %-10s %-8s", "redis-cache", "Stopped", "0%"))

	tableCardStyle := lipgloss.NewStyle().
		Background(t.Palette.Overlay).
		Border(t.Border).
		BorderForeground(t.Palette.BorderSubtle).
		Padding(0, 1)

	tableBox := tableCardStyle.Render(tblHeader + "\n" + row1 + "\n" + row2 + "\n" + row3)
	rightBuf.WriteString(tableBox + "\n\n")

	// Sample Toast / Notification with explicit Overlay background
	toastStyle := lipgloss.NewStyle().
		Border(t.Border).
		BorderForeground(t.Palette.Success).
		Foreground(t.Palette.Success).
		Background(t.Palette.Overlay).
		Padding(0, 1)

	toast := toastStyle.Render("✓ Saved configuration preview active")
	rightBuf.WriteString(toast)

	rightBox := previewBoxStyle.Render(rightBuf.String())

	// Combine Left and Right columns side-by-side using Lip Gloss
	combined := lipgloss.JoinHorizontal(lipgloss.Top,
		leftBox,
		"  ",
		rightBox,
	)

	// Main App container with solid Theme Background
	mainAppStyle := lipgloss.NewStyle().
		Background(t.Palette.Background).
		Padding(1, 1)

	b.WriteString(mainAppStyle.Render(combined) + "\n\n")

	// Footer hints
	footerKeyStyle := lipgloss.NewStyle().Bold(true).Foreground(t.Palette.Secondary).Background(t.Palette.Background)
	footerDescStyle := lipgloss.NewStyle().Foreground(t.Palette.TextMuted).Background(t.Palette.Background)

	footer := footerKeyStyle.Render("[↑/↓ or k/j]") + " " + footerDescStyle.Render("Navigate") + "   " +
		footerKeyStyle.Render("[Enter]") + " " + footerDescStyle.Render("Save & Apply") + "   " +
		footerKeyStyle.Render("[Esc/q]") + " " + footerDescStyle.Render("Cancel")

	b.WriteString(footer + "\n")

	return tea.NewView(b.String())
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
