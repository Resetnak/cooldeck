package views

import (
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/theme"
)

// MaxTailApps caps how many applications can be tailed at once. Coolify serves
// logs as whole snapshots rather than a stream, so every tailed application is
// a separate poll on every tick - the same rate limit config.MinRefreshInterval
// exists to protect. Five is also about as many interleaved streams as anyone
// can actually read.
const MaxTailApps = 5

// tailSource is one application's latest log snapshot.
type tailSource struct {
	uuid     string
	name     string
	lines    []domain.LogLine
	loadedAt time.Time
	// failure is the last error text for this source, kept so one unreachable
	// application degrades to a note in the header instead of emptying the view.
	failure string
}

// Tail merges the runtime logs of several applications into one buffer.
//
// Lines are ordered by timestamp where the source provides one. A line without
// a timestamp inherits the one before it from the same application, so a stack
// trace stays under the line it belongs to; an application that prints no
// timestamps at all is ordered by when its snapshot was taken.
type Tail struct {
	sources []tailSource
	loaded  bool

	follow bool
	paused bool
	wrap   bool
	search string
	match  int
	scroll int

	maxScroll int
}

// NewTail returns an empty tail view.
func NewTail() *Tail { return &Tail{follow: true} }

// SetApplications resets the view to a new set of applications, preserving any
// snapshot already held for one that stays.
func (v *Tail) SetApplications(apps []domain.Application) {
	previous := make(map[string]tailSource, len(v.sources))
	for _, s := range v.sources {
		previous[s.uuid] = s
	}

	v.sources = make([]tailSource, 0, min(len(apps), MaxTailApps))
	for _, a := range apps {
		if len(v.sources) == MaxTailApps {
			break
		}
		source := tailSource{uuid: a.UUID, name: a.Name}
		if old, ok := previous[a.UUID]; ok {
			source = old
			source.name = a.Name
		}
		v.sources = append(v.sources, source)
	}
	v.loaded = false
	v.scroll = 0
}

// UUIDs lists the applications being tailed, in display order.
func (v *Tail) UUIDs() []string {
	out := make([]string, 0, len(v.sources))
	for _, s := range v.sources {
		out = append(out, s.uuid)
	}
	return out
}

// Count returns how many applications are being tailed.
func (v *Tail) Count() int { return len(v.sources) }

// SetLines stores one application's snapshot.
func (v *Tail) SetLines(uuid string, lines []domain.LogLine, at time.Time) {
	for i := range v.sources {
		if v.sources[i].uuid != uuid {
			continue
		}
		v.sources[i].lines = append([]domain.LogLine{}, lines...)
		v.sources[i].loadedAt = at
		v.sources[i].failure = ""
		v.loaded = true
		return
	}
}

// SetFailure records that one source could not be read. The rest keep working.
func (v *Tail) SetFailure(uuid, reason string) {
	for i := range v.sources {
		if v.sources[i].uuid == uuid {
			v.sources[i].failure = reason
			v.loaded = true
			return
		}
	}
}

// Loaded reports whether at least one snapshot has arrived.
func (v *Tail) Loaded() bool { return v.loaded }

// Paused reports whether polling is suspended.
func (v *Tail) Paused() bool { return v.paused }

// TogglePause suspends or resumes polling and returns the new state.
func (v *Tail) TogglePause() bool {
	v.paused = !v.paused
	return v.paused
}

// Following reports whether the view stays pinned to the newest line.
func (v *Tail) Following() bool { return v.follow }

// ToggleFollow pins or unpins the tail and returns the new state.
func (v *Tail) ToggleFollow() bool {
	v.follow = !v.follow
	return v.follow
}

// ToggleWrap switches line wrapping and returns the new state.
func (v *Tail) ToggleWrap() bool {
	v.wrap = !v.wrap
	return v.wrap
}

// Search returns the active query.
func (v *Tail) Search() string { return v.search }

// SetSearch replaces the query and resets the match cursor.
func (v *Tail) SetSearch(query string) {
	v.search = query
	v.match = 0
}

// Scroll moves the buffer by delta lines and drops out of follow mode, because
// scrolling up while pinned to the tail would fight the user.
func (v *Tail) Scroll(delta int) {
	if delta < 0 {
		v.follow = false
	}
	v.scroll = min(max(v.scroll+delta, 0), v.maxScroll)
}

