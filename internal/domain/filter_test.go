package domain

import (
	"testing"
	"time"
)

func testApps() []Application {
	return []Application{
		{
			UUID: "a1", Name: "stimustop-api", Branch: "main",
			Status:      ParseStatus("running:healthy"),
			FQDNs:       []string{"api.stimustop.com"},
			Project:     ResourceRef{Name: "StimuStop"},
			Environment: ResourceRef{Name: "production"},
			Server:      ResourceRef{Name: "hetzner-1"},
		},
		{
			UUID: "a2", Name: "resetnak-web", Branch: "main",
			Status:      ParseStatus("in_progress"),
			FQDNs:       []string{"resetnak.cz", "www.resetnak.cz"},
			Project:     ResourceRef{Name: "Personal"},
			Environment: ResourceRef{Name: "production"},
		},
		{
			UUID: "a3", Name: "maintenea-worker", Branch: "develop",
			Status:      ParseStatus("failed"),
			Project:     ResourceRef{Name: "Maintenea"},
			Environment: ResourceRef{Name: "staging"},
		},
		{
			UUID: "a4", Name: "stimustop-cron", Branch: "main",
			Status:      ParseStatus("running:unhealthy"),
			Project:     ResourceRef{Name: "StimuStop"},
			Environment: ResourceRef{Name: "production"},
		},
	}
}

func matchNames(f Filter, apps []Application) []string {
	var out []string
	for _, a := range apps {
		if f.MatchApplication(a) {
			out = append(out, a.Name)
		}
	}
	return out
}

func TestFilterMatching(t *testing.T) {
	apps := testApps()
	tests := []struct {
		query string
		want  []string
	}{
		{"", []string{"stimustop-api", "resetnak-web", "maintenea-worker", "stimustop-cron"}},
		{"stimustop", []string{"stimustop-api", "stimustop-cron"}},
		{"status:failed", []string{"maintenea-worker"}},
		{"status:running", []string{"stimustop-api"}},
		// A degraded app is not "running"; it needs its own bucket.
		{"status:degraded", []string{"stimustop-cron"}},
		{"status:unhealthy", []string{"stimustop-cron"}},
		{"project:stimustop", []string{"stimustop-api", "stimustop-cron"}},
		{"env:production", []string{"stimustop-api", "resetnak-web", "stimustop-cron"}},
		{"branch:main", []string{"stimustop-api", "resetnak-web", "stimustop-cron"}},
		{"domain:resetnak.cz", []string{"resetnak-web"}},
		// Terms combine with AND.
		{"project:stimustop status:failed", nil},
		{"branch:main env:production", []string{"stimustop-api", "resetnak-web", "stimustop-cron"}},
		// Negation excludes.
		{"-status:running", []string{"resetnak-web", "maintenea-worker", "stimustop-cron"}},
		{"project:stimustop -status:degraded", []string{"stimustop-api"}},
		// Free text searches every relevant column.
		{"hetzner-1", []string{"stimustop-api"}},
		{"a3", []string{"maintenea-worker"}},
		// Case is irrelevant.
		{"STATUS:FAILED", []string{"maintenea-worker"}},
		// Aliases.
		{"p:personal", []string{"resetnak-web"}},
		{"s:failed", []string{"maintenea-worker"}},
		// Synonyms.
		{"status:bad", []string{"maintenea-worker", "stimustop-cron"}},
		{"status:busy", []string{"resetnak-web"}},
		{"status:down", []string{"maintenea-worker"}},
		// A half-typed scoped term must not blank the list.
		{"status:", []string{"stimustop-api", "resetnak-web", "maintenea-worker", "stimustop-cron"}},
		// An unknown prefix falls back to free text and matches nothing here.
		{"colour:red", nil},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := matchNames(ParseFilter(tt.query), apps)
			if len(got) != len(tt.want) {
				t.Fatalf("ParseFilter(%q) matched %v, want %v", tt.query, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ParseFilter(%q) matched %v, want %v", tt.query, got, tt.want)
				}
			}
		})
	}
}

