package tui

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/resetnak/cooldeck/internal/app"
	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/domain"
	"github.com/resetnak/cooldeck/internal/tui/components"
	"github.com/resetnak/cooldeck/internal/tui/views"
	"github.com/resetnak/cooldeck/internal/version"
)

// updateGolden is set by `make test-update-golden` via UPDATE_GOLDEN=1, or by
// `go test -update`.
var updateGolden = os.Getenv("UPDATE_GOLDEN") == "1"

func init() {
	flag.BoolFunc("update", "update golden files", func(string) error {
		updateGolden = true
		return nil
	})
}

var (
	// spinnerDots matches the rotating spinner frames so goldens do not flake.
	spinnerDots = regexp.MustCompile(`[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]`)
	// orphanCSI catches colour sequences left behind when a stripper removes
	// only the ESC byte, or when a style emits partial sequences.
	orphanCSI = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]|\[[0-9;]*m`)
	// trailingSpace collapses accidental trailing spaces on each line.
	trailingSpace = regexp.MustCompile(`[ \t]+$`)
	// absoluteTime is rendered via time.Local in domain.HumanizeTime, so the
	// clock face differs between developer laptops and UTC CI runners.
	absoluteTime = regexp.MustCompile(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`)
	// hostPlatform covers diagnostics lines that embed runtime.GOOS/GOARCH.
	hostPlatform = regexp.MustCompile(`(?m)^(Platform|GOOS/GOARCH)(\s+)\S+/\S+`)
	// goVersion may differ slightly between local toolchains and CI setup-go.
	goVersion = regexp.MustCompile(`(?m)^(Go)(\s+)go\d+\.\d+(?:\.\d+)?`)
)

func TestGoldenViews(t *testing.T) {
	prevV, prevC, prevD := version.Version, version.Commit, version.Date
	version.Version, version.Commit, version.Date = "test", "abcdef1", "2026-08-04"
	t.Cleanup(func() {
		version.Version, version.Commit, version.Date = prevV, prevC, prevD
	})

	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	service := demo.New(demo.Options{
		Seed: demo.DefaultSeed,
		Now:  func() time.Time { return now },
	})
	snapshot, err := service.Dashboard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	detail, err := service.ApplicationDetail(t.Context(), snapshot.Applications[0].UUID)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := service.RuntimeLogs(t.Context(), snapshot.Applications[0].UUID, 40)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := service.Connect(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		width  int
		height int
		setup  func(*Model)
	}{
		{
			name: "dashboard-wide", width: 160, height: 40,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
			},
		},
		{
			name: "dashboard-standard", width: 100, height: 28,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
			},
		},
		{
			name: "dashboard-compact", width: 70, height: 22,
			setup: func(m *Model) {
				m.forceCompact = true
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
			},
		},
		{
			name: "dashboard-empty", width: 100, height: 28,
			setup: func(m *Model) {
				m.apps.SetApplications(nil, nil, now)
			},
		},
		{
			name: "dashboard-offline", width: 100, height: 28,
			setup: func(m *Model) {
				m.connection = components.ConnectionOffline
				m.lastError = domain.NewError(domain.ErrorNetwork, nil)
			},
		},
		{
			name: "app-detail", width: 120, height: 32,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.screen = screenDetail
				m.detail.SetApplication(detail.Application, now)
				m.detail.SetDeployments(detail.Deployments)
			},
		},
		{
			name: "runtime-logs", width: 120, height: 32,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.screen = screenDetail
				m.detail.SetApplication(detail.Application, now)
				m.detail.SetTab(views.TabRuntimeLogs)
				m.detail.SetRuntimeLogs(logs.Lines, now, logs.Truncated)
			},
		},
		{
			name: "deploy-confirm", width: 100, height: 28,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.pendingAction = &pendingAction{kind: actionDeploy, app: snapshot.Applications[0]}
			},
		},
		{
			name: "help-overlay", width: 100, height: 28,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.openHelp()
			},
		},
		{
			name: "command-palette", width: 100, height: 28,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.openPalette()
			},
		},
		{
			// Wide enough for the preview strip, which this section does not
			// use — the table has to spread into it.
			name: "deployments-wide", width: 160, height: 40,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.section = SectionDeployments
				m.deployments.SetItems(snapshot.RecentDeployments, now)
			},
		},
		{
			// The active-only queue is where the progress bars live: every row
			// is in flight, so every row has one.
			name: "deployments-active", width: 160, height: 40,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.section = SectionDeployments
				m.deployments.SetItems(snapshot.RecentDeployments, now)
				m.deployments.ToggleActiveOnly()
			},
		},
		{
			name: "instances-wide", width: 160, height: 40,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.section = SectionInstances
			},
		},
		{
			name: "diagnostics", width: 100, height: 28,
			setup: func(m *Model) {
				m.apps.SetApplications(snapshot.Applications, snapshot.ActiveDeployments, now)
				m.lastConnection = conn
				m.coolifyVer = conn.Version
				m.section = SectionDiagnostics
				m.refreshDiagnostics()
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			model := New(Options{
				Config:       config.Default(),
				Service:      service,
				InstanceName: "Demo",
				Demo:         true,
				ASCII:        true,
				Theme:        config.ThemeDark,
				Now:          func() time.Time { return now },
				ConfigPath:   "/tmp/cooldeck-test/config.toml",
				LogPath:      "/tmp/cooldeck-test/cooldeck.log",
			})
			model.loading = false
			model.connection = components.ConnectionOnline
			model.lastSuccess = now
			model.lastConnection = conn
			model.coolifyVer = conn.Version
			model.Update(tea.WindowSizeMsg{Width: tc.width, Height: tc.height})
			if tc.setup != nil {
				tc.setup(model)
				model.recomputeLayout()
				model.refreshInstances()
				if model.section == SectionDiagnostics {
					model.refreshDiagnostics()
				}
			}

			got := normalizeGolden(ansi.Strip(model.render()))
			path := filepath.Join("testdata", tc.name+".golden")
			if updateGolden {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				// Always write LF-only files so Windows and Unix CI compare equal.
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("updated %s", path)
				return
			}
			wantBytes, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden %s: %v (run: make test-update-golden)", path, err)
			}
			// Normalize the fixture too: Git on Windows may check out CRLF even
			// when the repo stores LF, which would false-fail every snapshot.
			want := normalizeGolden(string(wantBytes))
			if got != want {
				t.Fatalf("golden mismatch for %s\n--- got (%d bytes) ---\n%s\n--- want (%d bytes) ---\n%s",
					tc.name, len(got), got, len(want), want)
			}
		})
	}
}

