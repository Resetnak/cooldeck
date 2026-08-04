package theme

import "github.com/resetnak/cooldeck/internal/domain"

// SymbolSet is the glyph vocabulary for one level of terminal capability.
// cooldeck must be fully usable without a Nerd Font, so the Unicode set is the
// default and Nerd Font glyphs are a pure upgrade, never a requirement.
type SymbolSet struct {
	StatusRunning    string
	StatusDegraded   string
	StatusStopped    string
	StatusFailed     string
	StatusBuilding   string
	StatusDeploying  string
	StatusQueued     string
	StatusRestarting string
	StatusUnknown    string

	Success string
	Warning string
	Error   string
	Info    string

	Bullet     string
	ArrowRight string
	ArrowUp    string
	ArrowDown  string
	Ellipsis   string
	Dash       string
	Separator  string
	Selected   string
	Check      string
	Cross      string
	Filter     string
	Lock       string
	Spinner    []string
}

// UnicodeSymbols is the default set: geometric shapes that every modern
// terminal font has, with no Nerd Font dependency.
var UnicodeSymbols = SymbolSet{
	StatusRunning:    "●",
	StatusDegraded:   "▲",
	StatusStopped:    "■",
	StatusFailed:     "×",
	StatusBuilding:   "◐",
	StatusDeploying:  "◌",
	StatusQueued:     "◌",
	StatusRestarting: "◑",
	StatusUnknown:    "?",

	Success: "✓",
	Warning: "!",
	Error:   "×",
	Info:    "·",

	Bullet:     "•",
	ArrowRight: "→",
	ArrowUp:    "↑",
	ArrowDown:  "↓",
	Ellipsis:   "…",
	Dash:       "—",
	Separator:  "│",
	Selected:   "▌",
	Check:      "✓",
	Cross:      "✕",
	Filter:     "/",
	Lock:       "⊘",
	Spinner:    []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
}

// ASCIISymbols is the fallback for terminals or fonts that mangle box-drawing
// and geometric characters, and for anyone who prefers plain output.
var ASCIISymbols = SymbolSet{
	StatusRunning:    "o",
	StatusDegraded:   "!",
	StatusStopped:    "#",
	StatusFailed:     "x",
	StatusBuilding:   "*",
	StatusDeploying:  ".",
	StatusQueued:     ".",
	StatusRestarting: "*",
	StatusUnknown:    "?",

	Success: "+",
	Warning: "!",
	Error:   "x",
	Info:    "-",

	Bullet:     "*",
	ArrowRight: "->",
	ArrowUp:    "^",
	ArrowDown:  "v",
	Ellipsis:   "...",
	Dash:       "-",
	Separator:  "|",
	Selected:   ">",
	Check:      "+",
	Cross:      "x",
	Filter:     "/",
	Lock:       "!",
	Spinner:    []string{"|", "/", "-", "\\"},
}

// NerdFontSymbols upgrades a few glyphs where a Nerd Font genuinely reads
// better. Everything else deliberately matches the Unicode set so the two
// layouts have identical widths.
var NerdFontSymbols = func() SymbolSet {
	s := UnicodeSymbols
	s.StatusRunning = ""  // filled circle
	s.StatusStopped = ""  // stop square
	s.StatusFailed = ""   // times
	s.StatusBuilding = "" // cogs
	s.Success = ""
	s.Filter = ""
	s.Lock = ""
	return s
}()

// StatusSymbol returns the glyph for a resource status. The glyph is always
// accompanied by its text label in the UI, so meaning never depends on the
// symbol or its colour alone.
func (s SymbolSet) StatusSymbol(st domain.Status) string {
	switch st.State {
	case domain.StatusRunning:
		return s.StatusRunning
	case domain.StatusDegraded:
		return s.StatusDegraded
	case domain.StatusStopped, domain.StatusPaused:
		return s.StatusStopped
	case domain.StatusFailed:
		return s.StatusFailed
	case domain.StatusBuilding:
		return s.StatusBuilding
	case domain.StatusDeploying:
		return s.StatusDeploying
	case domain.StatusStarting, domain.StatusStopping, domain.StatusRestarting:
		return s.StatusRestarting
	default:
		return s.StatusUnknown
	}
}

// DeploymentSymbol returns the glyph for a deployment status.
func (s SymbolSet) DeploymentSymbol(st domain.DeploymentStatus) string {
	switch st {
	case domain.DeploymentFinished:
		return s.Check
	case domain.DeploymentFailed:
		return s.StatusFailed
	case domain.DeploymentInProgress:
		return s.StatusBuilding
	case domain.DeploymentQueued:
		return s.StatusQueued
	case domain.DeploymentCancelled:
		return s.StatusStopped
	default:
		return s.StatusUnknown
	}
}
