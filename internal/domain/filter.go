package domain

import (
	"strings"
)

// FilterField identifies a scoped filter term such as "status:" or "branch:".
type FilterField string

// Supported filter fields.
const (
	FieldFree    FilterField = ""
	FieldStatus  FilterField = "status"
	FieldProject FilterField = "project"
	FieldEnv     FilterField = "env"
	FieldBranch  FilterField = "branch"
	FieldServer  FilterField = "server"
	FieldDomain  FilterField = "domain"
	FieldName    FilterField = "name"
)

// FilterFields lists the scoped fields, used for the filter input's hint line.
var FilterFields = []FilterField{
	FieldStatus, FieldProject, FieldEnv, FieldBranch, FieldServer, FieldDomain, FieldName,
}

// aliases maps the prefixes users actually type onto canonical fields.
var fieldAliases = map[string]FilterField{
	"status": FieldStatus, "state": FieldStatus, "s": FieldStatus,
	"project": FieldProject, "proj": FieldProject, "p": FieldProject,
	"env": FieldEnv, "environment": FieldEnv, "e": FieldEnv,
	"branch": FieldBranch, "b": FieldBranch,
	"server": FieldServer,
	"domain": FieldDomain, "fqdn": FieldDomain, "url": FieldDomain,
	"name": FieldName, "app": FieldName,
}

// FilterTerm is one parsed token of a filter query.
type FilterTerm struct {
	Field FilterField
	Value string
	// Negated is set when the term was prefixed with '-', excluding matches.
	Negated bool
}

// Filter is a parsed dashboard query. The zero value matches everything.
type Filter struct {
	// Raw is the text the user typed, kept verbatim for the empty state message.
	Raw   string
	Terms []FilterTerm
}

// ParseFilter turns a query string into a filter.
//
// Whitespace separates terms, all of which must match (AND). A term may be
// scoped with "field:value", negated with a leading "-", or left as free text
// that is searched across every relevant column.
func ParseFilter(query string) Filter {
	f := Filter{Raw: query}
	for _, tok := range strings.Fields(query) {
		term := FilterTerm{}
		if rest, ok := strings.CutPrefix(tok, "-"); ok && rest != "" {
			term.Negated = true
			tok = rest
		}
		if name, value, ok := strings.Cut(tok, ":"); ok {
			if field, known := fieldAliases[strings.ToLower(name)]; known {
				// A scoped term with an empty value ("status:") is the user
				// mid-typing; ignore it rather than matching nothing.
				if value == "" {
					continue
				}
				term.Field = field
				term.Value = strings.ToLower(value)
				f.Terms = append(f.Terms, term)
				continue
			}
		}
		term.Value = strings.ToLower(tok)
		f.Terms = append(f.Terms, term)
	}
	return f
}

// IsEmpty reports whether the filter matches everything.
func (f Filter) IsEmpty() bool { return len(f.Terms) == 0 }

// MatchApplication reports whether an application satisfies every term.
func (f Filter) MatchApplication(a Application) bool {
	for _, t := range f.Terms {
		if t.match(a) == t.Negated {
			return false
		}
	}
	return true
}

func (t FilterTerm) match(a Application) bool {
	switch t.Field {
	case FieldStatus:
		return matchStatus(a.Status, t.Value)
	case FieldProject:
		return contains(a.Project.Name, t.Value)
	case FieldEnv:
		return contains(a.Environment.Name, t.Value)
	case FieldBranch:
		return contains(a.Branch, t.Value)
	case FieldServer:
		return contains(a.Server.Name, t.Value)
	case FieldDomain:
		return containsAny(a.FQDNs, t.Value)
	case FieldName:
		return contains(a.Name, t.Value)
	default:
		return contains(a.Name, t.Value) ||
			contains(a.Description, t.Value) ||
			contains(a.Project.Name, t.Value) ||
			contains(a.Environment.Name, t.Value) ||
			contains(a.Branch, t.Value) ||
			contains(a.Server.Name, t.Value) ||
			contains(a.UUID, t.Value) ||
			containsAny(a.FQDNs, t.Value) ||
			contains(a.RepositoryURL, t.Value)
	}
}

// matchStatus accepts the normalised state name, the health verdict and a few
// convenient synonyms, so "status:down" and "status:stopped" both work.
func matchStatus(s Status, value string) bool {
	if strings.HasPrefix(strings.ToLower(string(s.State)), value) {
		return true
	}
	if strings.HasPrefix(strings.ToLower(string(s.Health)), value) && s.Health != HealthNone {
		return true
	}
	switch value {
	case "up", "ok", "healthy":
		return s.State == StatusRunning && s.Health != HealthUnhealthy
	case "down":
		return s.State == StatusStopped || s.State == StatusFailed
	case "bad", "broken", "attention":
		return s.NeedsAttention()
	case "busy", "transitional", "pending":
		return s.IsTransitional()
	}
	return false
}

func contains(haystack, needle string) bool {
	if haystack == "" {
		return false
	}
	return strings.Contains(strings.ToLower(haystack), needle)
}

func containsAny(haystack []string, needle string) bool {
	for _, h := range haystack {
		if contains(h, needle) {
			return true
		}
	}
	return false
}
