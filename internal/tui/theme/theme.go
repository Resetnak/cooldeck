package theme

import (
	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
)

// Mode is the resolved colour scheme. "auto" is resolved before a Theme is
// built, so a Theme is always concretely dark or light.
type Mode string

// Resolved modes.
const (
	ModeDark  Mode = "dark"
	ModeLight Mode = "light"
)

// Options selects which palette and glyphs a Theme is built from.
type Options struct {
	Mode Mode
	// PaletteName selects a named palette (e.g. "dracula"). When set, Mode
	// is ignored for palette selection but still recorded on the Theme.
	PaletteName string
	// NerdFont upgrades a handful of glyphs. Never required.
	NerdFont bool
	// ASCII forces the plain-text glyph set for terminals that mangle
	// box-drawing characters.
	ASCII bool
}

// Theme bundles the palette, the glyph set and every style the UI draws with.
// Styles are built once, when the theme is created, because rebuilding them
// per frame shows up immediately as input lag over SSH.
type Theme struct {
	Mode    Mode
	Palette Palette
	Sym     SymbolSet

	// Base surfaces.
	App     lipgloss.Style
	Panel   lipgloss.Style
	PanelOn lipgloss.Style
	Title   lipgloss.Style

	// Typography roles.
	Text      lipgloss.Style
	Muted     lipgloss.Style
	Subtle    lipgloss.Style
	Strong    lipgloss.Style
	Accent    lipgloss.Style
	Label     lipgloss.Style
	Value     lipgloss.Style
	Danger    lipgloss.Style
	Positive  lipgloss.Style
	Attention lipgloss.Style
	Note      lipgloss.Style

	// Header.
	HeaderBar      lipgloss.Style
	HeaderLogo     lipgloss.Style
	HeaderInstance lipgloss.Style
	HeaderMeta     lipgloss.Style
	HeaderRule     lipgloss.Style

	// Footer.
	FooterBar  lipgloss.Style
	FooterKey  lipgloss.Style
	FooterDesc lipgloss.Style
	FooterSep  lipgloss.Style

	// Navigation.
	NavTitle       lipgloss.Style
	NavItem        lipgloss.Style
	NavItemActive  lipgloss.Style
	NavItemBlurred lipgloss.Style
	NavCount       lipgloss.Style
	TabActive      lipgloss.Style
	TabInactive    lipgloss.Style

	// Table.
	TableHeader      lipgloss.Style
	TableRow         lipgloss.Style
	TableRowAlt      lipgloss.Style
	TableRowActive   lipgloss.Style
	TableRowInactive lipgloss.Style
	TableCellMuted   lipgloss.Style
	TableMarker      lipgloss.Style

	// Modals and overlays.
	Modal        lipgloss.Style
	ModalDanger  lipgloss.Style
	ModalTitle   lipgloss.Style
	ModalBody    lipgloss.Style
	ModalHint    lipgloss.Style
	Backdrop     lipgloss.Style
	ButtonPrime  lipgloss.Style
	ButtonDanger lipgloss.Style
	ButtonGhost  lipgloss.Style

	// Toasts.
	ToastSuccess lipgloss.Style
	ToastInfo    lipgloss.Style
	ToastWarning lipgloss.Style
	ToastError   lipgloss.Style

	// Empty, loading and error states.
	EmptyTitle lipgloss.Style
	EmptyBody  lipgloss.Style
	EmptyHint  lipgloss.Style
	Skeleton   lipgloss.Style
	Spinner    lipgloss.Style

	// Command palette.
	PaletteInput    lipgloss.Style
	PaletteItem     lipgloss.Style
	PaletteActive   lipgloss.Style
	PaletteShortcut lipgloss.Style
	PaletteDisabled lipgloss.Style
	PaletteReason   lipgloss.Style
	PaletteGroup    lipgloss.Style

	// Filter input.
	FilterPrompt lipgloss.Style
	FilterText   lipgloss.Style
	FilterHint   lipgloss.Style

	// Logs.
	LogTimestamp lipgloss.Style
	LogText      lipgloss.Style
	LogError     lipgloss.Style
	LogWarn      lipgloss.Style
	LogInfo      lipgloss.Style
	LogDebug     lipgloss.Style
	LogMatch     lipgloss.Style

	// Badges, keyed by semantic role.
	badgeNeutral lipgloss.Style
	badgeSuccess lipgloss.Style
	badgeWarning lipgloss.Style
	badgeError   lipgloss.Style
	badgeInfo    lipgloss.Style

	// statusFg maps a normalised state onto its foreground style.
	statusFg map[domain.ResourceStatus]lipgloss.Style

	// Border is the box style used for panels, chosen for the terminal's
	// glyph capability.
	Border lipgloss.Border
}

