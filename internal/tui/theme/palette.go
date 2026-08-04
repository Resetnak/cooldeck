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

// Spacing is the spacing scale. Every gap in the UI is one of these values, so
// rhythm stays consistent without magic numbers scattered through views.
const (
	SpaceNone = 0
	SpaceXS   = 1
	SpaceSM   = 2
	SpaceMD   = 3
	SpaceLG   = 4
)
