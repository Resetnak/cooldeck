package views

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// logContent is one rendered log buffer. The detail screen and the fleet tail
// share it so their ergonomics - follow, wrap, search highlighting - cannot
// drift apart.
type logContent struct {
	title       string
	meta        string
	lines       []domain.LogLine
	wrap        bool
	follow      bool
	trackScroll bool
	search      string
	// tag returns a per-line prefix, used by the fleet tail to name the
	// application a line came from. Nil for a single-application buffer.
	tag func(i int) string
}

// renderLogBlock draws the buffer and returns the body along with the resolved
// scroll position, so the caller owns the scroll state rather than this
// function reaching back into a view.
func renderLogBlock(th *theme.Theme, width, height int, content logContent, scroll int) (body string, resolved, maxScroll int) {
	lines := []string{th.Title.Render(content.title), th.Subtle.Render(content.meta), ""}

	var search *regexp.Regexp
	if content.search != "" {
		search = regexp.MustCompile("(?i)" + regexp.QuoteMeta(content.search))
	}

	for i, line := range content.lines {
		prefix := ""
		if content.tag != nil {
			prefix = content.tag(i)
		}
		timestamp := line.TimestampText
		if timestamp == "" && !line.Timestamp.IsZero() {
			timestamp = line.Timestamp.Format("15:04:05")
		}
		if timestamp != "" {
			prefix += th.Subtle.Render(timestamp) + " "
		}
		if label := line.Level.Label(); label != "" {
			prefix += logLevelStyle(th, line.Level).Render(components.Pad(label, 5)) + " "
		}

		text := prefix + line.Text
		if search != nil {
			text = search.ReplaceAllStringFunc(text, func(match string) string {
				return th.FilterPrompt.Render(match)
			})
		}
		if content.wrap {
			lines = append(lines, strings.Split(components.Wrap(text, width), "\n")...)
		} else {
			lines = append(lines, components.Truncate(text, width, th.Sym.Ellipsis))
		}
	}

	maxScroll = max(len(lines)-height, 0)
	if content.follow {
		scroll = maxScroll
	}
	scroll = min(max(scroll, 0), maxScroll)
	return components.FitBlock(strings.Join(lines[scroll:], "\n"), width, height), scroll, maxScroll
}

func logLevelStyle(th *theme.Theme, level domain.LogLevel) lipgloss.Style {
	switch level {
	case domain.LogLevelWarn:
		return th.Attention
	case domain.LogLevelError, domain.LogLevelFatal:
		return th.Danger
	case domain.LogLevelInfo:
		return th.Note
	default:
		return th.Subtle
	}
}
