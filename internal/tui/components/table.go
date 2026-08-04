package components

import (
	"strings"

	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// Column describes one table column.
type Column struct {
	// Title is the header label at full width.
	Title string
	// ShortTitle is used when the column is narrowed. Empty falls back to Title.
	ShortTitle string
	// MinWidth is the width below which the column is dropped entirely.
	MinWidth int
	// MaxWidth caps the column; 0 means uncapped.
	MaxWidth int
	// Flex is the share of the leftover width this column receives. A column
	// with Flex 0 is sized to MinWidth and never grows.
	Flex int
	// Priority orders columns for dropping when space runs out. Lower drops
	// first, so status and name (priority 0) survive longest.
	Priority int
	// Right right-aligns the cell content.
	Right bool
}

// Row is one table row. Cells are indexed in the same order as the columns.
type Row struct {
	// Cells holds the already-styled content for each column.
	Cells []string
	// Key identifies the row across refreshes so the selection can be
	// preserved when the underlying data is replaced.
	Key string
}

// Table renders a fixed-width table with a sticky header. It is a plain
// renderer: scroll position and selection are owned by the caller, which keeps
// selection stable across data refreshes.
type Table struct {
	Columns []Column
	Rows    []Row
	// Selected is the index of the highlighted row, -1 for none.
	Selected int
	// Offset is the index of the first visible row.
	Offset int
	// Focused controls whether the selection is drawn at full or reduced
	// emphasis, so the user can always tell where keystrokes will land.
	Focused bool
	// ZebraStripes tints alternate rows. Off by default; it competes with the
	// status colours for attention.
	ZebraStripes bool
}

// visibleColumns picks the columns that fit and assigns each a width.
// Columns with the highest numeric priority are dropped until the rest fit at MinWidth,
// then the leftover space is distributed by Flex.
func (t Table) visibleColumns(width int) ([]Column, []int) {
	const gap = 1

	cols := append([]Column(nil), t.Columns...)
	keep := make([]bool, len(cols))
	for i := range keep {
		keep[i] = true
	}

	fits := func() bool {
		total, n := 0, 0
		for i, c := range cols {
			if !keep[i] {
				continue
			}
			total += c.MinWidth
			n++
		}
		if n == 0 {
			return true
		}
		return total+gap*(n-1) <= width
	}

	for !fits() {
		// Status and name carry priority 0 and are therefore the last to go.
		victim, weakest := -1, -1
		for i, c := range cols {
			if keep[i] && c.Priority > weakest {
				victim, weakest = i, c.Priority
			}
		}
		if victim < 0 {
			break
		}
		keep[victim] = false
	}

	var out []Column
	var idx []int
	for i, c := range cols {
		if keep[i] {
			out = append(out, c)
			idx = append(idx, i)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}

	widths := make([]int, len(out))
	used := gap * (len(out) - 1)
	totalFlex := 0
	for i, c := range out {
		widths[i] = c.MinWidth
		used += c.MinWidth
		totalFlex += c.Flex
	}

	// Distribute the remaining width proportionally to Flex, respecting
	// MaxWidth so a single long column cannot swallow the table.
	spare := width - used
	for spare > 0 && totalFlex > 0 {
		distributed := 0
		for i, c := range out {
			if c.Flex == 0 {
				continue
			}
			share := spare * c.Flex / totalFlex
			if share == 0 {
				share = 1
			}
			if c.MaxWidth > 0 && widths[i]+share > c.MaxWidth {
				share = c.MaxWidth - widths[i]
			}
			if share <= 0 {
				continue
			}
			widths[i] += share
			distributed += share
			if distributed >= spare {
				break
			}
		}
		if distributed == 0 {
			break
		}
		spare -= distributed
	}

	// Store the resolved index mapping on the columns for the render pass.
	for i := range out {
		out[i].MinWidth = widths[i]
	}
	_ = idx
	return out, idx
}

// Render draws the table into width x height cells. The first line is the
// header, which stays put while the body scrolls.
func (t Table) Render(th *theme.Theme, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	cols, idx := t.visibleColumns(width)
	if len(cols) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(t.renderHeader(th, cols, width))
	// Hairline under the sticky header makes column labels scan faster.
	b.WriteByte('\n')
	b.WriteString(th.HeaderRule.Render(strings.Repeat("─", width)))

	bodyHeight := height - 2
	if bodyHeight <= 0 {
		return b.String()
	}

	end := min(t.Offset+bodyHeight, len(t.Rows))
	for i := t.Offset; i < end; i++ {
		b.WriteByte('\n')
		b.WriteString(t.renderRow(th, cols, idx, i, width))
	}
	// Pad the remaining rows so the panel below never shifts upward.
	for i := end - t.Offset; i < bodyHeight; i++ {
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(" ", width))
	}
	return b.String()
}

func (t Table) renderHeader(th *theme.Theme, cols []Column, width int) string {
	parts := make([]string, 0, len(cols))
	for _, c := range cols {
		title := c.Title
		if c.ShortTitle != "" && Width(title) > c.MinWidth {
			title = c.ShortTitle
		}
		cell := Fit(strings.ToUpper(title), c.MinWidth, "")
		if c.Right {
			cell = PadLeft(Truncate(strings.ToUpper(title), c.MinWidth, ""), c.MinWidth)
		}
		parts = append(parts, cell)
	}
	// The two leading cells account for the selection marker gutter.
	return th.TableHeader.Render(Pad("  "+strings.Join(parts, " "), width))
}

func (t Table) renderRow(th *theme.Theme, cols []Column, idx []int, row, width int) string {
	r := t.Rows[row]
	selected := row == t.Selected

	parts := make([]string, 0, len(cols))
	for i, c := range cols {
		var raw string
		if src := idx[i]; src < len(r.Cells) {
			raw = r.Cells[src]
		}
		if c.Right {
			parts = append(parts, PadLeft(Truncate(raw, c.MinWidth, th.Sym.Ellipsis), c.MinWidth))
			continue
		}
		parts = append(parts, Fit(raw, c.MinWidth, th.Sym.Ellipsis))
	}

	marker := "  "
	if selected {
		marker = th.TableMarker.Render(th.Sym.Selected) + " "
	}
	line := Pad(marker+strings.Join(parts, " "), width)

	switch {
	case selected && t.Focused:
		// Restyling the whole line, rather than only the background, keeps the
		// selection visible on terminals that drop background colours.
		return th.TableRowActive.Render(stripStyles(marker+strings.Join(parts, " "), width))
	case selected:
		return th.TableRowInactive.Render(stripStyles(marker+strings.Join(parts, " "), width))
	case t.ZebraStripes && row%2 == 1:
		return th.TableRowAlt.Render(line)
	default:
		return line
	}
}

// stripStyles removes embedded styling so a selected row renders in one
// consistent colour instead of a patchwork of per-cell foregrounds.
func stripStyles(s string, width int) string {
	return Pad(stripANSI(s), width)
}

// VisibleRows returns how many body rows fit in the given height.
// One row is reserved for the header label and one for the hairline rule.
func VisibleRows(height int) int {
	if height <= 2 {
		return 0
	}
	return height - 2
}

// ClampOffset returns a scroll offset that keeps the selected row on screen,
// scrolling by the minimum amount needed. Called on every selection change.
func ClampOffset(offset, selected, rows, visible int) int {
	if visible <= 0 || rows == 0 {
		return 0
	}
	if selected < 0 {
		return 0
	}
	if selected < offset {
		offset = selected
	}
	if selected >= offset+visible {
		offset = selected - visible + 1
	}
	if maxOffset := rows - visible; offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}
	return offset
}
