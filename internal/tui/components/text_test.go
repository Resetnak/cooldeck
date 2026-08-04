package components

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestWidthIsUnicodeAndANSIAware(t *testing.T) {
	tests := map[string]int{
		"hello":  5,
		"":       0,
		"příliš": 6, // combining-free Latin
		"日本語":    6, // double-width
		"é":     1, // combining acute
		lipgloss.NewStyle().Bold(true).Render("hi"): 2,
	}
	for in, want := range tests {
		if got := Width(in); got != want {
			t.Errorf("Width(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		in       string
		width    int
		ellipsis string
		want     string
	}{
		{"hello world", 20, "…", "hello world"},
		{"hello world", 11, "…", "hello world"},
		{"hello world", 8, "…", "hello w…"},
		{"hello", 0, "…", ""},
		// No room for both content and marker: show what fits.
		{"hello", 1, "…", "h"},
		// A double-width glyph cannot be split, so the result may come in
		// under the budget rather than overrunning it.
		{"日本語テスト", 6, "…", "日本…"},
	}
	for _, tt := range tests {
		got := Truncate(tt.in, tt.width, tt.ellipsis)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.in, tt.width, got, tt.want)
		}
		if w := Width(got); w > tt.width {
			t.Errorf("Truncate(%q, %d) produced %d cells", tt.in, tt.width, w)
		}
	}
}

// TestFitAlwaysProducesExactWidth is the invariant the whole table layout
// rests on: if any cell is off by one, every column to its right shears.
func TestFitAlwaysProducesExactWidth(t *testing.T) {
	inputs := []string{
		"", "a", "hello world", "日本語テスト", "écombining",
		strings.Repeat("x", 200),
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ff0000")).Render("styled"),
		"emoji 👍 here",
	}
	for _, in := range inputs {
		for w := 1; w <= 24; w++ {
			if got := Width(Fit(in, w, "…")); got != w {
				t.Errorf("Fit(%q, %d) rendered %d cells", in, w, got)
			}
			if got := Width(Pad(in, w)); got != w {
				t.Errorf("Pad(%q, %d) rendered %d cells", in, w, got)
			}
			if got := Width(PadLeft(in, w)); got != w {
				t.Errorf("PadLeft(%q, %d) rendered %d cells", in, w, got)
			}
		}
	}
}

func TestFitBlockNormalisesGeometry(t *testing.T) {
	block := "one\ntwo\nthree"

	got := FitBlock(block, 10, 5)
	lines := Lines(got)
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	for i, l := range lines {
		if w := Width(l); w != 10 {
			t.Errorf("line %d is %d cells wide, want 10", i, w)
		}
	}

	// Overflow must be cut, not allowed to push the layout down.
	if got := FitBlock(block, 10, 2); len(Lines(got)) != 2 {
		t.Errorf("got %d lines, want 2", len(Lines(got)))
	}
	if FitBlock(block, 10, 0) != "" {
		t.Error("zero height should render nothing")
	}
}

func TestOrDash(t *testing.T) {
	if OrDash("") != Dash || OrDash("   ") != Dash {
		t.Error("blank values should render as a dash")
	}
	if OrDash("value") != "value" {
		t.Error("non-blank values should pass through")
	}
}

func TestClampOffsetKeepsSelectionVisible(t *testing.T) {
	tests := []struct {
		name                            string
		offset, selected, rows, visible int
		want                            int
	}{
		{"already visible", 0, 3, 100, 10, 0},
		{"scroll down by one", 0, 10, 100, 10, 1},
		{"scroll up", 20, 5, 100, 10, 5},
		{"jump to end", 0, 99, 100, 10, 90},
		{"clamp past end", 95, 99, 100, 10, 90},
		{"fewer rows than viewport", 0, 2, 3, 10, 0},
		{"no rows", 5, 0, 0, 10, 0},
		{"no selection", 5, -1, 100, 10, 0},
		{"no viewport", 5, 3, 100, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampOffset(tt.offset, tt.selected, tt.rows, tt.visible)
			if got != tt.want {
				t.Errorf("ClampOffset(%d,%d,%d,%d) = %d, want %d",
					tt.offset, tt.selected, tt.rows, tt.visible, got, tt.want)
			}
			if tt.visible > 0 && tt.rows > 0 && tt.selected >= 0 {
				if tt.selected < got || tt.selected >= got+tt.visible {
					t.Errorf("selection %d is outside the window [%d,%d)",
						tt.selected, got, got+tt.visible)
				}
			}
		})
	}
}
