package theme

// Breakpoint is the responsive tier the current terminal falls into.
type Breakpoint int

// Responsive tiers.
const (
	// BreakpointTooSmall means the terminal cannot host a usable layout.
	BreakpointTooSmall Breakpoint = iota
	// BreakpointCompact is a single column with minimal chrome.
	BreakpointCompact
	// BreakpointStandard adds top tabs and more columns.
	BreakpointStandard
	// BreakpointWide adds the sidebar and a side preview panel.
	BreakpointWide
)

// Breakpoint thresholds, in terminal cells.
const (
	WideMinWidth     = 120
	StandardMinWidth = 80
	MinUsableWidth   = 60
	MinUsableHeight  = 18
	// CompactMaxHeight is the height below which even a wide terminal drops to
	// the compact layout: with few rows, chrome costs more than it gives.
	CompactMaxHeight = 24
	// PreviewMinWidth is the width at which a side preview panel earns its space.
	PreviewMinWidth = 150
)

// Layout is the resolved geometry for one frame. It is recomputed on resize,
// never during rendering, so every component in a frame agrees on the sizes.
type Layout struct {
	Width  int
	Height int
	Break  Breakpoint

	// ShowSidebar reports whether the vertical navigation is drawn. When
	// false, navigation moves to top tabs and the command palette.
	ShowSidebar bool
	// ShowPreview reports whether a detail preview is drawn beside the table.
	ShowPreview bool
	// ShowHeader and ShowFooter honour both the config and the available space.
	ShowHeader bool
	ShowFooter bool

	SidebarWidth  int
	PreviewWidth  int
	ContentWidth  int
	ContentHeight int
	HeaderHeight  int
	FooterHeight  int
}

// LayoutOptions carries the user preferences that influence geometry.
type LayoutOptions struct {
	ShowHeader bool
	ShowFooter bool
	// ForceCompact pins the compact layout regardless of the terminal size.
	ForceCompact bool
	// ForceRoomy prevents the automatic drop to compact on short terminals.
	ForceRoomy bool
}

// ComputeLayout resolves the geometry for a terminal of the given size.
func ComputeLayout(width, height int, opts LayoutOptions) Layout {
	l := Layout{Width: width, Height: height}

	if width < MinUsableWidth || height < MinUsableHeight {
		l.Break = BreakpointTooSmall
		return l
	}

	switch {
	case opts.ForceCompact:
		l.Break = BreakpointCompact
	case width >= WideMinWidth && (height > CompactMaxHeight || opts.ForceRoomy):
		l.Break = BreakpointWide
	case width >= StandardMinWidth && (height > CompactMaxHeight || opts.ForceRoomy):
		l.Break = BreakpointStandard
	default:
		l.Break = BreakpointCompact
	}

	l.ShowHeader = opts.ShowHeader
	l.ShowFooter = opts.ShowFooter
	if l.ShowHeader {
		// One line of content plus the hairline rule beneath it.
		l.HeaderHeight = 2
	}
	if l.ShowFooter {
		// Hairline rule plus the key-hint bar.
		l.FooterHeight = 2
	}

	l.ShowSidebar = l.Break == BreakpointWide
	l.ShowPreview = l.Break == BreakpointWide && width >= PreviewMinWidth

	if l.ShowSidebar {
		l.SidebarWidth = sidebarWidthFor(width)
	}
	if l.ShowPreview {
		l.PreviewWidth = previewWidthFor(width - l.SidebarWidth)
	}

	l.ContentWidth = width - l.SidebarWidth - l.PreviewWidth
	l.ContentHeight = height - l.HeaderHeight - l.FooterHeight
	if l.Break != BreakpointWide {
		// Standard and compact show navigation as a single row of tabs.
		l.ContentHeight--
	}
	if l.ContentHeight < 1 {
		l.ContentHeight = 1
	}
	return l
}

// Usable reports whether the terminal is large enough to render the UI.
func (l Layout) Usable() bool { return l.Break != BreakpointTooSmall }

// IsCompact reports whether labels should be shortened and chrome minimised.
func (l Layout) IsCompact() bool { return l.Break == BreakpointCompact }

// sidebarWidthFor scales the sidebar with the terminal instead of pinning it,
// so an ultra-wide terminal does not end up with a stripe of empty navigation.
func sidebarWidthFor(width int) int {
	w := width / 6
	switch {
	case w < 18:
		return 18
	case w > 26:
		return 26
	default:
		return w
	}
}

func previewWidthFor(remaining int) int {
	w := remaining / 3
	switch {
	case w < 34:
		return 34
	case w > 56:
		return 56
	default:
		return w
	}
}
