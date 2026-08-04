package theme

import "testing"

func defaultOpts() LayoutOptions {
	return LayoutOptions{ShowHeader: true, ShowFooter: true}
}

func TestComputeLayoutBreakpoints(t *testing.T) {
	tests := []struct {
		name        string
		w, h        int
		want        Breakpoint
		wantSidebar bool
		wantPreview bool
	}{
		{"ultrawide", 200, 60, BreakpointWide, true, true},
		{"wide", 130, 40, BreakpointWide, true, false},
		{"wide boundary", WideMinWidth, 40, BreakpointWide, true, false},
		{"standard", 100, 40, BreakpointStandard, false, false},
		{"standard boundary", StandardMinWidth, 40, BreakpointStandard, false, false},
		{"compact by width", 70, 40, BreakpointCompact, false, false},
		// A short terminal drops to compact even when it is wide: with few
		// rows, chrome costs more than it gives.
		{"compact by height", 160, 22, BreakpointCompact, false, false},
		{"too narrow", 50, 40, BreakpointTooSmall, false, false},
		{"too short", 120, 10, BreakpointTooSmall, false, false},
		{"minimum usable", MinUsableWidth, MinUsableHeight, BreakpointCompact, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := ComputeLayout(tt.w, tt.h, defaultOpts())
			if l.Break != tt.want {
				t.Errorf("breakpoint = %v, want %v", l.Break, tt.want)
			}
			if l.ShowSidebar != tt.wantSidebar {
				t.Errorf("ShowSidebar = %v, want %v", l.ShowSidebar, tt.wantSidebar)
			}
			if l.ShowPreview != tt.wantPreview {
				t.Errorf("ShowPreview = %v, want %v", l.ShowPreview, tt.wantPreview)
			}
		})
	}
}

// TestComputeLayoutPartitionsWidthExactly is the guard against off-by-one
// layout bugs: the panels must add up to the terminal width at every size, or
// the frame wraps and the whole screen shears.
func TestComputeLayoutPartitionsWidthExactly(t *testing.T) {
	for w := MinUsableWidth; w <= 400; w++ {
		for _, h := range []int{MinUsableHeight, 24, 25, 40, 60} {
			l := ComputeLayout(w, h, defaultOpts())
			if !l.Usable() {
				continue
			}
			if got := l.SidebarWidth + l.PreviewWidth + l.ContentWidth; got != w {
				t.Fatalf("at %dx%d panels sum to %d, want %d", w, h, got, w)
			}
			if l.ContentWidth < 1 {
				t.Fatalf("at %dx%d content width collapsed to %d", w, h, l.ContentWidth)
			}
			if l.ContentHeight < 1 {
				t.Fatalf("at %dx%d content height collapsed to %d", w, h, l.ContentHeight)
			}
			if l.ContentHeight > h {
				t.Fatalf("at %dx%d content height %d exceeds the terminal", w, h, l.ContentHeight)
			}
		}
	}
}

func TestComputeLayoutHonoursChromePreferences(t *testing.T) {
	with := ComputeLayout(160, 50, LayoutOptions{ShowHeader: true, ShowFooter: true})
	without := ComputeLayout(160, 50, LayoutOptions{})

	if without.HeaderHeight != 0 || without.FooterHeight != 0 {
		t.Error("disabled chrome should occupy no rows")
	}
	if without.ContentHeight <= with.ContentHeight {
		t.Error("hiding chrome should give the content more rows")
	}
}

func TestForceCompactOverridesSize(t *testing.T) {
	opts := defaultOpts()
	opts.ForceCompact = true

	l := ComputeLayout(200, 60, opts)
	if l.Break != BreakpointCompact {
		t.Fatalf("breakpoint = %v, want compact", l.Break)
	}
	if l.ShowSidebar || l.ShowPreview {
		t.Error("compact mode must not draw the sidebar or preview")
	}
	if !l.IsCompact() {
		t.Error("IsCompact should agree with the breakpoint")
	}
}

func TestForceRoomyKeepsChromeOnShortTerminals(t *testing.T) {
	opts := defaultOpts()
	opts.ForceRoomy = true

	if l := ComputeLayout(160, 22, opts); l.Break != BreakpointWide {
		t.Fatalf("breakpoint = %v, want wide when compact mode is forced off", l.Break)
	}
}

func TestTooSmallLayoutIsInert(t *testing.T) {
	l := ComputeLayout(20, 5, defaultOpts())
	if l.Usable() {
		t.Fatal("a 20x5 terminal should not be usable")
	}
	// Nothing should be laid out, so the caller renders the "resize me" screen
	// rather than a sheared frame.
	if l.ContentWidth != 0 || l.ContentHeight != 0 {
		t.Errorf("unusable layout should have no content area, got %+v", l)
	}
}