func TestFilterIsEmpty(t *testing.T) {
	if !ParseFilter("   ").IsEmpty() {
		t.Error("whitespace-only query should be empty")
	}
	if ParseFilter("x").IsEmpty() {
		t.Error("non-empty query should not be empty")
	}
	if f := ParseFilter("status:failed"); f.Raw != "status:failed" {
		t.Errorf("Raw should preserve the typed query, got %q", f.Raw)
	}
}

func TestIsProductionEnvironment(t *testing.T) {
	prod := []string{"production", "prod", "Production", "PROD", "live", "prd", "app-prod", "prod-eu"}
	for _, name := range prod {
		if !IsProductionEnvironment(name) {
			t.Errorf("%q should be production", name)
		}
	}
	// A false positive only makes a confirmation louder; a false negative
	// lets a production restart through unannounced, so err toward matching.
	notProd := []string{"staging", "dev", "development", "test", "reproduction", "preview", ""}
	for _, name := range notProd {
		if IsProductionEnvironment(name) {
			t.Errorf("%q should not be production", name)
		}
	}

	app := Application{Environment: ResourceRef{Name: "production"}}
	if !app.IsProduction() {
		t.Error("application in a production environment should report as production")
	}
}

func TestHumanizeDuration(t *testing.T) {
	tests := map[time.Duration]string{
		250 * time.Millisecond: "250ms",
		18 * time.Second:       "18s",
		84 * time.Second:       "1m 24s",
		2 * time.Minute:        "2m",
		125 * time.Minute:      "2h 5m",
		2 * time.Hour:          "2h",
		76 * time.Hour:         "3d 4h",
		72 * time.Hour:         "3d",
	}
	for d, want := range tests {
		if got := HumanizeDuration(d); got != want {
			t.Errorf("HumanizeDuration(%s) = %q, want %q", d, got, want)
		}
	}
}

func TestHumanizeAge(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	if got := HumanizeAge(time.Time{}, now); got != "never" {
		t.Errorf("zero time = %q, want never", got)
	}
	if got := HumanizeAge(now.Add(-3*time.Minute), now); got != "3m ago" {
		t.Errorf("got %q, want 3m ago", got)
	}
	if got := HumanizeAge(now.Add(-time.Second), now); got != "just now" {
		t.Errorf("got %q, want just now", got)
	}
	// Server clock ahead of ours must not render as a negative age.
	if got := HumanizeAge(now.Add(time.Hour), now); got != "just now" {
		t.Errorf("future timestamp = %q, want just now", got)
	}
}

func TestWebURLForRepository(t *testing.T) {
	tests := map[string]string{
		"https://github.com/resetnak/cooldeck.git": "https://github.com/resetnak/cooldeck",
		"git@github.com:resetnak/cooldeck.git":     "https://github.com/resetnak/cooldeck",
		"ssh://git@gitlab.com/group/repo.git":      "https://gitlab.com/group/repo",
		"ssh://git@git.example.com:2222/a/b.git":   "https://git.example.com/a/b",
		"resetnak/cooldeck":                        "https://github.com/resetnak/cooldeck",
		// Anything that cannot be mapped safely yields nothing: the result is
		// handed to the OS URL opener, so a wrong guess is worse than no link.
		"":                    "",
		"not a url":           "",
		"file:///etc/hosts":   "",
		"javascript:alert(1)": "",
	}
	for in, want := range tests {
		if got := WebURLForRepository(in); got != want {
			t.Errorf("WebURLForRepository(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDomainURL(t *testing.T) {
	tests := map[string]string{
		"example.com":          "https://example.com",
		"https://example.com":  "https://example.com",
		"http://localhost:300": "http://localhost:300",
		" example.com ":        "https://example.com",
		"":                     "",
		// A non-web scheme must never reach the OS opener.
		"javascript:alert(1)": "",
		"file:///etc/passwd":  "",
	}
	for in, want := range tests {
		if got := DomainURL(in); got != want {
			t.Errorf("DomainURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitFQDNs(t *testing.T) {
	got := SplitFQDNs("a.com, b.com ,, c.com")
	want := []string{"a.com", "b.com", "c.com"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if SplitFQDNs("  ") != nil {
		t.Error("blank input should yield nil")
	}
}