// New builds a theme. It is called once at startup and again whenever the
// user toggles the theme or the terminal reports a background colour change.
func New(opts Options) *Theme {
	p := resolvePalette(opts)

	sym := UnicodeSymbols
	switch {
	case opts.ASCII:
		sym = ASCIISymbols
	case opts.NerdFont:
		sym = NerdFontSymbols
	}

	border := lipgloss.RoundedBorder()
	if opts.ASCII {
		border = lipgloss.ASCIIBorder()
	}

	base := lipgloss.NewStyle().Foreground(p.Text)
	t := &Theme{Mode: opts.Mode, Palette: p, Sym: sym, Border: border}

	t.App = lipgloss.NewStyle().Background(p.Background).Foreground(p.Text)
	t.Panel = lipgloss.NewStyle().
		Border(border).
		BorderForeground(p.Border).
		Padding(0, SpaceXS)
	t.PanelOn = t.Panel.BorderForeground(p.BorderFocused)
	t.Title = base.Bold(true).Foreground(p.Text)

	t.Text = base
	t.Muted = lipgloss.NewStyle().Foreground(p.TextMuted)
	t.Subtle = lipgloss.NewStyle().Foreground(p.TextSubtle)
	t.Strong = base.Bold(true)
	t.Accent = lipgloss.NewStyle().Foreground(p.Primary)
	t.Label = lipgloss.NewStyle().Foreground(p.TextMuted)
	t.Value = base
	t.Danger = lipgloss.NewStyle().Foreground(p.Error)
	t.Positive = lipgloss.NewStyle().Foreground(p.Success)
	t.Attention = lipgloss.NewStyle().Foreground(p.Warning)
	t.Note = lipgloss.NewStyle().Foreground(p.Info)

	// Header: a single surface line plus a hairline rule. Anything taller
	// steals rows from the table, which is the only content that actually matters.
	t.HeaderBar = lipgloss.NewStyle().
		Background(p.Surface).
		Foreground(p.Text).
		Padding(0, SpaceXS)
	t.HeaderLogo = lipgloss.NewStyle().Bold(true).Foreground(p.Primary).Background(p.Surface)
	t.HeaderInstance = lipgloss.NewStyle().Bold(true).Foreground(p.Text).Background(p.Surface)
	t.HeaderMeta = lipgloss.NewStyle().Foreground(p.TextMuted).Background(p.Surface)
	t.HeaderRule = lipgloss.NewStyle().Foreground(p.BorderSubtle)

	t.FooterBar = lipgloss.NewStyle().
		Background(p.Surface).
		Foreground(p.TextMuted).
		Padding(0, SpaceXS)
	t.FooterKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Secondary).
		Background(p.SurfaceRaised).
		Padding(0, SpaceXS)
	t.FooterDesc = lipgloss.NewStyle().Foreground(p.TextMuted).Background(p.Surface)
	t.FooterSep = lipgloss.NewStyle().Foreground(p.BorderSubtle).Background(p.Surface)

	t.NavTitle = lipgloss.NewStyle().Bold(true).Foreground(p.TextSubtle).Padding(0, SpaceXS)
	t.NavItem = lipgloss.NewStyle().Foreground(p.TextMuted).Padding(0, SpaceXS)
	t.NavItemActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.SelectionText).
		Background(p.Selection).
		Padding(0, SpaceXS)
	t.NavItemBlurred = lipgloss.NewStyle().
		Foreground(p.Text).
		Background(p.SelectionDim).
		Padding(0, SpaceXS)
	t.NavCount = lipgloss.NewStyle().
		Foreground(p.TextSubtle).
		Background(p.SurfaceRaised).
		Padding(0, SpaceXS)
	t.TabActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Primary).
		Underline(true).
		Padding(0, SpaceXS)
	t.TabInactive = lipgloss.NewStyle().Foreground(p.TextMuted).Padding(0, SpaceXS)

	t.TableHeader = lipgloss.NewStyle().Bold(true).Foreground(p.TextSubtle)
	t.TableRow = base
	t.TableRowAlt = lipgloss.NewStyle().Foreground(p.Text).Background(p.Surface)
	t.TableRowActive = lipgloss.NewStyle().
		Foreground(p.SelectionText).
		Background(p.Selection).
		Bold(true)
	t.TableRowInactive = lipgloss.NewStyle().
		Foreground(p.Text).
		Background(p.SelectionDim)
	t.TableCellMuted = lipgloss.NewStyle().Foreground(p.TextMuted)
	t.TableMarker = lipgloss.NewStyle().Foreground(p.Primary)

	t.Modal = lipgloss.NewStyle().
		Border(border).
		BorderForeground(p.BorderFocused).
		Background(p.Overlay).
		Padding(SpaceXS, SpaceSM)
	t.ModalDanger = t.Modal.BorderForeground(p.Error)
	t.ModalTitle = lipgloss.NewStyle().Bold(true).Foreground(p.Text).Background(p.Overlay)
	t.ModalBody = lipgloss.NewStyle().Foreground(p.TextMuted).Background(p.Overlay)
	t.ModalHint = lipgloss.NewStyle().Foreground(p.TextSubtle).Background(p.Overlay)
	// The backdrop dims the underlying screen by restyling it rather than by
	// using real transparency, which terminals do not reliably support.
	t.Backdrop = lipgloss.NewStyle().Foreground(p.TextSubtle)
	t.ButtonPrime = lipgloss.NewStyle().
		Bold(true).Foreground(p.TextInvert).Background(p.Primary).Padding(0, SpaceSM)
	t.ButtonDanger = lipgloss.NewStyle().
		Bold(true).Foreground(p.TextInvert).Background(p.Error).Padding(0, SpaceSM)
	t.ButtonGhost = lipgloss.NewStyle().
		Foreground(p.TextMuted).Background(p.Overlay).Padding(0, SpaceSM)

	toast := lipgloss.NewStyle().Border(border).Padding(0, SpaceXS)
	t.ToastSuccess = toast.BorderForeground(p.Success).Foreground(p.Success)
	t.ToastInfo = toast.BorderForeground(p.Info).Foreground(p.Info)
	t.ToastWarning = toast.BorderForeground(p.Warning).Foreground(p.Warning)
	t.ToastError = toast.BorderForeground(p.Error).Foreground(p.Error)

	t.EmptyTitle = lipgloss.NewStyle().Bold(true).Foreground(p.Text)
	t.EmptyBody = lipgloss.NewStyle().Foreground(p.TextMuted)
	t.EmptyHint = lipgloss.NewStyle().Foreground(p.TextSubtle)
	t.Skeleton = lipgloss.NewStyle().Foreground(p.BorderSubtle)
	t.Spinner = lipgloss.NewStyle().Foreground(p.Primary)

	t.PaletteInput = lipgloss.NewStyle().Foreground(p.Text).Background(p.Overlay)
	t.PaletteItem = lipgloss.NewStyle().Foreground(p.Text).Background(p.Overlay)
	t.PaletteActive = lipgloss.NewStyle().
		Bold(true).Foreground(p.SelectionText).Background(p.Selection)
	t.PaletteShortcut = lipgloss.NewStyle().Foreground(p.Secondary).Background(p.Overlay)
	t.PaletteDisabled = lipgloss.NewStyle().Foreground(p.TextSubtle).Background(p.Overlay)
	t.PaletteReason = lipgloss.NewStyle().Italic(true).Foreground(p.TextSubtle).Background(p.Overlay)
	t.PaletteGroup = lipgloss.NewStyle().Bold(true).Foreground(p.TextSubtle).Background(p.Overlay)

	t.FilterPrompt = lipgloss.NewStyle().Bold(true).Foreground(p.Primary)
	t.FilterText = base
	t.FilterHint = lipgloss.NewStyle().Foreground(p.TextSubtle)

	t.LogTimestamp = lipgloss.NewStyle().Foreground(p.TextSubtle)
	t.LogText = base
	t.LogError = lipgloss.NewStyle().Foreground(p.Error)
	t.LogWarn = lipgloss.NewStyle().Foreground(p.Warning)
	t.LogInfo = lipgloss.NewStyle().Foreground(p.Info)
	t.LogDebug = lipgloss.NewStyle().Foreground(p.TextMuted)
	t.LogMatch = lipgloss.NewStyle().Bold(true).
		Foreground(p.TextInvert).Background(p.Warning)

	badge := lipgloss.NewStyle().Padding(0, SpaceXS)
	t.badgeNeutral = badge.Foreground(p.TextMuted).Background(p.SurfaceRaised)
	t.badgeSuccess = badge.Foreground(p.TextInvert).Background(p.Success)
	t.badgeWarning = badge.Foreground(p.TextInvert).Background(p.Warning)
	t.badgeError = badge.Foreground(p.TextInvert).Background(p.Error)
	t.badgeInfo = badge.Foreground(p.TextInvert).Background(p.Info)

	t.statusFg = map[domain.ResourceStatus]lipgloss.Style{
		domain.StatusRunning:    t.Positive,
		domain.StatusDegraded:   t.Attention,
		domain.StatusFailed:     t.Danger,
		domain.StatusStopped:    t.Muted,
		domain.StatusPaused:     t.Muted,
		domain.StatusBuilding:   t.Attention,
		domain.StatusDeploying:  t.Accent,
		domain.StatusStarting:   t.Note,
		domain.StatusStopping:   t.Note,
		domain.StatusRestarting: t.Note,
		domain.StatusUnknown:    t.Subtle,
	}

	return t
}

