// Package theme holds cooldeck's design tokens and the pre-built Lip Gloss
// styles derived from them. Views never construct styles of their own, which
// is what keeps spacing, borders and colour meaning consistent across screens
// and makes a theme switch a single assignment.
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette is the full set of colour tokens. Colours carry meaning rather than
// decoration: green is always "healthy", red is always "failed or
// destructive", and no state is ever communicated by colour alone.
type Palette struct {
	// Surfaces, from furthest back to closest.
	Background    color.Color
	Surface       color.Color
	SurfaceRaised color.Color
	Overlay       color.Color

	// Structure.
	Border        color.Color
	BorderFocused color.Color
	BorderSubtle  color.Color

	// Typography.
	Text       color.Color
	TextMuted  color.Color
	TextSubtle color.Color
	TextInvert color.Color

	// Accents and semantics.
	Primary   color.Color
	Secondary color.Color
	Success   color.Color
	Warning   color.Color
	Error     color.Color
	Info      color.Color

	// Selection.
	Selection     color.Color
	SelectionText color.Color
	// SelectionDim is the selection colour used when the panel is not focused,
	// so the user can always tell which panel keystrokes will go to.
	SelectionDim color.Color
}

// DarkPalette is tuned for dark terminals. Contrast ratios for text on
// Background are kept above 7:1, and above 4.5:1 for muted text, so the UI
// stays readable on a dimmed laptop screen over SSH.
var DarkPalette = Palette{
	Background:    lipgloss.Color("#0B0E14"),
	Surface:       lipgloss.Color("#11161F"),
	SurfaceRaised: lipgloss.Color("#1A212D"),
	Overlay:       lipgloss.Color("#151A24"),

	Border:        lipgloss.Color("#2A3341"),
	BorderFocused: lipgloss.Color("#7C6CF5"),
	BorderSubtle:  lipgloss.Color("#1C2430"),

	Text:       lipgloss.Color("#DCE3EE"),
	TextMuted:  lipgloss.Color("#8B97AC"),
	TextSubtle: lipgloss.Color("#5A6579"),
	TextInvert: lipgloss.Color("#0B0E14"),

	Primary:   lipgloss.Color("#A78BFA"),
	Secondary: lipgloss.Color("#22D3EE"),
	Success:   lipgloss.Color("#4ADE80"),
	Warning:   lipgloss.Color("#FBBF24"),
	Error:     lipgloss.Color("#F87171"),
	Info:      lipgloss.Color("#60A5FA"),

	Selection:     lipgloss.Color("#243044"),
	SelectionText: lipgloss.Color("#FFFFFF"),
	SelectionDim:  lipgloss.Color("#181F2B"),
}

// LightPalette is tuned for light terminals. It is not a naive inversion:
// accent colours are darkened so they stay legible on white, and the
// selection is a tinted surface rather than a dark block.
var LightPalette = Palette{
	Background:    lipgloss.Color("#FFFFFF"),
	Surface:       lipgloss.Color("#F6F7F9"),
	SurfaceRaised: lipgloss.Color("#ECEEF2"),
	Overlay:       lipgloss.Color("#FFFFFF"),

	Border:        lipgloss.Color("#D2D7E0"),
	BorderFocused: lipgloss.Color("#6D4AFF"),
	BorderSubtle:  lipgloss.Color("#E6E9EF"),

	Text:       lipgloss.Color("#151A22"),
	TextMuted:  lipgloss.Color("#5B6472"),
	TextSubtle: lipgloss.Color("#8C95A4"),
	TextInvert: lipgloss.Color("#FFFFFF"),

	Primary:   lipgloss.Color("#6D4AFF"),
	Secondary: lipgloss.Color("#0E7490"),
	Success:   lipgloss.Color("#15803D"),
	Warning:   lipgloss.Color("#B45309"),
	Error:     lipgloss.Color("#DC2626"),
	Info:      lipgloss.Color("#1D4ED8"),

	Selection:     lipgloss.Color("#E4E1FF"),
	SelectionText: lipgloss.Color("#1B1440"),
	SelectionDim:  lipgloss.Color("#EFF1F5"),
}

// DraculaPalette implements the Dracula colour scheme.
// Reference: https://draculatheme.com/contribute
var DraculaPalette = Palette{
	Background:    lipgloss.Color("#282A36"),
	Surface:       lipgloss.Color("#2D303D"),
	SurfaceRaised: lipgloss.Color("#343746"),
	Overlay:       lipgloss.Color("#2D303D"),

	Border:        lipgloss.Color("#44475A"),
	BorderFocused: lipgloss.Color("#BD93F9"),
	BorderSubtle:  lipgloss.Color("#383A4A"),

	Text:       lipgloss.Color("#F8F8F2"),
	TextMuted:  lipgloss.Color("#BFBFBF"),
	TextSubtle: lipgloss.Color("#6272A4"),
	TextInvert: lipgloss.Color("#282A36"),

	Primary:   lipgloss.Color("#BD93F9"),
	Secondary: lipgloss.Color("#8BE9FD"),
	Success:   lipgloss.Color("#50FA7B"),
	Warning:   lipgloss.Color("#F1FA8C"),
	Error:     lipgloss.Color("#FF5555"),
	Info:      lipgloss.Color("#8BE9FD"),

	Selection:     lipgloss.Color("#44475A"),
	SelectionText: lipgloss.Color("#F8F8F2"),
	SelectionDim:  lipgloss.Color("#363948"),
}

