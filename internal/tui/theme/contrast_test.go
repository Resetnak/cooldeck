package theme

import (
	"image/color"
	"math"
	"os"
	"sort"
	"testing"
)

// minContrast is WCAG AA for normal-size text. Terminal cells are normal-size
// text; there is no "large text" exemption to lean on here.
const minContrast = 4.5

// relativeLuminance implements the WCAG 2.1 definition for sRGB.
func relativeLuminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	lin := func(v uint32) float64 {
		s := float64(v) / 65535.0
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

func contrastRatio(fg, bg color.Color) float64 {
	l1, l2 := relativeLuminance(fg), relativeLuminance(bg)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

type pair struct {
	role string
	fg   func(Palette) color.Color
	bg   func(Palette) color.Color
}

// pairs lists every foreground/background combination the UI actually renders,
// read off the style constructors in New. A style with no explicit Background
// sits on the terminal background, which is p.Background.
//
// Keep this in step with New: a style added there with a new colour pairing is
// unmeasured until it appears here.
var pairs = []pair{
	// Typography roles on the bare background: Text, Muted, Subtle, plus the
	// semantic foregrounds used by status text, toasts and log levels.
	{"Text/Background", func(p Palette) color.Color { return p.Text }, func(p Palette) color.Color { return p.Background }},
	{"TextMuted/Background", func(p Palette) color.Color { return p.TextMuted }, func(p Palette) color.Color { return p.Background }},
	{"TextSubtle/Background", func(p Palette) color.Color { return p.TextSubtle }, func(p Palette) color.Color { return p.Background }},
	{"Primary/Background", func(p Palette) color.Color { return p.Primary }, func(p Palette) color.Color { return p.Background }},
	{"Success/Background", func(p Palette) color.Color { return p.Success }, func(p Palette) color.Color { return p.Background }},
	{"Warning/Background", func(p Palette) color.Color { return p.Warning }, func(p Palette) color.Color { return p.Background }},
	{"Error/Background", func(p Palette) color.Color { return p.Error }, func(p Palette) color.Color { return p.Background }},
	{"Info/Background", func(p Palette) color.Color { return p.Info }, func(p Palette) color.Color { return p.Background }},

	// Header, footer and alternating table rows sit on Surface.
	{"Text/Surface", func(p Palette) color.Color { return p.Text }, func(p Palette) color.Color { return p.Surface }},
	{"TextMuted/Surface", func(p Palette) color.Color { return p.TextMuted }, func(p Palette) color.Color { return p.Surface }},
	{"Primary/Surface", func(p Palette) color.Color { return p.Primary }, func(p Palette) color.Color { return p.Surface }},

	// Footer key chips, nav counts and neutral badges sit on SurfaceRaised,
	// which is the tightest surface in every palette.
	{"Secondary/SurfaceRaised", func(p Palette) color.Color { return p.Secondary }, func(p Palette) color.Color { return p.SurfaceRaised }},
	{"TextSubtle/SurfaceRaised", func(p Palette) color.Color { return p.TextSubtle }, func(p Palette) color.Color { return p.SurfaceRaised }},
	{"TextMuted/SurfaceRaised", func(p Palette) color.Color { return p.TextMuted }, func(p Palette) color.Color { return p.SurfaceRaised }},

	// Modals and the command palette sit on Overlay.
	{"Text/Overlay", func(p Palette) color.Color { return p.Text }, func(p Palette) color.Color { return p.Overlay }},
	{"TextMuted/Overlay", func(p Palette) color.Color { return p.TextMuted }, func(p Palette) color.Color { return p.Overlay }},
	{"TextSubtle/Overlay", func(p Palette) color.Color { return p.TextSubtle }, func(p Palette) color.Color { return p.Overlay }},
	{"Secondary/Overlay", func(p Palette) color.Color { return p.Secondary }, func(p Palette) color.Color { return p.Overlay }},

	// Selected and blurred rows.
	{"SelectionText/Selection", func(p Palette) color.Color { return p.SelectionText }, func(p Palette) color.Color { return p.Selection }},
	{"Text/SelectionDim", func(p Palette) color.Color { return p.Text }, func(p Palette) color.Color { return p.SelectionDim }},

	// Filled badges and buttons print TextInvert on a semantic colour.
	{"TextInvert/Primary", func(p Palette) color.Color { return p.TextInvert }, func(p Palette) color.Color { return p.Primary }},
	{"TextInvert/Success", func(p Palette) color.Color { return p.TextInvert }, func(p Palette) color.Color { return p.Success }},
	{"TextInvert/Warning", func(p Palette) color.Color { return p.TextInvert }, func(p Palette) color.Color { return p.Warning }},
	{"TextInvert/Error", func(p Palette) color.Color { return p.TextInvert }, func(p Palette) color.Color { return p.Error }},
	{"TextInvert/Info", func(p Palette) color.Color { return p.TextInvert }, func(p Palette) color.Color { return p.Info }},
}

var allPalettes = []struct {
	name string
	p    Palette
}{
	{"dark", DarkPalette},
	{"light", LightPalette},
	{"dracula", DraculaPalette},
	{"catppuccin", CatppuccinPalette},
	{"nord", NordPalette},
	{"gruvbox", GruvboxPalette},
	{"tokyo-night", TokyoNightPalette},
}

// TestPaletteContrast holds every shipped palette to WCAG AA. The named
// palettes reproduce published schemes, but several of those put secondary
// text below 3:1 on their own surfaces, and TextSubtle here carries real data
// (log timestamps, list counters, the disabled-capability marker) rather than
// decoration. Where a value had to move, the palette's doc comment records the
// upstream original and the ratio that forced the change.
func TestPaletteContrast(t *testing.T) {
	for _, pal := range allPalettes {
		for _, pr := range pairs {
			got := contrastRatio(pr.fg(pal.p), pr.bg(pal.p))
			if got < minContrast {
				t.Errorf("%s: %s is %.2f:1, want >= %.1f:1", pal.name, pr.role, got, minContrast)
			}
		}
	}
}

// TestContrastReport dumps every ratio, worst first. It is a reporting aid for
// tuning a palette, not a gate: run it with CONTRAST_REPORT=1.
func TestContrastReport(t *testing.T) {
	if os.Getenv("CONTRAST_REPORT") == "" {
		t.Skip("set CONTRAST_REPORT=1 to dump the full table")
	}
	for _, pal := range allPalettes {
		type row struct {
			role  string
			ratio float64
		}
		rows := make([]row, 0, len(pairs))
		for _, pr := range pairs {
			rows = append(rows, row{pr.role, contrastRatio(pr.fg(pal.p), pr.bg(pal.p))})
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].ratio < rows[j].ratio })
		t.Logf("--- %s ---", pal.name)
		for _, r := range rows {
			t.Logf("  %5.2f  %s", r.ratio, r.role)
		}
	}
}