// Text returns the merged buffer as plain text for copying.
func (v *Tail) Text() string {
	entries := v.merged(time.Time{})
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		parts = append(parts, e.name+"  "+e.line.String())
	}
	return strings.Join(parts, "\n")
}

// entry is one line with the application it came from.
type entry struct {
	name string
	// tagIndex selects the colour, so an application keeps the same one for as
	// long as it is being tailed.
	tagIndex int
	line     domain.LogLine
	at       time.Time
}

// merged flattens the sources into one time-ordered buffer.
func (v *Tail) merged(now time.Time) []entry {
	total := 0
	for _, s := range v.sources {
		total += len(s.lines)
	}
	entries := make([]entry, 0, total)

	for i, s := range v.sources {
		// Carry the last seen timestamp forward so continuation lines stay
		// with the line they belong to.
		carried := s.loadedAt
		if carried.IsZero() {
			carried = now
		}
		for _, line := range s.lines {
			if !line.Timestamp.IsZero() {
				carried = line.Timestamp
			}
			entries = append(entries, entry{name: s.name, tagIndex: i, line: line, at: carried})
		}
	}

	// A stable sort keeps each application's own lines in the order it printed
	// them, even when several share a timestamp.
	sortStableByTime(entries)
	return entries
}

func sortStableByTime(entries []entry) {
	// Insertion sort: the buffer is already almost ordered - each source is
	// internally sorted - and this keeps equal timestamps in source order.
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].at.Before(entries[j-1].at); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
}

// tagStyles colours the application name on each line. They are existing theme
// styles rather than new colours, so the WCAG contrast test already covers
// them, and there are exactly as many as MaxTailApps.
func tagStyles(th *theme.Theme) []lipgloss.Style {
	return []lipgloss.Style{th.Accent, th.Note, th.Positive, th.Label, th.Strong}
}

// Render draws the merged buffer.
func (v *Tail) Render(th *theme.Theme, width, height int, now time.Time) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if len(v.sources) == 0 {
		return components.EmptyState(th, width, height,
			"Nothing selected to tail",
			"Mark applications with space on the applications list, then press t.",
			[]components.KeyHint{{Key: "1", Desc: "applications"}, {Key: "esc", Desc: "back"}})
	}
	if !v.loaded {
		return components.Skeleton(th, width, height)
	}

	entries := v.merged(now)
	lines := make([]domain.LogLine, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, e.line)
	}

	styles := tagStyles(th)
	// The widest name sets the gutter, so the log text lines up across sources.
	nameWidth := 0
	for _, s := range v.sources {
		nameWidth = max(nameWidth, components.Width(s.name))
	}
	nameWidth = min(nameWidth, 20)

	tag := func(i int) string {
		e := entries[i]
		style := styles[e.tagIndex%len(styles)]
		return style.Render(th.Sym.Selected+components.Pad(components.Truncate(e.name, nameWidth, ""), nameWidth)) + " "
	}

	body, scroll, maxScroll := renderLogBlock(th, width, height, logContent{
		title:  "FLEET TAIL",
		meta:   v.meta(now, len(entries)),
		lines:  lines,
		wrap:   v.wrap,
		follow: v.follow,
		search: v.search,
		tag:    tag,
	}, v.scroll)
	v.scroll = scroll
	v.maxScroll = maxScroll
	return body
}

func (v *Tail) meta(now time.Time, count int) string {
	names := make([]string, 0, len(v.sources))
	for _, s := range v.sources {
		label := s.name
		if s.failure != "" {
			label += " (" + s.failure + ")"
		}
		names = append(names, label)
	}

	meta := strconv.Itoa(count) + " lines  " + strings.Join(names, "  ")
	newest := time.Time{}
	for _, s := range v.sources {
		if s.loadedAt.After(newest) {
			newest = s.loadedAt
		}
	}
	if !newest.IsZero() {
		meta += "  " + domain.HumanizeAge(newest, now)
	}
	if v.paused {
		meta += "  paused"
	}
	if v.follow {
		meta += "  follow"
	}
	if v.wrap {
		meta += "  wrap"
	}
	if v.search != "" {
		meta += "  /" + v.search
	}
	return meta
}
