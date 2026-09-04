package theme

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestFillRestoresBackgroundAfterNestedReset(t *testing.T) {
	outer := lipgloss.NewStyle().Background(lipgloss.Color("#FFFFFF"))
	inner := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("x") + " tail"

	got := Fill(outer, inner)
	if !strings.Contains(got, "\x1b[m\x1b[48;2;255;255;255m tail") {
		t.Fatalf("background not restored after inner reset: %q", got)
	}
	if strings.Contains(got, "38;2;0;0;0") {
		t.Fatalf("unset foreground leaked as black: %q", got)
	}
}