func normalizeGolden(s string) string {
	// Windows checkouts may introduce CR; snapshots always compare as LF.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// ASCII mode uses plain spinner frames; still strip any braille just in case.
	s = spinnerDots.ReplaceAllString(s, "*")
	s = orphanCSI.ReplaceAllString(s, "")
	// Host- and locale-dependent fields must not live in committed goldens.
	s = absoluteTime.ReplaceAllString(s, "YYYY-MM-DD HH:MM:SS")
	s = hostPlatform.ReplaceAllString(s, "${1}${2}GOOS/GOARCH")
	s = goVersion.ReplaceAllString(s, "${1}${2}goX.Y.Z")
	// Collapse runs of spaces left where colour codes used to be, but keep
	// intentional column padding (only fix double-space holes from SGR).
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = trailingSpace.ReplaceAllString(line, "")
	}
	// Drop a trailing empty element from a final newline before re-joining so
	// we control exactly one terminating newline.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestNormalizeGoldenStabilizesHostDependentFields(t *testing.T) {
	in := "" +
		"Platform            darwin/arm64\r\n" +
		"GOOS/GOARCH         linux/amd64\r\n" +
		"Go                  go1.26.5\r\n" +
		"Started        3m ago  (2026-08-04 13:57:00)\r\n"
	got := normalizeGolden(in)
	if strings.Contains(got, "\r") {
		t.Fatal("normalizeGolden left CR bytes in output")
	}
	for _, bad := range []string{"darwin", "linux", "arm64", "amd64", "go1.26.5", "13:57:00"} {
		if strings.Contains(got, bad) {
			t.Fatalf("normalizeGolden left host-dependent value %q in:\n%s", bad, got)
		}
	}
	for _, want := range []string{"GOOS/GOARCH", "goX.Y.Z", "YYYY-MM-DD HH:MM:SS"} {
		if !strings.Contains(got, want) {
			t.Fatalf("normalizeGolden missing %q in:\n%s", want, got)
		}
	}
}

func TestNormalizeGoldenMatchesCRLFFixtures(t *testing.T) {
	lf := normalizeGolden("hello\nworld\n")
	crlf := normalizeGolden("hello\r\nworld\r\n")
	if lf != crlf {
		t.Fatalf("CRLF fixture normalized differently:\n%q\nvs\n%q", lf, crlf)
	}
}

// Keep the app import referenced when Connect's return type is inlined away.
var _ app.Connection