// StatusStyle returns the foreground style for a resource state.
func (t *Theme) StatusStyle(st domain.Status) lipgloss.Style {
	if s, ok := t.statusFg[st.State]; ok {
		return s
	}
	return t.Subtle
}

// StatusText renders "<symbol> <label>", which is how status appears
// everywhere. The label is always present, so the meaning survives a
// monochrome terminal or a colour-blind reader.
func (t *Theme) StatusText(st domain.Status) string {
	return t.StatusStyle(st).Render(t.Sym.StatusSymbol(st) + " " + st.Label())
}

// DeploymentStatusStyle returns the foreground style for a deployment state.
func (t *Theme) DeploymentStatusStyle(st domain.DeploymentStatus) lipgloss.Style {
	return t.StatusStyle(domain.Status{State: st.AsResourceStatus()})
}

// DeploymentStatusText renders a deployment status with its symbol and label.
func (t *Theme) DeploymentStatusText(st domain.DeploymentStatus) string {
	return t.DeploymentStatusStyle(st).Render(t.Sym.DeploymentSymbol(st) + " " + st.Label())
}

// BadgeKind selects a badge's semantic colour.
type BadgeKind int

// Badge kinds.
const (
	BadgeNeutral BadgeKind = iota
	BadgeSuccess
	BadgeWarning
	BadgeError
	BadgeInfo
)

// Badge renders a small filled label.
func (t *Theme) Badge(kind BadgeKind, text string) string {
	switch kind {
	case BadgeSuccess:
		return t.badgeSuccess.Render(text)
	case BadgeWarning:
		return t.badgeWarning.Render(text)
	case BadgeError:
		return t.badgeError.Render(text)
	case BadgeInfo:
		return t.badgeInfo.Render(text)
	default:
		return t.badgeNeutral.Render(text)
	}
}

func resolvePalette(opts Options) Palette {
	switch opts.PaletteName {
	case "dracula":
		return DraculaPalette
	case "catppuccin":
		return CatppuccinPalette
	case "nord":
		return NordPalette
	case "gruvbox":
		return GruvboxPalette
	case "tokyo-night":
		return TokyoNightPalette
	default:
		if opts.Mode == ModeLight {
			return LightPalette
		}
		return DarkPalette
	}
}
