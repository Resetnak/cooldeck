package components

import (
	"math/rand"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// EmptyState renders a centred explanation with optional next steps. Every
// empty state says what happened and what the user can do about it; a bare
// "no results" is a dead end.
func EmptyState(th *theme.Theme, width, height int, title, body string, hints []KeyHint) string {
	inner := min(width-4, 64)
	if inner < 10 {
		inner = max(width-2, 1)
	}

	block := []string{th.EmptyTitle.Render(Truncate(title, inner, th.Sym.Ellipsis))}
	if body != "" {
		block = append(block, "", th.EmptyBody.Render(Wrap(body, inner)))
	}
	if len(hints) > 0 {
		parts := make([]string, 0, len(hints))
		for _, h := range hints {
			parts = append(parts, th.FooterKey.Render("["+h.Key+"]")+" "+th.EmptyHint.Render(h.Desc))
		}
		block = append(block, "", strings.Join(parts, "   "))
	}

	card := th.Panel.Width(inner).Render(lipgloss.JoinVertical(lipgloss.Left, block...))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, card)
}

// ErrorState renders a domain error as a readable panel. It shows the title,
// the human message and the suggested action, never a stack trace, and never
// the raw response body, which could contain credentials.
func ErrorState(th *theme.Theme, width, height int, err *domain.Error, hints []KeyHint) string {
	if err == nil {
		return EmptyState(th, width, height, "Something went wrong", "", hints)
	}

	inner := min(width-4, 64)
	if inner < 10 {
		inner = max(width-2, 1)
	}

	block := []string{
		th.Danger.Render(th.Sym.Error + " " + Truncate(err.Title, inner-2, th.Sym.Ellipsis)),
		"",
		th.EmptyBody.Render(Wrap(err.Message, inner)),
	}
	if err.Suggestion != "" {
		block = append(block, "", th.EmptyHint.Render(Wrap(err.Suggestion, inner)))
	}
	if len(hints) > 0 {
		parts := make([]string, 0, len(hints))
		for _, h := range hints {
			parts = append(parts, th.FooterKey.Render("["+h.Key+"]")+" "+th.EmptyHint.Render(h.Desc))
		}
		block = append(block, "", strings.Join(parts, "   "))
	}

	card := th.Modal.Width(inner).Render(lipgloss.JoinVertical(lipgloss.Left, block...))
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, card)
}

// skeletonSeed keeps the placeholder bar lengths stable between frames, so the
// loading state does not shimmer while it waits.
const skeletonSeed = 7

// Skeleton renders placeholder rows for the first load. Unlike a bare spinner
// it communicates the shape of what is coming, and it keeps the layout from
// jumping once the data arrives.
func Skeleton(th *theme.Theme, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	r := rand.New(rand.NewSource(skeletonSeed))

	lines := make([]string, 0, height)
	lines = append(lines, th.TableHeader.Render(Pad("  LOADING"+strings.Repeat(".", 3), width)))
	for range height - 1 {
		barWidth := min(width/3+r.Intn(max(width/2, 1)), width-4)
		barWidth = max(barWidth, 1)
		lines = append(lines, th.Skeleton.Render(Pad("  "+strings.Repeat("░", barWidth), width)))
	}
	return strings.Join(lines, "\n")
}

// TooSmall is shown when the terminal cannot host a usable layout. It has to
// render correctly at any size, so it uses no borders and no colour blocks.
func TooSmall(th *theme.Theme, width, height int) string {
	lines := []string{
		"Terminal too small",
		"",
		"cooldeck needs at least",
		strconv.Itoa(theme.MinUsableWidth) + "x" + strconv.Itoa(theme.MinUsableHeight) + " characters.",
		"",
		"Now: " + strconv.Itoa(width) + "x" + strconv.Itoa(height),
	}
	for i, l := range lines {
		lines[i] = lipgloss.PlaceHorizontal(max(width, 1), lipgloss.Center, l)
	}
	body := strings.Join(lines, "\n")
	if height >= len(lines) {
		return lipgloss.PlaceVertical(height, lipgloss.Center, body)
	}
	return body
}