// CatppuccinPalette implements the Catppuccin Mocha flavour.
// Reference: https://github.com/catppuccin/catppuccin
var CatppuccinPalette = Palette{
	Background:    lipgloss.Color("#1E1E2E"),
	Surface:       lipgloss.Color("#232334"),
	SurfaceRaised: lipgloss.Color("#313244"),
	Overlay:       lipgloss.Color("#232334"),

	Border:        lipgloss.Color("#45475A"),
	BorderFocused: lipgloss.Color("#CBA6F7"),
	BorderSubtle:  lipgloss.Color("#313244"),

	Text:       lipgloss.Color("#CDD6F4"),
	TextMuted:  lipgloss.Color("#A6ADC8"),
	TextSubtle: lipgloss.Color("#6C7086"),
	TextInvert: lipgloss.Color("#1E1E2E"),

	Primary:   lipgloss.Color("#CBA6F7"),
	Secondary: lipgloss.Color("#89DCEB"),
	Success:   lipgloss.Color("#A6E3A1"),
	Warning:   lipgloss.Color("#F9E2AF"),
	Error:     lipgloss.Color("#F38BA8"),
	Info:      lipgloss.Color("#89B4FA"),

	Selection:     lipgloss.Color("#45475A"),
	SelectionText: lipgloss.Color("#CDD6F4"),
	SelectionDim:  lipgloss.Color("#2A2A3C"),
}

// NordPalette implements the Nord colour scheme.
// Reference: https://www.nordtheme.com/docs/colors-and-palettes
var NordPalette = Palette{
	Background:    lipgloss.Color("#2E3440"),
	Surface:       lipgloss.Color("#333946"),
	SurfaceRaised: lipgloss.Color("#3B4252"),
	Overlay:       lipgloss.Color("#333946"),

	Border:        lipgloss.Color("#4C566A"),
	BorderFocused: lipgloss.Color("#88C0D0"),
	BorderSubtle:  lipgloss.Color("#434C5E"),

	Text:       lipgloss.Color("#ECEFF4"),
	TextMuted:  lipgloss.Color("#D8DEE9"),
	TextSubtle: lipgloss.Color("#737D8C"),
	TextInvert: lipgloss.Color("#2E3440"),

	Primary:   lipgloss.Color("#88C0D0"),
	Secondary: lipgloss.Color("#81A1C1"),
	Success:   lipgloss.Color("#A3BE8C"),
	Warning:   lipgloss.Color("#EBCB8B"),
	Error:     lipgloss.Color("#BF616A"),
	Info:      lipgloss.Color("#5E81AC"),

	Selection:     lipgloss.Color("#434C5E"),
	SelectionText: lipgloss.Color("#ECEFF4"),
	SelectionDim:  lipgloss.Color("#3B4252"),
}

// GruvboxPalette implements the Gruvbox Dark colour scheme.
// Reference: https://github.com/morhetz/gruvbox
var GruvboxPalette = Palette{
	Background:    lipgloss.Color("#282828"),
	Surface:       lipgloss.Color("#2D2D2D"),
	SurfaceRaised: lipgloss.Color("#3C3836"),
	Overlay:       lipgloss.Color("#2D2D2D"),

	Border:        lipgloss.Color("#504945"),
	BorderFocused: lipgloss.Color("#D79921"),
	BorderSubtle:  lipgloss.Color("#3C3836"),

	Text:       lipgloss.Color("#EBDBB2"),
	TextMuted:  lipgloss.Color("#BDAE93"),
	TextSubtle: lipgloss.Color("#7C6F64"),
	TextInvert: lipgloss.Color("#282828"),

	Primary:   lipgloss.Color("#D79921"),
	Secondary: lipgloss.Color("#458588"),
	Success:   lipgloss.Color("#B8BB26"),
	Warning:   lipgloss.Color("#FE8019"),
	Error:     lipgloss.Color("#FB4934"),
	Info:      lipgloss.Color("#83A598"),

	Selection:     lipgloss.Color("#504945"),
	SelectionText: lipgloss.Color("#EBDBB2"),
	SelectionDim:  lipgloss.Color("#3C3836"),
}

// TokyoNightPalette implements the Tokyo Night colour scheme.
// Reference: https://github.com/enkia/tokyo-night-vscode-theme
var TokyoNightPalette = Palette{
	Background:    lipgloss.Color("#1A1B26"),
	Surface:       lipgloss.Color("#1E2030"),
	SurfaceRaised: lipgloss.Color("#24283B"),
	Overlay:       lipgloss.Color("#1E2030"),

	Border:        lipgloss.Color("#3B4261"),
	BorderFocused: lipgloss.Color("#7AA2F7"),
	BorderSubtle:  lipgloss.Color("#292E42"),

	Text:       lipgloss.Color("#C0CAF5"),
	TextMuted:  lipgloss.Color("#A9B1D6"),
	TextSubtle: lipgloss.Color("#565F89"),
	TextInvert: lipgloss.Color("#1A1B26"),

	Primary:   lipgloss.Color("#7AA2F7"),
	Secondary: lipgloss.Color("#7DCFFF"),
	Success:   lipgloss.Color("#9ECE6A"),
	Warning:   lipgloss.Color("#E0AF68"),
	Error:     lipgloss.Color("#F7768E"),
	Info:      lipgloss.Color("#2AC3DE"),

	Selection:     lipgloss.Color("#33467C"),
	SelectionText: lipgloss.Color("#C0CAF5"),
	SelectionDim:  lipgloss.Color("#292E42"),
}

// Spacing is the spacing scale. Every gap in the UI is one of these values, so
// rhythm stays consistent without magic numbers scattered through views.
const (
	SpaceNone = 0
	SpaceXS   = 1
	SpaceSM   = 2
	SpaceMD   = 3
	SpaceLG   = 4
)
